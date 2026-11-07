package proxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/hostnamealias"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

type errTransfer struct {
	transfer *packet.Transfer
}

func (e *errTransfer) Error() string {
	return fmt.Sprintf("transfer to %s:%d", e.transfer.Address, e.transfer.Port)
}

type Context struct {
	ctx           context.Context
	wg            sync.WaitGroup
	settings      ProxySettings
	hostnameAlias *hostnamealias.Lease
	OnPlayerMove  []func()

	addedPacks []*resource.Pack
	handlers   []func() *Handler
}

func New(ctx context.Context, settings ProxySettings) (*Context, error) {
	p := &Context{
		ctx:      ctx,
		settings: settings,
	}
	if handlers, ok := ctx.Value(extraHandlersKey{}).([]func() *Handler); ok {
		p.handlers = append(p.handlers, handlers...)
	}
	return p, nil
}

func (p *Context) AddHandler(handler func() *Handler) {
	p.handlers = append(p.handlers, handler)
}

func (p *Context) Context() context.Context {
	return p.ctx
}

func (p *Context) connect(connectInfo *connectinfo.ConnectInfo, withClient bool) (err error) {
	session := NewSession(p.ctx, p.settings, p.addedPacks, connectInfo, withClient)
	session.hostnameAlias = p.hostnameAlias
	for _, handlerFunc := range p.handlers {
		session.handlers = append(session.handlers, handlerFunc())
	}

	err = session.Run()
	session.handlers.OnSessionEnd(session, &p.wg)

	if err, ok := err.(*errTransfer); ok {
		if connectInfo.IsReplay() {
			return nil
		}
		address := net.JoinHostPort(err.transfer.Address, strconv.Itoa(int(err.transfer.Port)))
		logrus.Infof("transferring to %s", address)
		return p.connect(&connectinfo.ConnectInfo{Value: address, Account: connectInfo.Account}, withClient)
	}
	return err
}

func (p *Context) Run(ctx context.Context, withClient bool) (err error) {
	defer func() {
		messages.SendEvent(&messages.EventSetUIState{
			State: messages.UIStateFinished,
		})
	}()

	if p.settings.ConnectInfo == nil || p.settings.ConnectInfo.Value == "" {
		return fmt.Errorf("no address")
	}
	if _, err := p.settings.ConnectInfo.Parsed(); err != nil {
		return err
	}
	if p.settings.HostnameAlias && withClient && !p.settings.ConnectInfo.IsReplay() {
		parsed, parseErr := p.settings.ConnectInfo.Parsed()
		if parseErr != nil {
			return parseErr
		}
		if parsed.Kind != connectinfo.KindServer {
			return fmt.Errorf("Destination hostname currently requires a direct server source. Switch it off for Realms or online server discovery")
		}
		host, _, splitErr := net.SplitHostPort(p.settings.ListenAddress)
		if splitErr != nil {
			return splitErr
		}
		if host != "" && host != "0.0.0.0" && host != "127.0.0.1" && host != "::" {
			return fmt.Errorf("Destination hostname requires a local IPv4 or dual-stack listen address")
		}
		logrus.Info("Destination hostname: requesting Windows permission for a temporary local redirect")
		p.hostnameAlias, err = hostnamealias.Acquire(ctx, parsed.Address)
		if err != nil {
			return err
		}
		// The same alias stays valid throughout transfers between proxy sessions.
		defer func() {
			if closeErr := p.hostnameAlias.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("could not remove local hostname alias: %w", closeErr))
			}
			p.hostnameAlias = nil
		}()
	}

	if !p.settings.ConnectInfo.IsReplay() && p.settings.ConnectInfo.Account == nil {
		if !auth.Auth.LoggedIn() {
			err := auth.Auth.Login(ctx, nil, "")
			if err != nil {
				return err
			}
		}
		p.settings.ConnectInfo.Account = auth.Auth.Account()
	}

	if p.settings.Capture && !p.settings.ConnectInfo.IsReplay() {
		p.AddHandler(NewPacketCapturer)
	}
	p.addedPacks, err = loadForcedPacks()
	if err != nil {
		return err
	}

	err = p.connect(p.settings.ConnectInfo, withClient)
	p.wg.Wait()
	return err
}

func loadForcedPacks() ([]*resource.Pack, error) {
	var packs []*resource.Pack
	if _, err := os.Stat("forcedpacks"); err == nil {
		if err = filepath.WalkDir("forcedpacks/", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			switch ext {
			case ".mcpack", ".zip":
				pack, err := resource.ReadPath(path)
				if err != nil {
					return err
				}
				packs = append(packs, pack)
				logrus.Infof("Added %s to the forced packs", pack.Name())
			default:
				logrus.Warnf("Unrecognized file %s in forcedpacks", path)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return packs, nil
}
