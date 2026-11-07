package desktop

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/internal/sourceprobe"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/franchise/gatherings"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling/messaging"
	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

var sourceProbeSlots = make(chan struct{}, 4)

func (a *App) InspectSource(target string, connection bool) (sourceprobe.Result, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	select {
	case sourceProbeSlots <- struct{}{}:
		defer func() { <-sourceProbeSlots }()
	case <-ctx.Done():
		return sourceprobe.Result{}, ctx.Err()
	}
	account := auth.Auth.Account()
	address := strings.TrimSpace(target)
	var assignment *gatherings.ExperienceAssignment
	if strings.HasPrefix(target, "source:") {
		a.mu.Lock()
		cached := a.sources[target]
		if cached == nil {
			a.mu.Unlock()
			return sourceprobe.Result{}, errors.New("Source expired. Refresh the source list")
		}
		copy := *cached
		a.mu.Unlock()
		copy.Account = account
		if account == nil {
			return sourceprobe.Result{}, errors.New("Sign in to inspect this source")
		}
		var resolved string
		var err error
		if strings.HasPrefix(copy.Value, "experience:") {
			assignment, err = copy.ResolveExperience(ctx)
			if err == nil {
				resolved, err = assignment.Address()
			}
		} else {
			resolved, err = copy.Address(ctx)
		}
		if err != nil {
			return sourceprobe.Result{}, err
		}
		address = resolved
	}
	jsonRPC := strings.HasPrefix(address, "nethernet-jsonrpc:")
	if jsonRPC || strings.HasPrefix(address, "nethernet-signaling:") {
		_, id, _ := strings.Cut(address, ":")
		if jsonRPC {
			parsed, err := uuid.Parse(id)
			if err != nil || parsed == uuid.Nil {
				return sourceprobe.Result{}, errors.New("Invalid NetherNet messaging ID")
			}
			id = parsed.String()
		}
		r := sourceprobe.Result{Endpoint: address, Discovery: "session", NetworkProtocol: "NetherNet_JsonRpc", Name: "Experience session", Transport: "not-tested", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
		if !jsonRPC {
			r.NetworkProtocol = "NetherNet"
			r.Name = "Realm session"
		}
		if assignment != nil {
			r.Name = assignment.DestinationInfo.ExperienceName
			r.Level = assignment.DestinationInfo.WorldName
		}
		if connection {
			r.Transport = "connected"
			if err := probeSignaled(ctx, account, id, jsonRPC); err != nil {
				r.Transport = "failed"
				r.TransportError = err.Error()
			} else {
				r.Reachable = true
			}
		}
		if auth.Auth.Account() != account {
			return sourceprobe.Result{}, errors.New("Account changed during the check. Check the source again")
		}
		return r, nil
	}
	if strings.HasPrefix(address, "nethernet:") {
		return sourceprobe.Result{}, errors.New("Xbox worlds advertise session metadata instead of HTTP status. Their authenticated signaling is negotiated when the download starts")
	}
	var dial func(context.Context, string, string) error
	if connection {
		dial = func(ctx context.Context, endpointAddress, networkID string) error {
			d, err := sourceDialer(ctx, account)
			if err != nil {
				return err
			}
			client := (endpoint.ClientConfig{NetworkID: networkID, ClientInfo: endpoint.ClientInfo{Version: protocol.CurrentVersion, Protocol: protocol.CurrentProtocol, Platform: 7}}).New()
			conn, err := d.DialContext(ctx, endpointAddress, client)
			if err != nil {
				return err
			}
			return closeProbeConnection(conn)
		}
	}
	result := sourceprobe.Inspect(ctx, address, dial)
	if auth.Auth.Account() != account {
		return sourceprobe.Result{}, errors.New("Account changed during the check. Check the source again")
	}
	return result, nil
}

func closeProbeConnection(c net.Conn) error { _ = c.Close(); return nil }

func sourceDialer(ctx context.Context, account *auth.Account) (nethernet.Dialer, error) {
	var d nethernet.Dialer
	if account == nil {
		return d, nil
	}
	env, err := account.AuthService(ctx)
	if err != nil {
		return d, err
	}
	src, err := account.TokenSource(ctx)
	if err != nil {
		return d, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return d, err
	}
	token, err := env.MultiplayerToken(ctx, src, &key.PublicKey)
	if err != nil {
		return d, err
	}
	d.Identity = &nethernet.Identity{PrivateKey: key, Token: token, Domain: env.Issuer.JoinPath().String()}
	return d, nil
}

func probeMessaging(ctx context.Context, account *auth.Account, id string) error {
	return probeSignaled(ctx, account, id, true)
}
func probeSignaled(ctx context.Context, account *auth.Account, id string, jsonRPC bool) error {
	if account == nil {
		return errors.New("Sign in to test the authenticated NetherNet messaging connection")
	}
	d, err := sourceDialer(ctx, account)
	if err != nil {
		return err
	}
	src, err := account.TokenSource(ctx)
	if err != nil {
		return err
	}
	var sig interface {
		nethernet.Signaling
		io.Closer
	}
	if jsonRPC {
		sig, err = (messaging.Dialer{}).DialContext(ctx, src)
	} else {
		sig, err = (signaling.Dialer{}).DialContext(ctx, src)
	}
	if err != nil {
		return err
	}
	defer sig.Close()
	conn, err := d.DialContext(ctx, id, sig)
	if err != nil {
		return err
	}
	return closeProbeConnection(conn)
}
