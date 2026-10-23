package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/identity"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/p2p"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling"
	"github.com/Daniel-Ric/CraftBox/utils/nethernet/signaling/messaging"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/blobcache"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/pcap2"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/resourcepacks"
	"github.com/df-mc/go-nethernet"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

type Session struct {
	log       *logrus.Entry
	ctx       context.Context
	cancelCtx context.CancelCauseFunc
	commands  map[string]ingameCommand
	settings  ProxySettings


	addedPacks  []*resource.Pack
	withClient  bool
	handlers    Handlers
	connectInfo *connectinfo.ConnectInfo

	packetLogger       *packetLogger
	packetLoggerClient *packetLogger

	signaling           io.Closer
	p2pSession          *p2p.Session
	listener            *minecraft.Listener
	blobCache           *blobcache.Blobcache
	resourcePackHandler *resourcepacks.ResourcePackHandler

	Server minecraft.IConn
	Client minecraft.IConn
	Player Player

	expectDisconnect bool
	dimensionData    *packet.DimensionData
	clientConnecting chan struct{}
	haveClientData   chan struct{}
	clientData       login.ClientData
	clientAddr       net.Addr
	spawned          bool
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


func (s *Session) AddCommand(exec func([]string) bool, cmd protocol.Command) {
	cmd.AliasesOffset = 0xffffffff
	cmd.PermissionLevel = protocol.CommandPermissionLevelAny
	s.commands[cmd.Name] = ingameCommand{exec, cmd}
}


func (s *Session) ClientWritePacket(pk packet.Packet) error {
	if s.Client == nil {
		return nil
	}
	return proxyEndpointError(s.Client.WritePacket(pk), true)
}


func (s *Session) SendMessage(text string) {
	_ = s.ClientWritePacket(&packet.Text{
		TextType: packet.TextTypeSystem,
		Message:  "§aCraftBox §8> §r" + text,
	})
}


func (s *Session) SendPopup(text string) {
	_ = s.ClientWritePacket(&packet.Text{
		TextType: packet.TextTypePopup,
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
	s.expectDisconnect = true
	_ = s.Server.Close()
}


func (s *Session) Disconnect() {
	s.DisconnectClient()
	s.DisconnectServer()
}

func (s *Session) newResourcePackHandler(ctx context.Context) *resourcepacks.ResourcePackHandler {
	rpHandler := resourcepacks.NewResourcePackHandler(ctx, s.addedPacks)
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

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State:      messages.ConnectStateBegin,
		ListenAddr: s.settings.ListenAddress,
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

		if s.expectDisconnect {
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
	s.handlers.GameDataModifier(s, &gameData)

	var wg sync.WaitGroup
	wg.Go(func() {
		err := s.Server.DoSpawnContext(s.ctx)
		if err != nil {
			s.cancelCtx(err)
			return
		}
	})

	if s.Client != nil {
		wg.Go(func() {
			if s.dimensionData != nil {
				s.Client.WritePacket(s.dimensionData)
			}
			err := s.Client.StartGameContext(s.ctx, gameData)
			if err != nil {
				s.cancelCtx(err)
				return
			}
		})
	}

	wg.Wait()

	err = context.Cause(s.ctx)
	if err != nil {
		reason := err.Error()
		s.disconnectReason.Store(&reason)
		if s.expectDisconnect {
			return nil
		}
		return err
	}

	if s.handlers.OnConnect(s) {
		logrus.Info("Disconnecting")
		return nil
	}
	messages.SendEvent(&messages.EventConnectStateUpdate{
		State: messages.ConnectStateDone,
	})

	err = s.runProxyLoops()
	if err != nil {
		reason := err.Error()
		s.disconnectReason.Store(&reason)
		return err
	}

	return nil
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
				cancelCause(err)
			}
		})
	}
	wg.Go(func() {
		err := s.connectServer(ctx, rpHandler)
		if err != nil && !errors.Is(err, context.Canceled) {
			cancelCause(err)
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
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State: messages.ConnectStateServerConnecting,
	})

	address, err := s.connectInfo.Address(s.ctx)
	if err != nil {
		return err
	}

	logrus.Info(locale.Loc("connecting", locale.Strmap{"Address": address}))

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
		ErrorLog:                   slog.Default(),
		PacketFunc: func(header packet.Header, payload []byte, src, dst net.Addr) {
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
			s.Server = conn
			rpHandler.SetServer(conn)
			conn.ResourcePackHandler = rpHandler
			conn.SetPrePlayPacketHandler(s.serverPrePlayHandler)
		},
	}

	var dialErr error
	isNetherNet := strings.HasPrefix(address, "nethernet:")
	if isNetherNet {
		id, err := uuid.Parse(strings.Split(address, ":")[1])
		if err != nil {
			return fmt.Errorf("parsing nethernet address: %s", err)
		}

		p2pClient := p2p.NewClient(xblClient)
		session, err := p2pClient.Join(ctx, id)
		if err != nil {
			return fmt.Errorf("joining session: %s", err)
		}
		s.p2pSession = session
		clientNonce = session.Nonce()

		tokenSource, err := s.connectInfo.Account.TokenSource(ctx)
		if err != nil {
			return fmt.Errorf("creating TokenSource: %s", err)
		}

		var signalingConn interface {
			nethernet.Signaling
			io.Closer
		}
		switch session.Connection().Type {
		case p2p.ConnectionTypeSignalingOverJSONRPC:
			var d messaging.Dialer
			signalingConn, err = d.DialContext(ctx, tokenSource)
		case p2p.ConnectionTypeSignalingOverWebSocket:
			var d signaling.Dialer
			signalingConn, err = d.DialContext(ctx, tokenSource)
		default:
			return fmt.Errorf("unimplemented signaling type %d", session.Connection().Type)
		}
		if err != nil {
			return fmt.Errorf("error dialing messaging conn: %s", err)
		}
		s.signaling = signalingConn

		_, dialErr = dialer.DialContextNetwork(ctx, minecraft.NetherNet{Signaling: signalingConn}, session.Connection().Address())
	} else {
		_, dialErr = dialer.DialContext(ctx, "nethernet", address)
	}
	if dialErr != nil {
		if s.expectDisconnect {
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

	serverName, err := s.connectInfo.Name(ctx)
	if err != nil {
		return err
	}

	listenerKey, err := identity.LoadOrCreate(utils.PathCache("nethernet", "listener-key.pem"))
	if err != nil {
		return fmt.Errorf("load local NetherNet listener identity: %w", err)
	}
	s.listener, err = minecraft.ListenConfig{
		AuthenticationDisabled: true,
		AllowUnknownPackets:    true,
		StatusProvider:         minecraft.NewStatusProvider(fmt.Sprintf("%s Proxy", serverName), "CraftBox"),
		ErrorLog:               slog.Default(),
		PacketFunc: func(header packet.Header, payload []byte, src, dst net.Addr) {
			clientPacketFunc(header, payload, src, dst, time.Now())
		},
		OnClientData: func(c *minecraft.Conn) {
			s.clientData = c.ClientData()
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
			rpHandler.SetClient(c)
			c.ResourcePackHandler = rpHandler
			close(s.clientConnecting)
		},
	}.ListenNetwork(minecraft.DirectNetherNet{
		Log: slog.Default(),
		IssueServerIdentity: func(context.Context) (*nethernet.Identity, error) {
			return nethernet.GenerateServerIdentity(listenerKey, "self")
		},
	}, s.settings.ListenAddress)
	if err != nil {
		return err
	}

	messages.SendEvent(&messages.EventConnectStateUpdate{
		State: messages.ConnectStateListening,
	})
	logrus.Info(locale.Loc("listening_on", locale.Strmap{"Address": s.listener.Addr()}))
	logrus.Info(locale.Loc("help_connect", nil))

	var accepted = false
	go func() {
		<-ctx.Done()
		if !accepted {
			_ = s.listener.Close()
		}
	}()

	_, err = s.listener.AcceptMinecraft()
	if err != nil {
		return err
	}
	accepted = true
	logrus.Info("Client Connected")
	return nil
}

func (s *Session) serverPrePlayHandler(conn *minecraft.Conn, pk packet.Packet, timeReceived time.Time) (handled bool, err error) {
	switch pk := pk.(type) {
	case *packet.BiomeDefinitionList:
		err = s.ClientWritePacket(pk)
		return true, err
	}
	return false, nil
}

func (s *Session) proxyLoop(ctx context.Context, toServer bool) (err error) {
	var c1, c2 minecraft.IConn
	if toServer {
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
			return proxyEndpointError(err, toServer)
		}
		s.lastPacketTime.Store(&timeReceived)

		pkName := reflect.TypeOf(pk).String()

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
			if err := c2.WritePacket(pk); err != nil {
				reason := err.Error()
				s.disconnectReason.Store(&reason)
				return proxyEndpointError(err, !toServer)
			}
		}

		if transfer != nil {
			if c2 != nil {
				if err := c2.Flush(); err != nil {
					return proxyEndpointError(err, !toServer)
				}
			}
			return &errTransfer{transfer: transfer}
		}
	}
}

func (s *Session) packetFunc(header packet.Header, payload []byte, src, dst net.Addr, timeReceived time.Time) {
	defer func() {
		if errRec := recover(); errRec != nil {
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
		s.spawned = true
	}

	s.handlers.PacketRaw(s, header, payload, src, dst, timeReceived)

	pk, ok := DecodePacket(header, payload, s.Server.ShieldID())
	if !ok {
		return
	}

	if s.packetLogger != nil {
		if s.IsClient(src) {
			s.packetLogger.PacketSend(pk, timeReceived)
		} else {
			s.packetLogger.PacketReceive(pk, timeReceived)
		}
	}

	if !s.spawned {
		switch pk := pk.(type) {
		case *packet.DimensionData:
			s.dimensionData = pk
		}

		var err error
		toServer := s.IsClient(src)
		pk, err = s.handlers.PacketCallback(s, pk, toServer, timeReceived, true)
		if err != nil {
			logrus.Error(err)
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

func (s *Session) commandHandlerPacketCB(pk packet.Packet, _ bool, _ time.Time, _ bool) (packet.Packet, error) {
	switch _pk := pk.(type) {
	case *packet.CommandRequest:
		cmd := strings.Split(_pk.CommandLine, " ")
		name := cmd[0][1:]
		if h, ok := s.commands[name]; ok {
			pk = nil
			h.Exec(cmd[1:])
		}
	case *packet.AvailableCommands:
		cmds := make([]protocol.Command, 0, len(s.commands))
		for _, ic := range s.commands {
			cmds = append(cmds, ic.Cmd)
		}
		_pk.Commands = append(_pk.Commands, cmds...)
	}
	return pk, nil
}

func (s *Session) playerPacketCB(pk packet.Packet, _ bool, _ time.Time, _ bool) (packet.Packet, error) {
	if pk, ok := pk.(*packet.PacketViolationWarning); ok {
		logrus.Infof("%+#v\n", pk)
	}

	haveMoved := s.Player.handlePackets(pk)
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
		s.cancelCtx(err)
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
