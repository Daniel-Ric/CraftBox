package minecraft

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/discovery"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

type DirectNetherNet struct {
	Log        *slog.Logger
	HTTPClient *http.Client

	IssueServerIdentity func(context.Context) (*nethernet.Identity, error)

	DiscoveryAddress string

	DisableLANDiscovery bool
}

func directEndpointAddress(address string) string {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return address
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "19132")
	}
	return "http://" + address
}

func (n DirectNetherNet) network() NetherNet {
	return NetherNet{Signaling: (endpoint.ClientConfig{Logger: n.Log, HTTPClient: n.HTTPClient}).New(), Log: n.Log}
}

func (n DirectNetherNet) resolveEndpoint(ctx context.Context, address string) (string, error) {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return address, nil
	}
	httpAddress := directEndpointAddress(address)
	client := (endpoint.ClientConfig{Logger: n.Log, HTTPClient: n.HTTPClient}).New()
	var attempts []error
	for _, candidate := range []string{"https://" + strings.TrimPrefix(httpAddress, "http://"), httpAddress} {
		probe, cancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := client.Status(probe, candidate)
		cancel()
		if err == nil {
			return candidate, nil
		}
		attempts = append(attempts, fmt.Errorf("%s: %w", candidate, err))
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}
	return "", fmt.Errorf("No reachable NetherNet endpoint at %s. HTTPS and HTTP signaling both failed; the server may require another address or transport: %w", address, errors.Join(attempts...))
}

func (n DirectNetherNet) DialContext(ctx context.Context, address string) (net.Conn, error) {
	address, err := n.resolveEndpoint(ctx, address)
	if err != nil {
		return nil, err
	}
	return n.network().DialContext(ctx, address)
}

func (n DirectNetherNet) DialContextIdentity(ctx context.Context, address, token string, key *ecdsa.PrivateKey) (net.Conn, error) {
	address, err := n.resolveEndpoint(ctx, address)
	if err != nil {
		return nil, err
	}
	return n.network().DialContextIdentity(ctx, address, token, key)
}

func (n DirectNetherNet) PingContext(ctx context.Context, address string) ([]byte, error) {
	address, err := n.resolveEndpoint(ctx, address)
	if err != nil {
		return nil, err
	}
	return (endpoint.ClientConfig{Logger: n.Log, HTTPClient: n.HTTPClient}).New().PingContext(ctx, address)
}

func (n DirectNetherNet) Listen(address string) (NetworkListener, error) {
	_, requestedPort, _ := net.SplitHostPort(address)
	var h *endpoint.Handler
	var mux *ice.MultiUDPMuxDefault
	var err error
	for attempt := 0; attempt < 16; attempt++ {
		h, err = (endpoint.HandlerConfig{Logger: n.Log}).Serve(address)
		if err != nil {
			return nil, err
		}
		port := h.Addr().(*net.TCPAddr).Port
		mux, err = ice.NewMultiUDPMuxFromPort(port, ice.UDPMuxFromPortWithLoopback())
		if err == nil {
			break
		}
		_ = h.Close()

		if requestedPort != "0" {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	settings := webrtc.SettingEngine{}
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetICEUDPMux(mux)
	l, err := (nethernet.ListenConfig{AllowAnonymous: true, Log: n.Log, IssueServerIdentity: n.IssueServerIdentity, API: webrtc.NewAPI(webrtc.WithSettingEngine(settings))}).Listen(h)
	if err != nil {
		_ = mux.Close()
		_ = h.Close()
		return nil, err
	}
	combined := &directNetherNetListener{Listener: l, endpoint: h, udpMux: mux, incoming: make(chan net.Conn), closed: make(chan struct{})}
	if err := combined.listenDiscovery(n, address, settings); err != nil {
		_ = combined.Close()
		return nil, err
	}
	go combined.acceptFrom(l)
	if combined.lanListener != nil {
		go combined.acceptFrom(combined.lanListener)
	}
	return combined, nil
}

type directNetherNetListener struct {
	*nethernet.Listener
	endpoint    *endpoint.Handler
	udpMux      *ice.MultiUDPMuxDefault
	lanListener *nethernet.Listener
	discovery   *discovery.Listener
	nonce       string
	incoming    chan net.Conn
	closed      chan struct{}
	once        sync.Once
	closeErr    error
}

func (l *directNetherNetListener) Addr() net.Addr { return l.endpoint.Addr() }
func (l *directNetherNetListener) Close() error {
	l.once.Do(func() {
		close(l.closed)
		var lanErr, discoveryErr error
		if l.lanListener != nil {
			lanErr = l.lanListener.Close()
		}
		if l.discovery != nil {
			discoveryErr = l.discovery.Close()
		}
		l.closeErr = errors.Join(l.Listener.Close(), l.endpoint.Close(), l.udpMux.Close(), lanErr, discoveryErr)
	})
	return l.closeErr
}

func init() {
	RegisterNetwork("nethernet", func(log *slog.Logger) Network { return DirectNetherNet{Log: log} })
}
