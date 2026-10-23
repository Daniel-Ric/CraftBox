package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
	"github.com/sandertv/gophertunnel/minecraft"
)

type Listener interface {
	Accept() (session.Conn, error)

	Disconnect(conn session.Conn, reason string) error
	io.Closer
}

func importPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("invalid PEM block")
	}
	var key *ecdsa.PrivateKey
	switch block.Type {
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		var ok bool
		key, ok = parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("must be *ecdsa.PrivateKey: %T", parsed)
		}
	default:
		return nil, fmt.Errorf("invalid block type: %s", block.Type)
	}
	if key.Curve != elliptic.P384() {
		return nil, fmt.Errorf("private key must use P-384, got %s", key.Curve.Params().Name)
	}
	return key, nil
}

func exportPrivateKey(path string, key *ecdsa.PrivateKey) error {
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	b := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("make parent directories: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close: %w", err)
	}
	return nil
}

func netherNetKey(path string, log *slog.Logger) (*ecdsa.PrivateKey, error) {
	if path == "" {
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}
		log.Warn("Using a temporary private key for the NetherNet listener. Players connecting over plain HTTP may see the TOFU (Trust On First Use) prompt every time the server restarts.")
		return key, nil
	}
	key, err := importPrivateKey(path)
	if os.IsNotExist(err) {
		key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}

		if err := exportPrivateKey(path, key); err != nil {
			return nil, fmt.Errorf("export private key: %w", err)
		}
		log.Info("Generated a private key for NetherNet listener.", "path", path)
	} else if err != nil {
		return nil, fmt.Errorf("import key file: %w", err)
	}
	return key, nil
}

func (uc UserConfig) netherNetListenerFunc(conf Config) (Listener, error) {
	nn := uc.Network.NetherNet
	address := nn.Address
	if address == "" {
		address = uc.Network.Address
	}
	key, err := netherNetKey(nn.KeyFile, conf.Log.With("net origin", "nethernet"))
	if err != nil {
		return nil, err
	}
	r, err := parsePortRange(nn.UDPPorts)
	if err != nil {
		return nil, fmt.Errorf("parse UDP port range: %w", err)
	}
	return NetherNetConfig{Address: address, Key: key, Domain: nn.Domain, UDPPorts: r}.Listener(conf)
}

func ListenNetwork(conf Config, network minecraft.Network, address string) (Listener, error) {
	l, err := listenerConfig(conf).ListenNetwork(network, address)
	if err != nil {
		return nil, err
	}
	return listener{Listener: l}, nil
}

type NetherNetConfig struct {
	Address string

	Key *ecdsa.PrivateKey

	Domain string

	HTTPServer *http.Server

	Credentials func(ctx context.Context) (*nethernet.Credentials, error)

	UDPPorts PortRange
}

func parsePortRange(s string) (PortRange, error) {
	if s == "" {
		return PortRange{}, nil
	}
	if !strings.Contains(s, "-") {
		v, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return PortRange{}, fmt.Errorf("parse single port: %w", err)
		}
		return PortRange{Min: uint16(v), Max: uint16(v)}, nil
	}
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return PortRange{}, fmt.Errorf("malformed port range: %s", s)
	}
	minimum, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return PortRange{}, fmt.Errorf("parse minimum port: %w", err)
	}
	maximum, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return PortRange{}, fmt.Errorf("parse maximum port: %w", err)
	}
	if minimum > maximum {
		return PortRange{}, fmt.Errorf("invalid port range: %d-%d", minimum, maximum)
	}
	return PortRange{Min: uint16(minimum), Max: uint16(maximum)}, nil
}

type PortRange struct {
	Min, Max uint16
}

type closeFuncs []func() error

func (c *closeFuncs) deferClose(f func() error) {
	*c = append(*c, f)
}

func (c *closeFuncs) Close() (err error) {
	for _, f := range *c {
		if f == nil {
			continue
		}
		if err2 := f(); err2 != nil {
			err = errors.Join(err, err2)
		}
	}
	return err
}

func (nc NetherNetConfig) Listener(conf Config) (Listener, error) {
	log := conf.Log.With("net origin", "nethernet")
	if nc.Key == nil {
		var err error
		if nc.Key, err = netherNetKey("", log); err != nil {
			return nil, err
		}
	}
	if nc.Domain == "" {
		nc.Domain = "self"
	}

	var deferred closeFuncs
	settingEngine := webrtc.SettingEngine{}
	if ports := nc.UDPPorts; ports.Min != 0 && ports.Min == ports.Max {
		mux, err := ice.NewMultiUDPMuxFromPort(int(ports.Min))
		if err != nil {
			return nil, fmt.Errorf("allocate UDP mux: %w", err)
		}
		deferred.deferClose(mux.Close)
		settingEngine.SetICEUDPMux(mux)
	} else if err := settingEngine.SetEphemeralUDPPortRange(ports.Min, ports.Max); err != nil {
		return nil, fmt.Errorf("configure ephemeral udp port range: %w", err)
	}
	lcfg := nethernet.ListenConfig{
		API:            webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine)),
		Log:            log,
		AllowAnonymous: conf.AuthDisabled,
		IssueServerIdentity: func(ctx context.Context) (*nethernet.Identity, error) {
			return nethernet.GenerateServerIdentity(nc.Key, nc.Domain)
		},
	}

	httpLog := conf.Log.With("net origin", "nethernet-http")
	httpServer := nc.HTTPServer
	if httpServer == nil {
		httpServer = &http.Server{
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			IdleTimeout:       30 * time.Second,
		}
	}

	tcp, err := net.Listen("tcp", nc.Address)
	if err != nil {
		_ = deferred.Close()
		return nil, fmt.Errorf("listen NetherNet HTTP: %w", err)
	}

	var serving bool
	deferred.deferClose(func() error {
		if serving {
			return httpServer.Close()
		}
		return tcp.Close()
	})

	handler := endpoint.HandlerConfig{Logger: httpLog, Credentials: nc.Credentials}.New()
	cfg := listenerConfig(conf)
	l, err := cfg.ListenNetwork(minecraft.NetherNet{
		Signaling:    handler,
		ListenConfig: lcfg,
	}, handler.NetworkID())
	if err != nil {
		_ = deferred.Close()
		return nil, fmt.Errorf("create NetherNet listener: %w", err)
	}

	httpServer.Handler = logHTTPRequests(httpLog, handler)
	serving = true
	go func() {
		err := httpServer.Serve(tcp)
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			conf.Log.Error("NetherNet HTTP listener closed unexpectedly: " + err.Error())
		}
	}()

	conf.Log.Info("NetherNet listener running.", "addr", tcp.Addr())
	return listener{Listener: l, close: deferred.Close}, nil
}

func logHTTPRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("NetherNet HTTP request.", "method", r.Method, "path", r.URL.Path, "raddr", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func listenerConfig(conf Config) minecraft.ListenConfig {
	cfg := minecraft.ListenConfig{
		MaximumPlayers:         conf.MaxPlayers,
		StatusProvider:         conf.StatusProvider,
		AuthenticationDisabled: conf.AuthDisabled,
		ResourcePacks:          conf.Resources,
		TexturePacksRequired:   conf.ResourcesRequired,
		Compression:            conf.Compression,
		Allow:                  conf.Allower.Allow,
	}
	if conf.Log.Enabled(context.Background(), slog.LevelDebug) {
		cfg.ErrorLog = conf.Log.With("net origin", "gophertunnel")
	}
	return cfg
}

type listener struct {
	*minecraft.Listener
	close func() error
}

func (l listener) Accept() (session.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return conn.(session.Conn), err
}

func (l listener) Disconnect(conn session.Conn, reason string) error {
	return l.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}

func (l listener) Close() error {
	err := l.Listener.Close()
	if l.close != nil {
		if closeErr := l.close(); err == nil {
			err = closeErr
		}
	}
	return err
}
