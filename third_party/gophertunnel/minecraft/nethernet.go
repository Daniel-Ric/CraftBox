package minecraft

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/df-mc/go-nethernet"
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
		return nil, errors.New("minecraft: NetherNet.DialContext: Signaling is nil")
	}
	if n.Dialer.Log == nil && n.Log != nil {
		n.Dialer.Log = n.Log
	}
	return n.Dialer.DialContext(ctx, address, n.Signaling)
}

func (n NetherNet) DialContextIdentity(ctx context.Context, address string, token string, privateKey *ecdsa.PrivateKey) (net.Conn, error) {
	if n.Dialer.Identity == nil {
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
	return n.DialContext(ctx, address)
}

func (n NetherNet) PingContext(context.Context, string) ([]byte, error) {
	return nil, errors.New("minecraft: NetherNet.PingContext: not supported")
}

func (n NetherNet) Listen(string) (NetworkListener, error) {
	if n.Signaling == nil {
		return nil, errors.New("minecraft: NetherNet.Listen: Signaling is nil")
	}
	if n.ListenConfig.Log == nil && n.Log != nil {
		n.ListenConfig.Log = n.Log
	}
	return n.ListenConfig.Listen(n.Signaling)
}
