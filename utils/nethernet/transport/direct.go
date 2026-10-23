
package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
)



func DirectAddress(address string) (string, error) {
	if address == "" {
		return "", errors.New("empty server address")
	}
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("parse NetherNet endpoint: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid NetherNet endpoint %q", address)
	}
	port := u.Port()
	if port == "" {
		port = "19132"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid NetherNet port %q", port)
	}
	u.Host = net.JoinHostPort(u.Hostname(), port)
	return u.String(), nil
}


type DirectListener struct {
	*nethernet.Listener
	handler  *endpoint.Handler
	once     sync.Once
	closeErr error
}


func ListenDirect(address string) (*DirectListener, error) {
	h, err := endpoint.Serve(address)
	if err != nil {
		return nil, err
	}


	l, err := (nethernet.ListenConfig{AllowAnonymous: true}).Listen(h)
	if err != nil {
		_ = h.Close()
		return nil, err
	}
	return &DirectListener{Listener: l, handler: h}, nil
}


func (l *DirectListener) Address() string { return "http://" + l.handler.Addr().String() }


func (l *DirectListener) Close() error {
	l.once.Do(func() { l.closeErr = errors.Join(l.Listener.Close(), l.handler.Close()) })
	return l.closeErr
}


func DialDirect(ctx context.Context, address string) (*nethernet.Conn, error) {
	address, err := DirectAddress(address)
	if err != nil {
		return nil, err
	}
	var d nethernet.Dialer
	return d.DialContext(ctx, address, endpoint.NewClient())
}
