package minecraft

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type NetherNet struct {
	Signaling nethernet.Signaling

	Dialer nethernet.Dialer

	ListenConfig nethernet.ListenConfig

	Log *slog.Logger
}

var _ packet.TransportCapabilities = (*nethernet.Conn)(nil)

func (n NetherNet) DialContext(ctx context.Context, address string) (net.Conn, error) {
	if n.Signaling == nil {
		n.Signaling = endpoint.NewClient()
	}
	if n.Dialer.Log == nil && n.Log != nil {
		n.Dialer.Log = n.Log
	}
	return n.Dialer.DialContext(ctx, address, n.Signaling)
}

func (n NetherNet) DialContextIdentity(ctx context.Context, address string, token string, privateKey *ecdsa.PrivateKey) (net.Conn, error) {
	if n.Dialer.Identity == nil {
		if token == "" {

			var err error
			n.Dialer.Identity, err = nethernet.SelfSignedIdentity(privateKey, "https://authorization.franchise.minecraft-services.net/")
			if err != nil {
				return nil, fmt.Errorf("generate self signed identity: %w", err)
			}
		} else {
			env, err := authEnv(ctx)
			if err != nil {
				return nil, fmt.Errorf("request authorization environment: %w", err)
			}
			n.Dialer.Identity = &nethernet.Identity{
				PrivateKey: privateKey,
				Token:      token,

				Domain: env.Issuer.JoinPath().String(),
			}
		}
	}
	return n.DialContext(ctx, address)
}

func (n NetherNet) PingContext(ctx context.Context, address string) ([]byte, error) {
	if n.Signaling == nil {
		n.Signaling = endpoint.NewClient()
	}
	if p, ok := n.Signaling.(pinger); ok {
		return p.PingContext(ctx, address)
	}
	return nil, errors.New("minecraft: NetherNet.PingContext: not supported")
}

var _ pinger = (*endpoint.Client)(nil)

type pinger interface {
	PingContext(ctx context.Context, address string) ([]byte, error)
}

func (n NetherNet) Listen(address string) (_ NetworkListener, err error) {
	var closeFunc func() error
	if n.Signaling == nil {
		server, err := endpoint.Serve(address)
		if err != nil {
			return nil, fmt.Errorf("serve HTTP: %w", err)
		}
		n.Signaling = server
		closeFunc = server.Close
	}
	if n.ListenConfig.Log == nil && n.Log != nil {
		n.ListenConfig.Log = n.Log
	}
	l, err := n.ListenConfig.Listen(n.Signaling)
	if err != nil {
		if closeFunc != nil {
			if err2 := closeFunc(); err2 != nil {
				err = errors.Join(err, fmt.Errorf("close HTTP server: %w", err2))
			}
		}
		return nil, err
	}
	return netherNetListener{NetworkListener: l, close: closeFunc}, nil
}

type netherNetListener struct {
	NetworkListener

	close func() error
}

func (n netherNetListener) Close() (err error) {
	if err2 := n.NetworkListener.Close(); err2 != nil {
		err = errors.Join(err, err2)
	}
	if n.close != nil {
		if err2 := n.close(); err2 != nil {
			err = errors.Join(err, fmt.Errorf("close HTTP server: %w", err2))
		}
	}
	return err
}

func init() {
	RegisterNetwork("nethernet", func(l *slog.Logger) Network {
		return NetherNet{Log: l}
	})
}
