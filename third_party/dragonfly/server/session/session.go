package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/debug"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/player/hud"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Session struct {
	conf           Config
	once, connOnce sync.Once

	ent      *world.EntityHandle
	conn     Conn
	handlers map[uint32]packetHandler
	packets  chan packet.Packet

	currentScoreboard atomic.Pointer[string]
	currentLines      atomic.Pointer[[]string]

	chunkLoader                 *world.Loader
	chunkRadius, maxChunkRadius int32

	emoteChatMuted bool

	teleportPos atomic.Pointer[mgl64.Vec3]

	entityMutex sync.RWMutex

	currentEntityRuntimeID uint64

	entityRuntimeIDs map[*world.EntityHandle]uint64
	entities         map[uint64]*world.EntityHandle
	hiddenEntities   map[uuid.UUID]struct{}

	heldSlot                     *uint32
	inv, offHand, enderChest, ui *inventory.Inventory
	armour                       *inventory.Armour

	joinSkin skin.Skin

	breakingPos cube.Pos

	inTransaction, containerOpened atomic.Bool
	openedWindowID                 atomic.Uint32
	openedContainerID              atomic.Uint32
	openedWindow                   atomic.Pointer[inventory.Inventory]
	openedPos                      atomic.Pointer[cube.Pos]
	swingingArm                    atomic.Bool
	changingSlot                   atomic.Bool
	changingDimension              atomic.Bool
	moving                         bool

	lastChunkPos world.ChunkPos

	recipes map[uint32]recipe.Recipe

	blobMu                sync.Mutex
	blobs                 map[uint64][]byte
	openChunkTransactions []map[uint64]struct{}
	invOpened             bool

	hudMu      sync.RWMutex
	hudUpdates map[hud.Element]bool
	hiddenHud  map[hud.Element]struct{}

	debugShapesMu     sync.RWMutex
	debugShapes       map[int]debug.Shape
	debugShapeUpdates []debugShapeUpdate

	viewLayer *world.ViewLayer

	inputLocksMu sync.RWMutex
	inputLocks   uint32

	closeBackground chan struct{}

	br world.BlockRegistry
}

type debugShapeUpdate struct {
	id    int
	shape debug.Shape
}

type Conn interface {
	io.Closer

	IdentityData() login.IdentityData

	ClientData() login.ClientData

	ClientCacheEnabled() bool

	ChunkRadius() int

	Latency() time.Duration

	Flush() error

	RemoteAddr() net.Addr

	ReadPacket() (pk packet.Packet, err error)

	WritePacket(pk packet.Packet) error

	StartGameContext(ctx context.Context, data minecraft.GameData) error
}

var Nop = &Session{conf: Config{Log: slog.New(slog.DiscardHandler)}}

const selfEntityRuntimeID = 1

var errSelfRuntimeID = errors.New("invalid entity runtime ID: runtime ID for self must always be 1")

type Config struct {
	Log *slog.Logger

	MaxChunkRadius int

	EmoteChatMuted bool

	JoinMessage, QuitMessage chat.Translation

	HandleStop func(*world.Tx, Controllable)

	BlockRegistry world.BlockRegistry
}

func (conf Config) New(conn Conn) *Session {
	r := conn.ChunkRadius()
	if r > conf.MaxChunkRadius {
		r = conf.MaxChunkRadius
		_ = conn.WritePacket(&packet.ChunkRadiusUpdated{ChunkRadius: int32(r)})
	}
	if conf.Log == nil {
		conf.Log = slog.Default()
	}
	conf.Log = conf.Log.With("name", conn.IdentityData().DisplayName, "uuid", conn.IdentityData().Identity, "raddr", conn.RemoteAddr().String())

	s := &Session{}
	*s = Session{
		openChunkTransactions:  make([]map[uint64]struct{}, 0, 8),
		closeBackground:        make(chan struct{}),
		handlers:               map[uint32]packetHandler{},
		packets:                make(chan packet.Packet, 256),
		entityRuntimeIDs:       map[*world.EntityHandle]uint64{},
		entities:               map[uint64]*world.EntityHandle{},
		hiddenEntities:         map[uuid.UUID]struct{}{},
		blobs:                  map[uint64][]byte{},
		chunkRadius:            int32(r),
		maxChunkRadius:         int32(conf.MaxChunkRadius),
		emoteChatMuted:         conf.EmoteChatMuted,
		conn:                   conn,
		currentEntityRuntimeID: 1,
		heldSlot:               new(uint32),
		recipes:                make(map[uint32]recipe.Recipe),
		conf:                   conf,
		hudUpdates:             make(map[hud.Element]bool),
		hiddenHud:              make(map[hud.Element]struct{}),
		debugShapes:            make(map[int]debug.Shape),
		debugShapeUpdates:      make([]debugShapeUpdate, 0, 256),
	}
	s.viewLayer = world.NewViewLayer(s)
	s.openedWindow.Store(inventory.New(1, nil))
	s.openedPos.Store(&cube.Pos{})

	var scoreboardName string
	var scoreboardLines []string
	s.currentScoreboard.Store(&scoreboardName)
	s.currentLines.Store(&scoreboardLines)

	if conf.BlockRegistry == nil {
		s.br = world.DefaultBlockRegistry
	} else {
		s.br = conf.BlockRegistry
	}

	s.registerHandlers()
	s.sendBiomes()
	groups, items := creativeContent(s.br)
	s.writePacket(&packet.CreativeContent{Groups: groups, Items: items})
	s.sendRecipes()
	s.sendArmourTrimData()
	s.SendSpeed(0.1)
	go func() {
		for {
			select {
			case <-s.closeBackground:
				return
			case pk := <-s.packets:
				_ = conn.WritePacket(pk)
			}
		}
	}()
	return s
}

func (s *Session) SetHandle(handle *world.EntityHandle, skin skin.Skin) {
	s.ent = handle
	s.entityRuntimeIDs[handle] = selfEntityRuntimeID
	s.entities[selfEntityRuntimeID] = handle

	s.joinSkin = skin
	sessions.Add(s)
}

func (s *Session) Spawn(c Controllable, tx *world.Tx) {
	s.SendHealth(c.Health(), c.MaxHealth(), c.Absorption())
	s.SendExperience(c.ExperienceLevel(), c.ExperienceProgress())
	s.SendFood(c.Food(), 0, 0)

	pos := c.Position()
	s.chunkLoader = world.NewLoader(int(s.chunkRadius), tx.World(), s)
	s.chunkLoader.Move(tx, pos)
	s.writePacket(&packet.NetworkChunkPublisherUpdate{
		Position: protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
		Radius:   uint32(s.chunkRadius) << 4,
	})

	s.sendAvailableEntities(tx.World())

	c.SetGameMode(c.GameMode())
	for _, e := range c.Effects() {
		s.SendEffect(e)
	}
	s.ViewEntityState(c)

	s.sendInv(s.inv, protocol.WindowIDInventory)
	s.sendInv(s.ui, protocol.WindowIDUI)
	s.sendInv(s.offHand, protocol.WindowIDOffHand)
	s.sendInv(s.armour.Inventory(), protocol.WindowIDArmour)

	chat.Global.Subscribe(c)
	if !s.conf.JoinMessage.Zero() {
		chat.Global.Writet(s.conf.JoinMessage, s.conn.IdentityData().DisplayName)
	}

	go s.background()
	go s.handlePackets()
}

func (s *Session) Close(tx *world.Tx, c Controllable) {
	s.once.Do(func() {
		s.close(tx, c)
	})
}

func (s *Session) close(tx *world.Tx, c Controllable) {
	if tx != nil {
		c.MoveItemsToInventory()
		s.closeCurrentContainer(tx, false)
	}
	if s.viewLayer != nil {
		_ = s.viewLayer.Close()
	}

	s.conf.HandleStop(tx, c)

	_ = s.inv.Close()
	_ = s.offHand.Close()
	_ = s.armour.Close()

	if tx != nil {
		s.chunkLoader.Close(tx)
	}

	if !s.conf.QuitMessage.Zero() {
		chat.Global.Writet(s.conf.QuitMessage, s.conn.IdentityData().DisplayName)
	}
	chat.Global.Unsubscribe(c)

	if tx != nil {
		tx.RemoveEntity(c)
		_ = s.ent.Close()
	}

	sessions.Remove(s, c)
	s.entityMutex.Lock()
	clear(s.entityRuntimeIDs)
	clear(s.entities)
	s.entityMutex.Unlock()
}

func (s *Session) CloseConnection() {
	s.connOnce.Do(func() {
		_ = s.conn.Close()
		close(s.closeBackground)
	})
}

func (s *Session) Addr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *Session) Latency() time.Duration {
	return s.conn.Latency()
}

func (s *Session) withControllable(ctx context.Context, f func(tx *world.Tx, c Controllable) error) error {
	_, err := world.CallRef(ctx, world.NewEntityRef[Controllable](s.ent), func(tx *world.Tx, c Controllable) (struct{}, error) {
		return struct{}{}, f(tx, c)
	})
	return err
}

func sessionOwnerStopped(err error) bool {
	return errors.Is(err, world.ErrEntityClosed) || errors.Is(err, world.ErrWorldClosed) || errors.Is(err, world.ErrTaskCancelled)
}

func (s *Session) ClientData() login.ClientData {
	return s.conn.ClientData()
}

func (s *Session) handlePackets() {
	defer func() {

		if err := s.withControllable(context.Background(), func(_ *world.Tx, c Controllable) error {
			_ = c.Close()
			return nil
		}); err != nil && !sessionOwnerStopped(err) {
			s.conf.Log.Debug("close controllable: " + err.Error())
		}

		if err := s.withControllable(context.Background(), func(tx *world.Tx, c Controllable) error {
			s.Close(tx, c)
			return nil
		}); err != nil && !sessionOwnerStopped(err) {
			s.conf.Log.Debug("close session: " + err.Error())
		}
	}()
	for {
		pk, err := s.conn.ReadPacket()
		if err != nil {
			return
		}
		err = s.withControllable(context.Background(), func(tx *world.Tx, c Controllable) error {
			return s.handlePacket(pk, tx, c)
		})
		if err != nil {
			if sessionOwnerStopped(err) {
				return
			}
			s.conf.Log.Debug("process packet: " + err.Error())
			return
		}
	}
}

func (s *Session) background() {
	var (
		r          map[string]map[int]cmd.Runnable
		enums      map[string]cmd.Enum
		enumValues map[string][]string
		softEnums  = make(map[string]struct{})
		ok         bool
		i          int
	)

	if err := s.withControllable(context.Background(), func(_ *world.Tx, c Controllable) error {
		r = s.sendAvailableCommands(c, softEnums)
		enums, enumValues = s.enums(c)
		return nil
	}); err != nil {
		if !sessionOwnerStopped(err) {
			s.conf.Log.Debug("prepare command updates: " + err.Error())
		}
		return
	}

	t := time.NewTicker(time.Second / 20)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := s.withControllable(context.Background(), func(tx *world.Tx, c Controllable) error {
				if i++; i%20 == 0 {

					r = s.resendEnums(enums, enumValues, softEnums, r, c)
				}
				if i%100 == 0 {

					if r, ok = s.resendCommands(r, c, softEnums); ok {
						enums, enumValues = s.enums(c)
					}
				}
				s.sendChunks(tx, c)
				return nil
			}); err != nil {
				if !sessionOwnerStopped(err) {
					s.conf.Log.Debug("update session background: " + err.Error())
				}
				return
			}
		case <-s.closeBackground:
			return
		}
	}
}

func (s *Session) sendChunks(tx *world.Tx, c Controllable) {
	var worldSwitched bool
	if w := tx.World(); s.chunkLoader.World() != w && w != nil {
		worldSwitched = true
		s.handleWorldSwitch(w, tx, c)
	}
	pos := c.Position()
	s.chunkLoader.Move(tx, pos)
	chunkPos := world.ChunkPos{int32(pos[0]) << 4, int32(pos[2]) << 4}
	if s.lastChunkPos != chunkPos || worldSwitched {
		s.lastChunkPos = chunkPos
		s.writePacket(&packet.NetworkChunkPublisherUpdate{
			Position: protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
			Radius:   uint32(s.chunkRadius) << 4,
		})
	}

	s.blobMu.Lock()
	const maxChunkTransactions = 8
	toLoad := maxChunkTransactions - len(s.openChunkTransactions)
	s.blobMu.Unlock()
	if toLoad > 4 {
		toLoad = 4
	}
	s.chunkLoader.Load(tx, toLoad)
}

func (s *Session) handleWorldSwitch(w *world.World, tx *world.Tx, c Controllable) {
	if s.conn.ClientCacheEnabled() {
		s.blobMu.Lock()
		s.blobs = map[uint64][]byte{}
		s.openChunkTransactions = nil
		s.blobMu.Unlock()
	}

	dim, _ := world.DimensionID(w.Dimension())
	same := w.Dimension() == s.chunkLoader.World().Dimension()
	if !same {
		s.changeDimension(int32(dim), false, c)
	}
	s.ViewEntityTeleport(c, c.Position())
	s.chunkLoader.ChangeWorld(tx, w)
}

func (s *Session) changeDimension(dim int32, silent bool, c Controllable) {
	s.changingDimension.Store(true)
	h := s.handlers[packet.IDServerBoundLoadingScreen].(*ServerBoundLoadingScreenHandler)
	id := h.currentID.Add(1)
	h.expectedID.Store(id)

	s.writePacket(&packet.ChangeDimension{
		Dimension:       dim,
		Position:        vec64To32(c.Position().Add(entityOffset(c))),
		LoadingScreenID: protocol.Option(id),
	})
	s.writePacket(&packet.StopSound{StopAll: silent})
	s.writePacket(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn})

	s.writePacket(&packet.PlayerAction{
		EntityRuntimeID: selfEntityRuntimeID,
		ActionType:      protocol.PlayerActionDimensionChangeDone,
	})
}

func (s *Session) ChangingDimension() bool {
	return s.changingDimension.Load()
}

func (s *Session) ChunkRadius() int32 {
	return s.chunkRadius
}

func (s *Session) handlePacket(pk packet.Packet, tx *world.Tx, c Controllable) (err error) {
	handler, ok := s.handlers[pk.ID()]
	if !ok {
		s.conf.Log.Debug("unhandled packet", "packet", fmt.Sprintf("%T", pk), "data", fmt.Sprintf("%+v", pk)[1:])
		return nil
	}
	if handler == nil {

		return nil
	}
	if err := handler.Handle(pk, s, tx, c); err != nil {
		return fmt.Errorf("%T: %w", pk, err)
	}
	return nil
}

func (s *Session) registerHandlers() {
	s.handlers = map[uint32]packetHandler{
		packet.IDActorEvent:                nil,
		packet.IDAdventureSettings:         nil,
		packet.IDAnimate:                   nil,
		packet.IDAnvilDamage:               nil,
		packet.IDBlockActorData:            &BlockActorDataHandler{},
		packet.IDBlockPickRequest:          &BlockPickRequestHandler{},
		packet.IDBookEdit:                  &BookEditHandler{},
		packet.IDBossEvent:                 nil,
		packet.IDClientCacheBlobStatus:     &ClientCacheBlobStatusHandler{},
		packet.IDCommandRequest:            &CommandRequestHandler{},
		packet.IDContainerClose:            &ContainerCloseHandler{},
		packet.IDEmote:                     &EmoteHandler{},
		packet.IDEmoteList:                 nil,
		packet.IDFilterText:                nil,
		packet.IDInteract:                  &InteractHandler{},
		packet.IDInventoryTransaction:      &InventoryTransactionHandler{},
		packet.IDItemStackRequest:          &ItemStackRequestHandler{changes: map[byte]map[byte]changeInfo{}, responseChanges: map[int32]map[*inventory.Inventory]map[byte]responseChange{}},
		packet.IDLecternUpdate:             &LecternUpdateHandler{},
		packet.IDMobEquipment:              &MobEquipmentHandler{},
		packet.IDModalFormResponse:         &ModalFormResponseHandler{forms: make(map[uint32]form.Form)},
		packet.IDMovePlayer:                nil,
		packet.IDNPCRequest:                &NPCRequestHandler{},
		packet.IDPlayerAction:              &PlayerActionHandler{},
		packet.IDPlayerAuthInput:           &PlayerAuthInputHandler{},
		packet.IDPlayerSkin:                &PlayerSkinHandler{},
		packet.IDRequestAbility:            &RequestAbilityHandler{},
		packet.IDRequestChunkRadius:        &RequestChunkRadiusHandler{},
		packet.IDRespawn:                   &RespawnHandler{},
		packet.IDSetPlayerFurnaceOptions:   nil,
		packet.IDSetPlayerInventoryOptions: nil,
		packet.IDSubChunkRequest:           &SubChunkRequestHandler{},
		packet.IDText:                      &TextHandler{},
		packet.IDServerBoundLoadingScreen:  &ServerBoundLoadingScreenHandler{},
		packet.IDServerBoundDiagnostics:    &ServerBoundDiagnosticsHandler{},
	}
}

func (s *Session) writePacket(pk packet.Packet) {
	if s == Nop {
		return
	}
	select {
	case s.packets <- pk:
	case <-s.closeBackground:
	}
}

type actorIdentifier struct {
	ID string `nbt:"id"`
}

func (s *Session) sendAvailableEntities(w *world.World) {
	var identifiers []actorIdentifier
	for _, t := range w.EntityRegistry().Types() {
		identifiers = append(identifiers, actorIdentifier{ID: t.EncodeEntity()})
	}
	serialisedEntityData, err := nbt.Marshal(map[string]any{"idlist": identifiers})
	if err != nil {
		panic("should never happen")
	}
	s.writePacket(&packet.AvailableActorIdentifiers{SerialisedEntityIdentifiers: serialisedEntityData})
}
