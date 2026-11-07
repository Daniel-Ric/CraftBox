package server

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"iter"
	"maps"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/df-mc/dragonfly/server/internal/blockinternal"
	"github.com/df-mc/dragonfly/server/internal/iteminternal"
	"github.com/df-mc/dragonfly/server/internal/sliceutil"
	_ "github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/text/language"
)

type Server struct {
	conf Config

	once    sync.Once
	started atomic.Pointer[time.Time]

	world, nether, end *world.World

	customBlocks     []protocol.BlockEntry
	customItems      []protocol.ItemEntry
	customDimensions []protocol.DimensionDefinition

	listeners []Listener
	incoming  chan incoming

	pmu sync.RWMutex

	p map[uuid.UUID]*onlinePlayer

	pwg sync.WaitGroup

	wg sync.WaitGroup
}

type incoming struct {
	conf player.Config
	s    *session.Session
	p    *onlinePlayer
	w    *world.World
}

type onlinePlayer struct {
	handle *world.EntityHandle
	xuid   string
	name   string
}

func New() *Server {
	var conf Config
	return conf.New()
}

func (srv *Server) Listen() {
	t := time.Now()
	if !srv.started.CompareAndSwap(nil, &t) {
		panic("start server: already started")
	}

	info, _ := debug.ReadBuildInfo()
	if info == nil {
		info = &debug.BuildInfo{GoVersion: "N/A", Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "N/A"}}}
	}
	revision := ""
	for _, set := range info.Settings {
		if set.Key == "vcs.revision" {
			revision = set.Value
		}
	}

	srv.conf.Log.Info("Dragonfly server started.", "mc-version", protocol.CurrentVersion, "go-version", info.GoVersion, "commit", revision)
	srv.startListening()
	go srv.wait()
}

func (srv *Server) Accept() iter.Seq[*player.Player] {
	return func(yield func(*player.Player) bool) {
		for {
			inc, ok := <-srv.incoming
			if !ok {
				return
			}
			srv.pmu.Lock()
			srv.p[inc.p.handle.UUID()] = inc.p
			srv.pmu.Unlock()

			ret, err := world.Call(context.Background(), inc.w, func(tx *world.Tx) (bool, error) {
				p := tx.AddEntity(inc.p.handle).(*player.Player)
				inc.s.Spawn(p, tx)
				return !yield(p), nil
			})
			if err != nil {
				srv.pmu.Lock()
				delete(srv.p, inc.p.handle.UUID())
				srv.pmu.Unlock()
				srv.pwg.Done()

				_ = inc.p.handle.Close()
				inc.s.Disconnect("join failed")
				inc.s.CloseConnection()
				continue
			}
			if ret {
				return
			}
		}
	}
}

func (srv *Server) World() *world.World {
	return srv.world
}

func (srv *Server) Nether() *world.World {
	return srv.nether
}

func (srv *Server) End() *world.World {
	return srv.end
}

func (srv *Server) MaxPlayerCount() int {
	if srv.conf.MaxPlayers == 0 {
		srv.pmu.RLock()
		defer srv.pmu.RUnlock()
		return len(srv.p) + 1
	}
	return srv.conf.MaxPlayers
}

func (srv *Server) PlayerCount() int {
	srv.pmu.RLock()
	defer srv.pmu.RUnlock()
	return len(srv.p)
}

func (srv *Server) Players(tx *world.Tx) iter.Seq[*player.Player] {
	srv.pmu.RLock()
	handles := make([]*world.EntityHandle, 0, len(srv.p))
	for _, p := range srv.p {
		handles = append(handles, p.handle)
	}
	srv.pmu.RUnlock()

	return func(yield func(*player.Player) bool) {
		for _, handle := range handles {
			if tx != nil {
				if e, ok := handle.Entity(tx); ok {
					if !yield(e.(*player.Player)) {
						break
					}
					continue
				}
			}
			ret, err := player.Call(context.Background(), handle, func(_ *world.Tx, p *player.Player) (bool, error) {
				return !yield(p), nil
			})
			if err != nil {
				continue
			}
			if ret {
				break
			}
		}
	}
}

func (srv *Server) Player(uuid uuid.UUID) (*world.EntityHandle, bool) {
	srv.pmu.RLock()
	defer srv.pmu.RUnlock()
	p, ok := srv.p[uuid]
	if !ok {
		return nil, false
	}
	return p.handle, ok
}

func (srv *Server) PlayerByName(name string) (*world.EntityHandle, bool) {
	if p, ok := sliceutil.SearchValue(slices.Collect(maps.Values(srv.p)), func(p *onlinePlayer) bool {
		return p.name == name
	}); ok {
		return p.handle, true
	}
	return nil, false
}

func (srv *Server) PlayerByXUID(xuid string) (*world.EntityHandle, bool) {
	if p, ok := sliceutil.SearchValue(slices.Collect(maps.Values(srv.p)), func(p *onlinePlayer) bool {
		return p.xuid == xuid
	}); ok {
		return p.handle, true
	}
	return nil, false
}

func (srv *Server) CloseOnProgramEnd() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		if err := srv.Close(); err != nil {
			srv.conf.Log.Error("close server: " + err.Error())
		}
	}()
}

func (srv *Server) Close() error {
	if srv.started.Load() == nil {
		panic("server not yet running")
	}
	srv.once.Do(srv.close)
	return nil
}

func (srv *Server) close() {
	srv.conf.Log.Info("Server closing...")

	srv.conf.Log.Debug("Disconnecting players...")
	for p := range srv.Players(nil) {
		p.Disconnect(chat.MessageServerDisconnect.Resolve(p.Locale()))
	}
	srv.pwg.Wait()

	srv.conf.Log.Debug("Closing player provider...")
	if err := srv.conf.PlayerProvider.Close(); err != nil {
		srv.conf.Log.Error("Close player provider: " + err.Error())
	}

	srv.conf.Log.Debug("Closing worlds...")
	for _, w := range []*world.World{srv.end, srv.nether, srv.world} {
		if err := w.Close(); err != nil {
			srv.conf.Log.Error(fmt.Sprintf("Close dimension %v: ", w.Dimension()) + err.Error())
		}
	}

	srv.conf.Log.Debug("Closing listeners...")
	for _, l := range srv.listeners {
		if err := l.Close(); err != nil {
			srv.conf.Log.Error("Close listener: " + err.Error())
		}
	}
}

func (srv *Server) listen(l Listener) {
	wg := new(sync.WaitGroup)
	ctx, cancel := context.WithCancel(context.Background())
	for {
		c, err := l.Accept()
		if err != nil {

			cancel()

			wg.Wait()
			srv.wg.Done()
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			srv.finaliseConn(ctx, c, l)
		}()
	}
}

func (srv *Server) startListening() {
	srv.makeBlockEntries()
	srv.makeItemComponents()
	srv.makeDimensionData()

	srv.wg.Add(len(srv.listeners))
	for _, l := range srv.listeners {
		go srv.listen(l)
	}
}

func (srv *Server) makeBlockEntries() {
	custom, dataDriven := slices.Collect(maps.Values(srv.conf.Blocks.CustomBlocks())), world.DataDrivenBlocks()
	srv.customBlocks = make([]protocol.BlockEntry, 0, len(dataDriven)+len(custom))
	for _, b := range dataDriven {
		srv.customBlocks = append(srv.customBlocks, protocol.BlockEntry{Name: b.Name, Properties: b.Components})
	}

	for i, b := range custom {
		name, _ := b.EncodeBlock()
		srv.customBlocks = append(srv.customBlocks, protocol.BlockEntry{
			Name:       name,
			Properties: blockinternal.Components(name, b, 10000+int32(i)),
		})
	}
}

func (srv *Server) makeItemComponents() {
	custom := world.CustomItems()
	srv.customItems = make([]protocol.ItemEntry, len(custom))

	for i, it := range custom {
		name, _ := it.EncodeItem()
		rid, _, _ := world.ItemRuntimeID(it)
		_, isCustomBlock := it.(world.CustomBlock)
		var entryVersion int32 = protocol.ItemEntryVersionDataDriven
		if isCustomBlock {
			entryVersion = protocol.ItemEntryVersionNone
		}
		srv.customItems[i] = protocol.ItemEntry{
			Name:           name,
			ComponentBased: !isCustomBlock,
			RuntimeID:      int16(rid),
			Version:        entryVersion,
			Data:           iteminternal.Components(it),
		}
	}
}

func (srv *Server) makeDimensionData() {
	dimensions := world.CustomDimensions()
	srv.customDimensions = make([]protocol.DimensionDefinition, 0, len(dimensions))
	for _, registration := range dimensions {
		r := registration.Dimension.Range()
		srv.customDimensions = append(srv.customDimensions, protocol.DimensionDefinition{
			Name:          registration.Name,
			MinimumY:      int32(r.Min()),
			HeightRange:   int32(r.Max() - r.Min()),
			Generator:     protocol.GeneratorVoid,
			DimensionType: int32(registration.ID),
		})
	}
}

func (srv *Server) wait() {
	srv.wg.Wait()
	srv.conf.Log.Info("Server closed.", "uptime", time.Since(*srv.started.Load()).String())
	close(srv.incoming)
}

func (srv *Server) finaliseConn(ctx context.Context, conn session.Conn, l Listener) {
	id := uuid.MustParse(conn.IdentityData().Identity)
	data := srv.defaultGameData()

	d, w, err := srv.conf.PlayerProvider.Load(id, srv.dimension)
	if err != nil {
		w = srv.world
		d.Position = w.Spawn().Vec3Centre()
		d.GameMode = w.DefaultGameMode()
	}

	data.PlayerPosition = vec64To32(d.Position).Add(mgl32.Vec3{0, 1.62})
	dim, _ := world.DimensionID(w.Dimension())
	data.Dimension = int32(dim)
	data.Yaw, data.Pitch = float32(d.Rotation.Yaw()), float32(d.Rotation.Pitch())

	data.EmoteChatMuted = srv.conf.MuteEmoteChat

	if err := conn.StartGameContext(ctx, data); err != nil {
		_ = l.Disconnect(conn, "Connection timeout.")

		srv.conf.Log.Debug("spawn failed: "+err.Error(), "raddr", conn.RemoteAddr())
		return
	}
	if _, ok := srv.Player(id); ok {
		_ = l.Disconnect(conn, "Already logged in.")
		srv.conf.Log.Debug("spawn failed: already logged in", "raddr", conn.RemoteAddr())
		return
	}
	_ = conn.WritePacket(&packet.ItemRegistry{Items: srv.customItems})
	srv.incoming <- srv.createPlayer(id, conn, d, w)
}

func (srv *Server) defaultGameData() minecraft.GameData {
	gm, _ := world.GameModeID(srv.world.DefaultGameMode())
	if srv.world.DefaultGameMode() == world.GameModeSpectator {
		gm = packet.GameTypeSpectator
	}
	return minecraft.GameData{

		EntityUniqueID:  1,
		EntityRuntimeID: 1,

		WorldName:       srv.conf.Name,
		BaseGameVersion: protocol.CurrentVersion,

		Time:       int64(srv.world.Time()),
		Difficulty: 2,

		PlayerGameMode:    int32(gm),
		PlayerPermissions: packet.PermissionLevelMember,
		PlayerPosition:    vec64To32(srv.world.Spawn().Vec3Centre().Add(mgl64.Vec3{0, 1.62})),

		Items:        srv.itemEntries(),
		CustomBlocks: srv.customBlocks,
		GameRules: []protocol.GameRule{
			{Name: "naturalregeneration", Value: false},
			{Name: "locatorBar", Value: false},
		},

		ServerAuthoritativeInventory: true,
		PlayerMovementSettings: protocol.PlayerMovementSettings{
			ServerAuthoritativeBlockBreaking: true,
		},
		Dimensions: srv.customDimensions,
	}
}

func (srv *Server) dimension(dimension world.Dimension) *world.World {
	switch dimension {
	default:
		return srv.world
	case world.Nether:
		return srv.nether
	case world.End:
		return srv.end
	}
}

func (srv *Server) handleSessionClose(tx *world.Tx, c session.Controllable) {
	srv.pmu.Lock()
	_, ok := srv.p[c.UUID()]
	delete(srv.p, c.UUID())
	srv.pmu.Unlock()
	if !ok {

		return
	}

	if tx != nil {
		if err := srv.conf.PlayerProvider.Save(c.UUID(), c.(*player.Player).Data(), tx.World()); err != nil {
			srv.conf.Log.Error("Save player data: " + err.Error())
		}
	} else {
		srv.conf.Log.Error("Save player data: player's worlds closed before teardown; data not saved", "uuid", c.UUID())
	}
	srv.pwg.Done()
}

func (srv *Server) createPlayer(id uuid.UUID, conn session.Conn, conf player.Config, w *world.World) incoming {
	srv.pwg.Add(1)

	s := session.Config{
		Log:            srv.conf.Log,
		MaxChunkRadius: srv.conf.MaxChunkRadius,
		EmoteChatMuted: srv.conf.MuteEmoteChat,
		JoinMessage:    srv.conf.JoinMessage,
		QuitMessage:    srv.conf.QuitMessage,
		HandleStop:     srv.handleSessionClose,
		BlockRegistry:  w.BlockRegistry(),
	}.New(conn)

	conf.Name = conn.IdentityData().DisplayName
	conf.XUID = conn.IdentityData().XUID
	conf.UUID = id
	conf.Locale, _ = language.Parse(strings.Replace(conn.ClientData().LanguageCode, "_", "-", 1))
	conf.Skin = srv.parseSkin(conn.ClientData())
	conf.Session = s

	handle := world.EntitySpawnOpts{Position: conf.Position, ID: id}.New(player.Type, conf)
	s.SetHandle(handle, conf.Skin)
	return incoming{s: s, w: w, conf: conf, p: &onlinePlayer{name: conf.Name, xuid: conf.XUID, handle: handle}}
}

func (srv *Server) createWorld(dim world.Dimension, nether, end **world.World) *world.World {
	logger := srv.conf.Log.With("dimension", strings.ToLower(fmt.Sprint(dim)))
	logger.Debug("Loading dimension...")

	conf := world.Config{
		Log:                 logger,
		Dim:                 dim,
		Provider:            srv.conf.WorldProvider,
		Generator:           srv.conf.Generator(dim),
		RandomTickSpeed:     srv.conf.RandomTickSpeed,
		ReadOnly:            srv.conf.ReadOnlyWorld,
		SaveInterval:        srv.conf.SaveInterval,
		ChunkUnloadInterval: srv.conf.ChunkUnloadInterval,
		ChunkLoadWorkers:    srv.conf.ChunkLoadWorkers,
		Entities:            srv.conf.Entities,
		Blocks:              srv.conf.Blocks,
		PortalDestination: func(dim world.Dimension) *world.World {
			switch dim {
			case world.Nether:
				return *nether
			case world.End:
				return *end
			default:
				return nil
			}
		},
	}
	w := conf.New()
	logger.Info("Opened dimension.", "name", w.Name())
	return w
}

func (srv *Server) parseSkin(data login.ClientData) skin.Skin {

	skinResourcePatch, _ := base64.StdEncoding.DecodeString(data.SkinResourcePatch)

	playerSkin := skin.New(data.SkinImageWidth, data.SkinImageHeight)
	playerSkin.Persona = data.PersonaSkin
	playerSkin.Pix, _ = base64.StdEncoding.DecodeString(data.SkinData)
	playerSkin.Model, _ = base64.StdEncoding.DecodeString(data.SkinGeometry)
	playerSkin.ModelConfig, _ = skin.DecodeModelConfig(skinResourcePatch)
	playerSkin.FullID = data.SkinID

	playerSkin.Cape = skin.NewCape(data.CapeImageWidth, data.CapeImageHeight)
	playerSkin.Cape.Pix, _ = base64.StdEncoding.DecodeString(data.CapeData)

	for _, animation := range data.AnimatedImageData {
		var t skin.AnimationType
		switch animation.Type {
		case protocol.SkinAnimationHead:
			t = skin.AnimationHead
		case protocol.SkinAnimationBody32x32:
			t = skin.AnimationBody32x32
		case protocol.SkinAnimationBody128x128:
			t = skin.AnimationBody128x128
		}

		anim := skin.NewAnimation(animation.ImageWidth, animation.ImageHeight, animation.AnimationExpression, t)
		anim.FrameCount = int(animation.Frames)
		anim.Pix, _ = base64.StdEncoding.DecodeString(animation.Image)

		playerSkin.Animations = append(playerSkin.Animations, anim)
	}

	return playerSkin
}

func vec64To32(vec3 mgl64.Vec3) mgl32.Vec3 {
	return mgl32.Vec3{float32(vec3[0]), float32(vec3[1]), float32(vec3[2])}
}

func (srv *Server) itemEntries() []protocol.ItemEntry {
	entries := make([]protocol.ItemEntry, 0, len(vanillaItems))

	for name, e := range vanillaItems {
		entries = append(entries, protocol.ItemEntry{
			Name:           name,
			RuntimeID:      int16(e.RuntimeID),
			ComponentBased: e.ComponentBased,
			Version:        e.Version,
			Data:           e.Data,
		})
	}
	entries = append(entries, srv.customItems...)
	return entries
}

var (
	//go:embed world/vanilla_items.nbt
	vanillaItemsData []byte
	vanillaItems     = map[string]struct {
		RuntimeID      int32          `nbt:"runtime_id"`
		ComponentBased bool           `nbt:"component_based"`
		Version        int32          `nbt:"version"`
		Data           map[string]any `nbt:"data,omitempty"`
	}{}
)

func init() {
	_ = nbt.Unmarshal(vanillaItemsData, &vanillaItems)
}
