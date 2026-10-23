package minecraft

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sandertv/gophertunnel/minecraft/internal"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sandertv/gophertunnel/minecraft/service"
	"golang.org/x/oauth2"
)

var errListenerClosed = errors.New("listener closed before connection delivery")

type ListenConfig struct {
	EarlyConnHandler func(*Conn)
	OnClientData     func(*Conn)

	ErrorLog *slog.Logger

	HTTPClient *http.Client

	AuthenticationDisabled bool

	DisablePacketEncryption bool

	MaximumPlayers int

	ListenerGroup *ListenerGroup

	LoginTimeout time.Duration

	AllowUnknownPackets bool

	AllowInvalidPackets bool

	StatusProvider ServerStatusProvider

	AcceptedProtocols []Protocol

	Compression packet.Compression

	CompressionSelector func(proto Protocol) packet.Compression

	CompressionThreshold int

	FlushRate time.Duration

	ResourcePacks []*resource.Pack

	TexturePacksRequired bool

	ForceDisableVibrantVisuals bool

	FetchResourcePacks func(identityData login.IdentityData, clientData login.ClientData, current []*resource.Pack) []*resource.Pack

	ResourcePackDelivery ResourcePackDeliveryConfig

	PacketFunc func(header packet.Header, payload []byte, src, dst net.Addr)

	MaxDecompressedLen int

	Allow func(addr net.Addr, identityData login.IdentityData, clientData login.ClientData) (string, bool)
}

type Listener struct {
	cfg      ListenConfig
	listener NetworkListener

	packs   []*resource.Pack
	packsMu sync.RWMutex

	group *ListenerGroup

	incoming chan *Conn
	close    chan struct{}

	key *ecdsa.PrivateKey

	verifier *oidc.IDTokenVerifier
}

type ListenerGroup struct {
	playerCount atomic.Int32
}

func (g *ListenerGroup) PlayerCount() int {
	return int(g.playerCount.Load())
}

func (g *ListenerGroup) add(max int) bool {
	for {
		current := g.playerCount.Load()
		if max > 0 && int(current) >= max {
			return false
		}
		if g.playerCount.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (g *ListenerGroup) remove() {
	g.playerCount.Add(-1)
}

func (cfg ListenConfig) Listen(network string, address string) (*Listener, error) {
	if cfg.ErrorLog == nil {
		cfg.ErrorLog = slog.New(internal.DiscardHandler{})
	}
	cfg.ErrorLog = cfg.ErrorLog.With("src", "listener")
	n, ok := networkByID(network, cfg.ErrorLog)
	if !ok {
		return nil, fmt.Errorf("listen: no network under id %v", network)
	}
	return cfg.ListenNetwork(n, address)
}

func (cfg ListenConfig) ListenNetwork(network Network, address string) (*Listener, error) {
	if cfg.ErrorLog == nil {
		cfg.ErrorLog = slog.New(internal.DiscardHandler{})
	}
	cfg.ErrorLog = cfg.ErrorLog.With("src", "listener")
	if cfg.StatusProvider == nil {
		cfg.StatusProvider = NewStatusProvider("Minecraft Server", "Gophertunnel")
	}
	if cfg.ListenerGroup == nil {
		cfg.ListenerGroup = new(ListenerGroup)
	}
	if cfg.Compression == nil {
		cfg.Compression = packet.DefaultCompression
	}
	if cfg.CompressionSelector == nil {
		cfg.CompressionSelector = func(Protocol) packet.Compression {
			return cfg.Compression
		}
	}
	if cfg.FlushRate == 0 {
		cfg.FlushRate = time.Second / 20
	}
	if cfg.CompressionThreshold == 0 {
		cfg.CompressionThreshold = 256
	} else if cfg.CompressionThreshold < 0 {
		cfg.CompressionThreshold = 0
	}

	var verifier *oidc.IDTokenVerifier
	if !cfg.AuthenticationDisabled {
		var err error
		ctx := context.Background()
		if cfg.HTTPClient != nil {
			ctx = context.WithValue(ctx, oauth2.HTTPClient, cfg.HTTPClient)
		}
		verifier, err = oidcVerifier(ctx)
		if err != nil {
			return nil, fmt.Errorf("create default OIDC verifier: %w", err)
		}
	}

	netListener, err := network.Listen(address)
	if err != nil {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating ECDSA key: %w", err)
	}
	listener := &Listener{
		cfg:      cfg,
		listener: netListener,
		group:    cfg.ListenerGroup,
		packs:    slices.Clone(cfg.ResourcePacks),
		incoming: make(chan *Conn),
		close:    make(chan struct{}),
		key:      key,
		verifier: verifier,
	}

	go listener.listen()
	return listener, nil
}

func Listen(network, address string) (*Listener, error) {
	var lc ListenConfig
	return lc.Listen(network, address)
}

var (
	authEnvCache   *service.AuthorizationEnvironment
	authEnvCacheMu sync.Mutex
)

func authEnv(ctx context.Context) (*service.AuthorizationEnvironment, error) {
	authEnvCacheMu.Lock()
	defer authEnvCacheMu.Unlock()
	if authEnvCache != nil {
		return authEnvCache, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	discovery, err := service.Discover(ctx, service.ApplicationTypeMinecraftPE, protocol.CurrentVersion)
	if err != nil {
		return nil, fmt.Errorf("discover service endpoints: %w", err)
	}
	e := new(service.AuthorizationEnvironment)
	if err := discovery.Environment(e); err != nil {
		return nil, fmt.Errorf("decode environment for auth: %w", err)
	}
	if client, _ := ctx.Value(oauth2.HTTPClient).(*http.Client); client != nil {
		e.HTTPClient = client
	}
	authEnvCache = e
	return e, nil
}

func oidcVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	e, err := authEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtain environment for authorization: %w", err)
	}

	return e.VerifierContext(ctx)
}

func (listener *Listener) Accept() (net.Conn, error) {
	conn, ok := <-listener.incoming
	if !ok {
		return nil, &net.OpError{Op: "accept", Net: "minecraft", Addr: listener.Addr(), Err: net.ErrClosed}
	}
	return conn, nil
}

func (listener *Listener) Disconnect(conn *Conn, message string) error {
	_ = conn.WritePacket(&packet.Disconnect{
		HideDisconnectionScreen: message == "",
		Message:                 message,
	})
	return conn.close(conn.closeErr(message))
}

func (listener *Listener) AddResourcePack(pack *resource.Pack) {
	listener.packsMu.Lock()
	defer listener.packsMu.Unlock()
	listener.packs = append(listener.packs, pack)
}

func (listener *Listener) RemoveResourcePack(uuid string) {
	listener.packsMu.Lock()
	listener.packs = slices.DeleteFunc(listener.packs, func(pack *resource.Pack) bool {
		return pack.UUID().String() == uuid
	})
	listener.packsMu.Unlock()
}

func (listener *Listener) Addr() net.Addr {
	return listener.listener.Addr()
}

func (listener *Listener) Close() error {
	return listener.listener.Close()
}

func (listener *Listener) PlayerCount() int {
	return listener.group.PlayerCount()
}

func (listener *Listener) updatePongData() {
	var (
		s    = listener.status()
		port uint16
	)
	if a, ok := listener.Addr().(interface {
		AddrPort() netip.AddrPort
	}); ok {
		port = a.AddrPort().Port()
	}

	listener.listener.PongData([]byte(fmt.Sprintf("MCPE;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;",
		s.ServerName, protocol.CurrentProtocol, protocol.CurrentVersion, s.PlayerCount, s.MaxPlayers,
		listener.listener.ID(), s.ServerSubName, gameTypeName(s.GameType), 1, port, port, 0, 0,
	)))
}

func (listener *Listener) listen() {
	listener.updatePongData()
	go func() {
		ticker := time.NewTicker(time.Second * 4)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				listener.updatePongData()
			case <-listener.close:
				return
			}
		}
	}()
	defer func() {
		close(listener.close)
		close(listener.incoming)
		_ = listener.Close()
	}()
	for {
		netConn, err := listener.listener.Accept()
		if err != nil {

			return
		}
		listener.createConn(netConn)
	}
}

func (listener *Listener) createConn(netConn net.Conn) {
	listener.packsMu.RLock()
	packs := slices.Clone(listener.packs)
	listener.packsMu.RUnlock()

	conn := newConn(netConn, listener.key, listener.cfg.ErrorLog, proto{}, listener.cfg.FlushRate, true)
	conn.onClientData = listener.cfg.OnClientData
	conn.disableEncryption = conn.disableEncryption || listener.cfg.DisablePacketEncryption
	conn.acceptedProto = append(slices.Clone(listener.cfg.AcceptedProtocols), proto{})
	conn.compression = listener.cfg.Compression
	conn.compressionSelector = listener.cfg.CompressionSelector
	conn.compressionThreshold = listener.cfg.CompressionThreshold
	conn.maxDecompressedLen = listener.cfg.MaxDecompressedLen
	conn.pool = conn.proto.Packets(true)
	conn.allow = listener.cfg.Allow

	conn.packetFunc = listener.cfg.PacketFunc
	conn.texturePacksRequired = listener.cfg.TexturePacksRequired
	conn.forceDisableVibrantVisuals = listener.cfg.ForceDisableVibrantVisuals
	conn.resourcePacks = packs
	conn.fetchResourcePacks = listener.cfg.FetchResourcePacks
	conn.resourcePackDelivery = listener.cfg.ResourcePackDelivery.normalized()
	conn.gameData.WorldName = listener.status().ServerName
	conn.authEnabled = !listener.cfg.AuthenticationDisabled
	conn.verifier = listener.verifier
	conn.disconnectOnUnknownPacket = !listener.cfg.AllowUnknownPackets
	conn.disconnectOnInvalidPacket = !listener.cfg.AllowInvalidPackets

	if !listener.group.add(listener.cfg.MaximumPlayers) {

		_ = conn.WritePacket(&packet.PlayStatus{Status: packet.PlayStatusLoginFailedServerFull})
		_ = conn.close(conn.closeErr("server full"))
		return
	}
	listener.updatePongData()

	var timer *time.Timer
	if timeout := listener.cfg.LoginTimeout; timeout > 0 {
		timer = time.AfterFunc(timeout, func() {
			if !conn.authenticated.Load() {
				conn.log.Debug(errLoginTimeout.Error(), "timeout", timeout)
				conn.closeTransport(errLoginTimeout)
			}
		})
	}
	if listener.cfg.EarlyConnHandler != nil {
		listener.cfg.EarlyConnHandler(conn)
	}
	go func() {
		listener.handleConn(conn)
		if timer != nil {
			timer.Stop()
		}
	}()
}

var errLoginTimeout = errors.New("login timed out")

func (listener *Listener) status() ServerStatus {
	status := listener.cfg.StatusProvider.ServerStatus(listener.group.PlayerCount(), listener.cfg.MaximumPlayers)
	if status.MaxPlayers == 0 {
		status.MaxPlayers = status.PlayerCount + 1
	}
	return status
}

func (listener *Listener) handleConn(conn *Conn) {
	defer func() {
		_ = conn.Close()
		listener.group.remove()
		listener.updatePongData()
	}()
	for {

		receiveErr := false
		if err := conn.dec.DecodeFunc(func(data []byte) error {
			loggedInBefore := conn.loggedIn
			if err := conn.receive(data); err != nil {
				receiveErr = true
				return err
			}
			if !loggedInBefore && conn.loggedIn {
				select {
				case <-listener.close:

					return errListenerClosed
				case listener.incoming <- conn:

				}
			}
			return nil
		}); err != nil {

			if receiveErr || (!errors.Is(err, net.ErrClosed) && !errors.Is(err, errListenerClosed) && !errors.Is(context.Cause(conn.ctx), errLoginTimeout)) {
				conn.log.Error(err.Error())
			}
			return
		}
	}
}
