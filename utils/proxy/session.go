package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/Daniel-Ric/CraftBox/internal/hostnamealias"
	"github.com/Daniel-Ric/CraftBox/internal/ingamefeedback"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/identity"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/p2p"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/blobcache"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/pcap2"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/resourcepacks"
	"github.com/df-mc/go-nethernet"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

type Session struct {
	panicSnapshot               sync.Once
	log                         *logrus.Entry
	ctx                         context.Context
	cancelCtx                   context.CancelCauseFunc
	commands                    map[string]ingameCommand
	forms                       localFormState
	feedbackPolicy              *ingamefeedback.Policy
	commandCompletions          func(*packet.AvailableCommands, []protocol.Command) []protocol.Command
	commandCompletionsPublished func(*packet.AvailableCommands)
	settings                    ProxySettings

	addedPacks                  []*resource.Pack
	withClient                  bool
	forwardClientInitialisation bool
	handlers                    Handlers
	connectInfo                 *connectinfo.ConnectInfo

	packetLogger       *packetLogger
	packetLoggerClient *packetLogger

	signaling           io.Closer
	p2pSession          *p2p.Session
	listener            *minecraft.Listener
	hostnameAlias       *hostnamealias.Lease
	blobCache           *blobcache.Blobcache
	resourcePackHandler *resourcepacks.ResourcePackHandler

	Server     minecraft.IConn
	Client     minecraft.IConn
	Player     Player
	playerMu   sync.RWMutex
	ServerName string

	expectDisconnect atomic.Bool
	dimensionData    *packet.DimensionData
	clientConnecting chan struct{}
	haveClientData   chan struct{}
	clientData       login.ClientData
	clientAddr       net.Addr
	spawned          atomic.Bool
	spawnPlacement   spawnPlacement
	prePlayMu        sync.Mutex
	prePlayWork      sync.WaitGroup
	relayStarted     bool
	disconnectReason atomic.Pointer[string]
	lastPacketTime   atomic.Pointer[time.Time]
}

func NewSession(ctx context.Context, settings ProxySettings, addedPacks []*resource.Pack, connectInfo *connectinfo.ConnectInfo, withClient bool) *Session {
	sctx, cancelCtx := context.WithCancelCause(ctx)
	return &Session{
		ctx:         sctx,
		cancelCtx:   cancelCtx,
		log:         logrus.StandardLogger().WithContext(ctx),
		settings:    settings,
		addedPacks:  addedPacks,
		withClient:  withClient,
		connectInfo: connectInfo,

		clientConnecting: make(chan struct{}),
		haveClientData:   make(chan struct{}),
		commands:         make(map[string]ingameCommand),
	}
}

func (s *Session) Now() time.Time {
	return *s.lastPacketTime.Load()
}

func (s *Session) CaptureStatusText() string {
	state := "§7off§r"
	if s.settings.Capture {
		state = "§aon§r"
	}
	names := make([]string, 0, len(s.handlers))
	statuses := []string{}
	for _, h := range s.handlers {
		names = append(names, h.Name)
		if h.CaptureStatus != nil {
			statuses = append(statuses, h.CaptureStatus())
		}
	}
	return "§7Packet archive configured: " + state + "\n§7Attached modules: §f" + strings.Join(names, ", ") + "§r\n" + strings.Join(statuses, "\n")
}

func (s *Session) PlayerSnapshot() Player {
	s.playerMu.RLock()
	defer s.playerMu.RUnlock()
	return s.Player
}

func (s *Session) AddCommand(exec func([]string) bool, cmd protocol.Command) {
	cmd.AliasesOffset = 0xffffffff
	cmd.PermissionLevel = protocol.CommandPermissionLevelAny
	s.commands[cmd.Name] = ingameCommand{exec, cmd}
}

func (s *Session) SetCommandCompletions(build func(*packet.AvailableCommands, []protocol.Command) []protocol.Command, published func(*packet.AvailableCommands)) {
	s.commandCompletions = build
	s.commandCompletionsPublished = published
}

func (s *Session) commandDefinitionsSent(pk packet.Packet) {
	if definitions, ok := pk.(*packet.AvailableCommands); ok && s.commandCompletionsPublished != nil {
		s.commandCompletionsPublished(definitions)
	}
}

func (s *Session) AppendLocalCommands(pk *packet.AvailableCommands) {
	names := make([]string, 0, len(s.commands))
	for name := range s.commands {
		names = append(names, name)
	}
	slices.Sort(names)
	cmds := make([]protocol.Command, 0, len(names))
	for _, name := range names {
		cmds = append(cmds, s.commands[name].Cmd)
	}
	if s.commandCompletions != nil {
		cmds = s.commandCompletions(pk, cmds)
	}
	pk.Commands = append(pk.Commands, cmds...)
}

func (s *Session) ClientWritePacket(pk packet.Packet) error {
	if text, ok := pk.(*packet.Text); ok && (text.TextType == packet.TextTypePopup || text.TextType == packet.TextTypeTip) {
		settings := s.interfaceSettings()
		if !settings.Enabled {
			return nil
		}
	}
	return s.clientWritePacket(pk)
}

func (s *Session) clientWritePacket(pk packet.Packet) error {
	if s.Client == nil {
		return nil
	}
	err := s.connectionFailure(proxyEndpointError(s.Client.WritePacket(pk), true), "client")
	if err == nil {
		s.commandDefinitionsSent(pk)
	}
	return err
}

func (s *Session) SendMessage(text string) {
	_ = s.ClientWritePacket(&packet.Text{
		TextType: packet.TextTypeSystem,
		Message:  "§8[§aCraftBox§8] §r" + text + "§r",
	})
}

func (s *Session) SendPopup(text string) {
	_ = s.ClientWritePacket(&packet.Text{
		TextType: packet.TextTypeTip,
		Message:  text,
	})
}

func (s *Session) DisconnectClient() {
	if s.Client == nil {
		return
	}
	_ = s.Client.Close()
}

func (s *Session) DisconnectServer() {
	if s.Server == nil {
		return
	}
	s.expectDisconnect.Store(true)
	_ = s.Server.Close()
}

func (s *Session) Disconnect() {
	s.DisconnectClient()
	s.DisconnectServer()
}

func (s *Session) newResourcePackHandler(ctx context.Context) *resourcepacks.ResourcePackHandler {
	rpHandler := resourcepacks.NewResourcePackHandler(ctx, s.addedPacks)
	if s.hostnameAlias != nil {
		rpHandler.SetHTTPClient(s.hostnameAlias.Client)
	}
	rpHandler.OnError = s.cancelCtx
	s.resourcePackHandler = rpHandler
	rpHandler.OnResourcePacksInfoCB = func() {
		messages.SendEvent(&messages.EventConnectStateUpdate{
			State: messages.ConnectStateReceivingResources,
		})
	}
	rpHandler.OnFinishedPack = func(p *resource.Pack) error {
		return s.handlers.OnFinishedPack(s, p)
	}
	return rpHandler
}

func (s *Session) ServerAddress() string {
	if s.connectInfo == nil {
		return ""
	}
	parsed, err := s.connectInfo.Parsed()
	if err != nil || parsed.Kind != connectinfo.KindServer {
		return ""
	}
	return parsed.Address
}

func (s *Session) ActiveResourcePacks() []*resource.Pack {
	if s.resourcePackHandler != nil {
		return s.resourcePackHandler.ActiveResourcePacks()
	}
	return s.Server.ResourcePacks()
}

func (s *Session) newBlobCache() (*blobcache.Blobcache, error) {
	return blobcache.NewBlobCache(s.log, func(pk packet.Packet) error {
		return s.Server.WritePacket(pk)
	}, func() minecraft.IConn {
		return s.Client
	}, func(pk packet.Packet, timeReceived time.Time, preLogin bool) error {
		_, err := s.handlers.PacketCallback(s, pk, false, timeReceived, preLogin)
		return err
	}, func(blobs []protocol.CacheBlob) {
		s.handlers.OnBlobs(s, blobs)
	}, s.connectInfo.IsReplay())
}

func (s *Session) Run() (err error) {
	defer func() { s.cancelCtx(errors.New("done")); s.closeConnections() }()
	if s.connectInfo == nil {
		return errors.New("no connect info")
	}
	if _, err = s.connectInfo.Parsed(); err != nil {
		return err
	}
	if err = s.resolveServerName(s.ctx); err != nil {
		return err
	}
	if err = s.handlers.SessionStart(s, s.ServerName); err != nil {
		return err
	}

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State:      messages.ConnectStateBegin,
		ListenAddr: s.clientJoinAddress(),
	})

	s.blobCache, err = s.newBlobCache()
	if err != nil {
		return err
	}
	defer s.blobCache.Close()

	if s.settings.Debug || s.settings.ExtraDebug {
		s.packetLogger, err = NewPacketLogger(s.settings.ExtraDebug, false)
		if err != nil {
			return err
		}
		defer s.packetLogger.Close()

		s.packetLoggerClient, err = NewPacketLogger(s.settings.ExtraDebug, true)
		if err != nil {
			return err
		}
		defer s.packetLoggerClient.Close()
	}

	if s.connectInfo.IsReplay() {
		replayName, err := s.connectInfo.Address(s.ctx)
		if err != nil {
			return err
		}
		rpHandler := s.newResourcePackHandler(s.ctx)
		replay, err := pcap2.CreateReplayConnector(s.ctx, utils.PathData(replayName), s.packetFunc, rpHandler)
		if err != nil {
			return err
		}
		s.Server = replay
		err = replay.ReadUntilLogin()
		if err != nil {
			if onlyClosedConnection(err) {
				return nil
			}
			return err
		}
	} else {
		if err = s.connect(); err != nil {
			return err
		}
	}

	if s.ctx.Err() != nil {
		err := context.Cause(s.ctx)
		if errors.Is(err, errCancelConnect) {
			err = nil
		}
		reason := "Disconnected"
		if err != nil {
			reason = err.Error()
		}
		s.disconnectReason.Store(&reason)

		if s.expectDisconnect.Load() {
			return nil
		}
		return err
	}

	disconnect, err := s.handlers.OnServerConnect(s)
	if disconnect {
		err = errCancelConnect
	}
	if err != nil {
		s.cancelCtx(err)
		return err
	}

	gameData := s.Server.GameData()
	if len(gameData.CustomBlocks) > 0 {
		definitions := [][]byte{}
		size := 0
		complete := true
		for _, block := range gameData.CustomBlocks {
			raw, err := behaviourpack.BlockRenderJSON(block)
			if err != nil || string(raw) == "null" {
				continue
			}
			if size+len(raw) > 16<<20 || len(definitions) >= 8192 {
				complete = false
				break
			}
			definitions = append(definitions, raw)
			size += len(raw)
		}
		if complete && len(definitions) > 0 {
			messages.SendEvent(&messages.EventCapturedBlockDefinitions{Packs: s.ActiveResourcePacks(), Definitions: definitions})
		}
	}
	s.handlers.GameDataModifier(s, &gameData)

	err = s.runGame(gameData)
	if err != nil {
		reason := err.Error()
		s.disconnectReason.Store(&reason)
	}
	return err
}

func (s *Session) runGame(gameData minecraft.GameData) error {
	if s.expectDisconnect.Load() {
		return nil
	}
	s.configureSpawn(gameData.EntityRuntimeID)
	if err := s.Server.DoSpawnContext(s.ctx); err != nil {
		if s.expectDisconnect.Load() && onlyClosedConnection(err) {
			return nil
		}
		return s.connectionFailure(err, "server")
	}
	if s.expectDisconnect.Load() {
		return nil
	}
	ready := make(chan struct{})
	var readyOnce sync.Once
	signalReady := func() {
		readyOnce.Do(func() {
			diagnostics.Record("info", "connection", "connection.terrain_ready", "Terrain relay is ready", map[string]any{"minecraftClient": s.Client != nil, "entityRuntimeID": gameData.EntityRuntimeID})
			close(ready)
		})
	}
	var spawn sync.WaitGroup
	connected := func() {
		if s.Client != nil {
			if s.forwardClientInitialisation {
				select {
				case <-s.spawnPlacement.ready:
				case <-s.ctx.Done():
					return
				}
			} else {
				if err := s.Server.WritePacket(&packet.SetLocalPlayerAsInitialised{EntityRuntimeID: gameData.EntityRuntimeID}); err != nil {
					s.cancelCtx(s.connectionFailure(err, "server"))
					return
				}
				if err := s.Server.Flush(); err != nil {
					s.cancelCtx(s.connectionFailure(err, "server"))
					return
				}
			}
		}
		s.completeSpawnPlacement(time.Now())
		if s.handlers.OnConnect(s) {
			s.cancelCtx(errCancelConnect)
			return
		}
		messages.SendEvent(&messages.EventConnectStateUpdate{State: messages.ConnectStateDone})
	}
	if s.Client == nil {
		signalReady()
		connected()
	} else {
		spawn.Go(func() {
			if s.dimensionData != nil {
				if err := s.Client.WritePacket(s.dimensionData); err != nil {
					s.cancelCtx(s.connectionFailure(err, "client"))
					return
				}
			}
			var err error
			if client, ok := s.Client.(interface {
				StartGameContextWithReady(context.Context, minecraft.GameData, func()) error
			}); ok {
				err = client.StartGameContextWithReady(s.ctx, s.spawnGameData(gameData), signalReady)
			} else {
				err = s.Client.StartGameContext(s.ctx, s.spawnGameData(gameData))
				signalReady()
			}
			if err != nil {
				s.cancelCtx(s.connectionFailure(err, "client"))
				return
			}
			connected()
		})
	}
	select {
	case <-ready:
	case <-s.ctx.Done():
		spawn.Wait()
		if errors.Is(context.Cause(s.ctx), errCancelConnect) {
			return nil
		}
		return context.Cause(s.ctx)
	}
	s.startRelay()
	err := s.runProxyLoops()
	spawn.Wait()
	if errors.Is(err, errCancelConnect) {
		return nil
	}
	return err
}

func onlyClosedConnection(err error) bool {
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		causes := wrapped.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !onlyClosedConnection(cause) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return onlyClosedConnection(wrapped.Unwrap())
	default:
		return err == net.ErrClosed
	}
}

func (s *Session) connect() error {
	var wg sync.WaitGroup
	ctx, cancelCause := context.WithCancelCause(s.ctx)
	defer cancelCause(nil)

	rpHandler := s.newResourcePackHandler(ctx)
	if s.withClient {
		wg.Go(func() {
			err := s.connectClient(ctx, rpHandler)
			if err != nil && !errors.Is(err, context.Canceled) {
				cancelCause(s.connectionFailure(err, "client"))
			}
		})
	}
	wg.Go(func() {
		err := s.connectServer(ctx, rpHandler)
		if err != nil && !errors.Is(err, context.Canceled) {
			cancelCause(s.connectionFailure(err, "server"))
		}
	})
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Session) connectServer(ctx context.Context, rpHandler *resourcepacks.ResourcePackHandler) (err error) {
	if s.withClient {
		select {
		case <-s.clientConnecting:
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State: messages.ConnectStateServerConnecting,
	})

	xblClient, err := s.connectInfo.Account.XBLClient(ctx)
	if err != nil {
		return err
	}

	playfabClient, err := s.connectInfo.Account.PlayFabClient(ctx)
	if err != nil {
		return err
	}

	var clientNonce string
	dialer := minecraft.Dialer{
		TokenSource:   s.connectInfo.Account,
		XBLClient:     xblClient,
		PlayFabClient: playfabClient,

		DisconnectOnUnknownPackets: false,
		MaxDecompressedLen:         256 << 20,
		ErrorLog:                   slog.Default(),
		PacketFunc: func(header packet.Header, payload []byte, src, dst net.Addr) {
			s.observeWire(header, payload, src, dst, false, time.Now())
			s.packetFunc(header, payload, src, dst, time.Now())
		},
		EnableClientCache: s.settings.ClientCache,
		GetClientData: func() login.ClientData {
			if s.withClient {
				select {
				case <-s.haveClientData:
				case <-ctx.Done():
				}
			}
			clientData := s.clientData
			clientData.Nonce = clientNonce
			return clientData
		},
		EarlyConnHandler: func(conn *minecraft.Conn) {
			conn.SetPlayerInitialisationDeferred(s.withClient)
			s.Server = conn
			rpHandler.SetServer(conn)
			conn.ResourcePackHandler = rpHandler
			conn.SetPrePlayPacketHandler(s.serverPrePlayHandler)
		},
	}
	if s.hostnameAlias != nil {
		dialer.HTTPClient = s.hostnameAlias.Client
	}

	parsed, err := s.connectInfo.Parsed()
	if err != nil {
		return err
	}
	address, nonce, sig, err := s.resolveServerAddress(ctx, parsed)
	if err != nil {
		return err
	}
	clientNonce = nonce
	s.signaling = sig
	logrus.Info(locale.Loc("connecting", locale.Strmap{"Address": address}))
	var dialErr error
	if sig != nil {
		_, dialErr = dialer.DialContextNetwork(ctx, minecraft.NetherNet{Signaling: sig, Log: slog.Default()}, address)
	} else {
		if s.hostnameAlias != nil {
			_, dialErr = dialer.DialContextNetwork(ctx, minecraft.DirectNetherNet{HTTPClient: s.hostnameAlias.Client, Log: slog.Default()}, address)
		} else {
			_, dialErr = dialer.DialContext(ctx, "nethernet", address)
		}
	}
	if dialErr != nil {
		if s.expectDisconnect.Load() {
			return nil
		}
		return dialErr
	}

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State: messages.ConnectStateEstablished,
	})
	logrus.Debug(locale.Loc("connected", nil))
	return nil
}

func (s *Session) connectClient(ctx context.Context, rpHandler *resourcepacks.ResourcePackHandler) (err error) {
	clientPacketFunc := func(header packet.Header, payload []byte, src, dst net.Addr, timeReceived time.Time) {
		s.observeWire(header, payload, src, dst, true, timeReceived)
		pk, ok := DecodePacket(header, payload, s.Client.ShieldID())
		if !ok {
			return
		}
		drop, err := s.blobPacketsFromClient(pk)
		if err != nil {
			logrus.Error(err)
			return
		}
		_ = drop
		if s.packetLoggerClient != nil {
			if src == s.listener.Addr() {
				err = s.packetLoggerClient.PacketSend(pk, timeReceived)
			} else {
				err = s.packetLoggerClient.PacketReceive(pk, timeReceived)
			}
		}
		if err != nil {
			logrus.Error(err)
			return
		}
	}

	serverName := s.ServerName

	listenerKey, err := identity.LoadOrCreate(utils.PathCache("nethernet", "listener-key.pem"))
	if err != nil {
		return fmt.Errorf("load local NetherNet listener identity: %w", err)
	}
	s.listener, err = minecraft.ListenConfig{
		MaxDecompressedLen:     256 << 20,
		AuthenticationDisabled: true,
		AllowUnknownPackets:    true,
		StatusProvider:         minecraft.NewStatusProvider(fmt.Sprintf("%s Proxy", serverName), "CraftBox"),
		ErrorLog:               slog.Default(),
		PacketFunc: func(header packet.Header, payload []byte, src, dst net.Addr) {
			clientPacketFunc(header, payload, src, dst, time.Now())
		},
		OnClientData: func(c *minecraft.Conn) {
			s.clientData = c.ClientData()
			if s.hostnameAlias != nil {
				logrus.Infof("Destination hostname join: requested %s; Minecraft login reports %s", s.clientJoinAddress(), s.clientData.ServerAddress)
				observed := strings.ToLower(s.clientData.ServerAddress)
				matches := strings.Contains(observed, "://"+s.hostnameAlias.Hostname+":") || strings.HasPrefix(observed, s.hostnameAlias.Hostname+":")
				messages.SendEvent(&messages.EventMinecraftJoinAddress{Expected: s.clientJoinAddress(), Observed: s.clientData.ServerAddress, HostnameMatches: matches})
				if !matches {
					logrus.Warn("Minecraft joined using a different address. Reconnect using CraftBox's destination hostname link, not the saved localhost or LAN entry")
				}
			}
			ident := c.IdentityData()
			s.handlers.PlayerDataModifier(s, &ident, &s.clientData)
			close(s.haveClientData)
		},
		EarlyConnHandler: func(c *minecraft.Conn) {
			if s.Client != nil {
				s.listener.Disconnect(c, "You are Already connected!")
				return
			}
			s.Client = c
			c.SetPlayerInitialisationForwarded(true)
			s.forwardClientInitialisation = true
			rpHandler.SetClient(c)
			c.ResourcePackHandler = rpHandler
			close(s.clientConnecting)
		},
	}.ListenNetwork(minecraft.DirectNetherNet{
		Log:                 slog.Default(),
		DisableLANDiscovery: s.hostnameAlias != nil || !s.settings.LANDiscovery,
		IssueServerIdentity: func(context.Context) (*nethernet.Identity, error) {
			return nethernet.GenerateServerIdentity(listenerKey, "self")
		},
	}, s.settings.ListenAddress)
	if err != nil {
		return err
	}

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State:      messages.ConnectStateListening,
		ListenAddr: s.clientJoinAddress(),
	})
	logrus.Info(locale.Loc("listening_on", locale.Strmap{"Address": s.listener.Addr()}))
	logrus.Info(locale.Loc("help_connect", nil))

	var accepted atomic.Bool
	go func() {
		<-ctx.Done()
		if !accepted.Load() {
			_ = s.listener.Close()
		}
	}()

	_, err = s.listener.AcceptMinecraft()
	if err != nil {
		return err
	}
	accepted.Store(true)
	logrus.Info("Client Connected")
	return nil
}

func (s *Session) serverPrePlayHandler(conn *minecraft.Conn, pk packet.Packet, timeReceived time.Time) (handled bool, err error) {
	logrus.WithField("part", "Handshake").Debugf("received early server packet %T (ID %d)", pk, pk.ID())
	switch pk := pk.(type) {
	case *packet.BiomeDefinitionList:
		err = s.ClientWritePacket(pk)
		return true, err
	}
	return false, nil
}

func (s *Session) proxyLoop(ctx context.Context, toServer bool) (err error) {
	var c1, c2 minecraft.IConn
	readLeg, writeLeg := "server", "client"
	if toServer {
		readLeg, writeLeg = "client", "server"
		c1 = s.Client
		c2 = s.Server
	} else {
		c1 = s.Server
		c2 = s.Client
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		pk, timeReceived, err := c1.ReadPacketWithTime()
		if err != nil {
			return s.connectionFailure(proxyEndpointError(err, toServer), readLeg)
		}
		s.lastPacketTime.Store(&timeReceived)

		pkName := reflect.TypeOf(pk).String()
		pk, err = s.handlers.Intercept(s, pk, toServer)
		if err != nil {
			return err
		}
		if pk == nil {
			continue
		}

		var forward = pk
		var process = true

		if !toServer {
			forward, process, err = s.blobPacketsFromServer(pk, timeReceived, false)
			if err != nil {
				return err
			}
		} else {
			if pk.ID() == packet.IDClientCacheBlobStatus {
				forward = nil
			}
		}

		pk, err = s.commandHandlerPacketCB(pk, toServer, timeReceived, false)
		if err != nil {
			return err
		}

		pk, err = s.playerPacketCB(pk, toServer, timeReceived, false)
		if err != nil {
			return err
		}

		if process {
			pk, err = s.handlers.PacketCallback(s, pk, toServer, timeReceived, false)
			if err != nil {
				return err
			}
			if pk == nil {
				logrus.Tracef("Dropped Packet: %s", pkName)
				continue
			}
		}

		if forward == nil {
			continue
		}

		var transfer *packet.Transfer
		switch _pk := pk.(type) {
		case *packet.Transfer:
			transfer = _pk
			if s.Client != nil {
				local, err := s.proxyTransfer(_pk)
				if err != nil {
					return err
				}
				pk = local
			}
		case *packet.InventoryContent:
			if _pk.StorageItem.Stack.NetworkID == -1 {
				_pk.StorageItem.Stack.NetworkID = 0
			}
		case *packet.InventorySlot:
			if item, ok := _pk.StorageItem.Value(); ok {
				if item.Stack.NetworkID == -1 {
					item.Stack.NetworkID = 0
					_pk.StorageItem = protocol.Option(item)
				}
			}
		}

		if pk != nil && c2 != nil {
			initialises := false
			if initialised, ok := pk.(*packet.SetLocalPlayerAsInitialised); ok && toServer && s.Client != nil {
				p := &s.spawnPlacement
				p.mu.Lock()
				duplicate := !p.readyAt.IsZero()
				valid := !p.known || initialised.EntityRuntimeID == p.runtimeID
				p.mu.Unlock()
				if !valid {
					return fmt.Errorf("Minecraft spawn initialisation runtime ID mismatch: %d", initialised.EntityRuntimeID)
				}
				if duplicate {
					continue
				}
				initialises = true
			}
			if err := c2.WritePacket(pk); err != nil {
				reason := err.Error()
				s.disconnectReason.Store(&reason)
				return s.connectionFailure(proxyEndpointError(err, !toServer), writeLeg)
			}
			if !toServer {
				s.commandDefinitionsSent(pk)
			}
			if initialises {
				if err := c2.Flush(); err != nil {
					return s.connectionFailure(proxyEndpointError(err, false), writeLeg)
				}
				s.completeSpawnPlacement(timeReceived)
			}
		}

		if transfer != nil {
			if c2 != nil {
				if err := c2.Flush(); err != nil {
					return s.connectionFailure(proxyEndpointError(err, !toServer), writeLeg)
				}
			}
			return &errTransfer{transfer: transfer}
		}
	}
}

func (s *Session) packetFunc(header packet.Header, payload []byte, src, dst net.Addr, timeReceived time.Time) {
	defer func() {
		if errRec := recover(); errRec != nil {
			if m := diagnostics.Current(); m != nil {
				s.panicSnapshot.Do(func() { m.Snapshot(fmt.Sprintf("Packet handler panic (packet ID %d): %v", header.PacketID, errRec)) })
			}
			switch err := errRec.(type) {
			case error:
				utils.ErrorHandler(err)
			case string:
				utils.ErrorHandler(errors.New(err))
			default:
				panic(errRec)
			}
		}
	}()

	if header.PacketID == packet.IDRequestNetworkSettings {
		s.clientAddr = src
	}
	if header.PacketID == packet.IDSetLocalPlayerAsInitialised {
		s.spawned.Store(true)
	}

	s.handlers.PacketRaw(s, header, payload, src, dst, timeReceived)

	pk, ok := DecodePacket(header, payload, s.Server.ShieldID())
	if !ok {
		return
	}
	if src != nil && s.Server.LocalAddr() != nil && src.String() != s.Server.LocalAddr().String() {
		s.observeSpawnPlacement(pk)
	}

	if s.packetLogger != nil {
		if s.IsClient(src) {
			s.packetLogger.PacketSend(pk, timeReceived)
		} else {
			s.packetLogger.PacketReceive(pk, timeReceived)
		}
	}

	if s.beginPrePlay() {
		defer s.prePlayWork.Done()
		switch pk := pk.(type) {
		case *packet.DimensionData:
			s.dimensionData = pk
		}

		var err error
		toServer := s.IsClient(src)
		pk, err = s.handlers.PacketCallback(s, pk, toServer, timeReceived, true)
		if err != nil {
			logrus.Error(err)
			s.cancelCtx(fmt.Errorf("process initial world data: %w", err))
			return
		}
		if pk == nil {
			return
		}

		if !toServer {
			forward, _, err := s.blobPacketsFromServer(pk, timeReceived, true)
			if err != nil {
				logrus.Error(err)
			}
			_ = forward
		}
	}
}

func (s *Session) IsClient(addr net.Addr) bool {
	return s.clientAddr.String() == addr.String()
}

func (s *Session) blobPacketsFromServer(pk packet.Packet, timeReceived time.Time, preLogin bool) (forward packet.Packet, process bool, err error) {
	forward = pk
	process = true
	switch pk := pk.(type) {
	case *packet.LevelChunk:
		if pk.CacheEnabled {
			process = false
			forward, err = s.blobCache.HandleLevelChunk(pk, timeReceived, preLogin)
		}

	case *packet.SubChunk:
		if pk.CacheEnabled {
			process = false
			forward, err = s.blobCache.HandleSubChunk(pk, timeReceived, preLogin)
		}

	case *packet.ClientCacheMissResponse:
		forward = nil
		process = false
		if !preLogin {
			err = s.blobCache.HandleClientCacheMissResponse(pk, timeReceived, preLogin)
		}
	}
	return
}

func (s *Session) blobPacketsFromClient(pk packet.Packet) (forward bool, err error) {
	switch pk := pk.(type) {
	case *packet.ClientCacheBlobStatus:
		forward = false
		err = s.blobCache.HandleClientCacheBlobStatus(pk)
	default:
		forward = true
	}
	return
}

func (s *Session) commandHandlerPacketCB(pk packet.Packet, toServer bool, _ time.Time, _ bool) (packet.Packet, error) {
	pk, err := s.formPacketCB(pk, toServer)
	if err != nil || pk == nil {
		return pk, err
	}
	switch _pk := pk.(type) {
	case *packet.CommandRequest:
		if !toServer {
			return pk, nil
		}
		cmd := strings.Fields(_pk.CommandLine)
		if len(cmd) == 0 || !strings.HasPrefix(cmd[0], "/") {
			return pk, nil
		}
		name := strings.ToLower(strings.TrimPrefix(cmd[0], "/"))
		if h, ok := s.commands[name]; ok {
			pk = nil
			h.Exec(cmd[1:])
		}
	case *packet.AvailableCommands:
		if toServer {
			return pk, nil
		}
		s.AppendLocalCommands(_pk)
	}
	return pk, nil
}

func (s *Session) playerPacketCB(pk packet.Packet, _ bool, _ time.Time, _ bool) (packet.Packet, error) {
	if pk, ok := pk.(*packet.PacketViolationWarning); ok {
		logrus.Infof("%+#v\n", pk)
	}

	s.playerMu.Lock()
	haveMoved := s.Player.handlePackets(pk)
	s.playerMu.Unlock()
	if haveMoved {
		s.handlers.OnPlayerMove(s)
	}
	return pk, nil
}

func (s *Session) runProxyLoops() error {
	var wg sync.WaitGroup
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-s.ctx.Done()
		_ = s.Server.Close()
		if s.Client != nil {
			var transfer *errTransfer
			if errors.As(context.Cause(s.ctx), &transfer) {

				timer := time.NewTimer(2 * time.Second)
				select {
				case <-s.Client.Context().Done():
				case <-timer.C:
				}
				timer.Stop()
				_ = s.Client.Close()
			} else {
				_ = s.Client.Close()
			}
		}
	}()
	forward := func(toServer bool) {
		defer wg.Done()
		err := s.proxyLoop(s.ctx, toServer)
		if err == nil {
			err = net.ErrClosed
		}
		report := s.ctx.Err() == nil
		s.cancelCtx(err)
		if report {
			var transfer *errTransfer
			if errors.As(err, &transfer) {
				diagnostics.Record("info", "connection", "connection.transfer", err.Error(), map[string]any{"readingFromClient": toServer})
			} else {
				diagnostics.Record("warning", "connection", "connection.ended", err.Error(), map[string]any{"readingFromClient": toServer})
			}
			if transfer == nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, context.Canceled) {
				if m := diagnostics.Current(); m != nil {
					m.Snapshot("Unexpected connection failure: " + err.Error())
				}
			}
		}
	}
	wg.Add(1)
	go forward(false)
	if s.Client != nil {
		wg.Add(1)
		go forward(true)
	}
	wg.Wait()
	<-stopped
	err := context.Cause(s.ctx)
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func proxyEndpointError(err error, client bool) error {
	if err == nil || !client || errors.Is(err, net.ErrClosed) {
		return err
	}
	cause := err
	for errors.Unwrap(cause) != nil {
		cause = errors.Unwrap(cause)
	}
	switch cause.Error() {
	case `nethernet: data channel "ReliableDataChannel" closed by remote peer`,
		`nethernet: data channel "UnreliableDataChannel" closed by remote peer`,
		`nethernet: SCTP transport closed`:
		logrus.Info("Minecraft client closed the connection")
		return errors.Join(err, net.ErrClosed)
	}
	return err
}

func (s *Session) proxyTransfer(original *packet.Transfer) (*packet.Transfer, error) {
	address, ok := s.listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("proxy transfer: no HTTP signaling address")
	}
	ip := address.IP.String()
	if local, ok := s.Client.LocalAddr().(*nethernet.Addr); ok && local.SelectedCandidate != nil {
		ip = local.SelectedCandidate.Address
	}
	if parsed := net.ParseIP(ip); parsed == nil || parsed.IsUnspecified() {
		return nil, fmt.Errorf("proxy transfer: no reachable local address")
	}
	result := *original
	result.Address = ip
	if s.hostnameAlias != nil {
		result.Address = s.hostnameAlias.Hostname
	}
	result.Port = uint16(address.Port)
	return &result, nil
}

func (s *Session) closeConnections() {
	if s.p2pSession != nil {
		_ = s.p2pSession.Close()
	}
	if s.signaling != nil {
		_ = s.signaling.Close()
	}
	if s.Server != nil {
		_ = s.Server.Close()
	}
	if s.listener != nil {
		if client, ok := s.Client.(*minecraft.Conn); ok {
			reason := "Disconnected"
			if saved := s.disconnectReason.Load(); saved != nil {
				reason = *saved
			}
			_ = s.listener.Disconnect(client, reason)
		}
	}
	if s.Client != nil {
		_ = s.Client.Close()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *Session) clientJoinAddress() string {
	address := s.settings.ListenAddress
	if s.listener != nil {
		address = s.listener.Addr().String()
	}
	if s.hostnameAlias != nil {
		_, port, err := net.SplitHostPort(address)
		if err == nil {
			return "http://" + net.JoinHostPort(s.hostnameAlias.Hostname, port)
		}
	}
	return address
}

func (s *Session) beginPrePlay() bool {
	s.prePlayMu.Lock()
	defer s.prePlayMu.Unlock()
	if s.relayStarted || s.spawned.Load() {
		return false
	}
	s.prePlayWork.Add(1)
	return true
}
func (s *Session) startRelay() {
	s.prePlayMu.Lock()
	s.relayStarted = true
	s.prePlayMu.Unlock()
	s.prePlayWork.Wait()
}
