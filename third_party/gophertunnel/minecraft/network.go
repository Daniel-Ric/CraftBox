package minecraft

import (
	"context"
	"crypto/ecdsa"
	"log/slog"
	"net"
)

type Network interface {
	DialContext(ctx context.Context, address string) (net.Conn, error)

	PingContext(ctx context.Context, address string) (response []byte, err error)

	Listen(address string) (NetworkListener, error)
}

type identityDialer interface {
	DialContextIdentity(ctx context.Context, address string, token string, privateKey *ecdsa.PrivateKey) (net.Conn, error)
}

type NetworkListener interface {
	net.Listener

	ID() int64

	PongData(data []byte)
}

var networks = map[string]func(l *slog.Logger) Network{}

func RegisterNetwork(id string, n func(l *slog.Logger) Network) {
	networks[id] = n
}

func networkByID(id string, l *slog.Logger) (Network, bool) {
	n, ok := networks[id]
	if ok {
		return n(l), true
	}
	return nil, false
}
