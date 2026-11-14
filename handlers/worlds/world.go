package worlds

import (
	"archive/zip"
	"context"
	"fmt"
	"image/png"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/scripting"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/Daniel-Ric/CraftBox/utils/resourcepack"
	"github.com/google/uuid"

	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/dragonfly/server/world/biome"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

type WorldSettings struct {
	VoidGen            bool
	SaveImage          bool
	SaveEntities       bool
	SaveCustomEntities bool
	SaveInventories    bool
	ExcludedMobs       []string
	ChunkRadius        int32
	Script             string
	Players            bool
	MapArt             bool
	BlockUpdates       bool
	EntityCulling      bool
}

type serverState struct {
	captureRules                   map[string]protocol.GameRule
	captureMode, captureDifficulty *int32
	captureCommands                *bool
	rainLevel, lightningLevel      float32
	serverName                     string
	useHashedRids                  bool
	haveStartGame                  bool
	worldCounter                   int
	worldName                      string
	realChunkRadius                int32

	biomes *world.BiomeRegistry
	blocks world.BlockRegistry

	behaviorPack *behaviourpack.Pack
	resourcePack *resourcepack.Pack

	customBlocks         []protocol.BlockEntry
	openItemContainers   map[byte]*itemContainer
	playerInventory      []protocol.ItemInstance
	itemNames            map[int32]string
	dimensions           map[int]protocol.DimensionDefinition
	playerSkins          map[uuid.UUID]*protocol.Skin
	playerUniqueIDs      map[uuid.UUID]int64
	playerXUIDs          map[uuid.UUID]string
	playerSaveOverrides  map[uuid.UUID]bool
	isHive               bool
	mapsMu               sync.Mutex
	knownMaps            map[int64]*worldstate.Map
	requestedMaps        map[int64]struct{}
	entityProperties     map[string][]entity.EntityPropertyDef
	playerPropertyValues map[string]any

	entityRenderDistances []float32
}

func (s *serverState) getEntityRenderDistance() float32 {
	var entityRenderDistance float32 = 0
	if len(s.entityRenderDistances) > 4 {
		var sum float32
		for _, dist := range s.entityRenderDistances {
			sum += dist
		}
		entityRenderDistance = sum / float32(len(s.entityRenderDistances))
	}
	return entityRenderDistance
}

type worldsHandler struct {
	wg      sync.WaitGroup
	ctx     context.Context
	session *proxy.Session
	mapUI   *MapUI
	log     *logrus.Entry

	scripting *scripting.VM

	worldStateMu      sync.Mutex
	menuWorldMu       sync.Mutex
	captureSettingsMu sync.Mutex
	worldState        *worldstate.World

	serverState               serverState
	settings                  WorldSettings
	commandEntityTypes        map[string]struct{}
	commandCompletionMu       sync.Mutex
	commandCompletionPrefix   string
	commandMetadataReady      bool
	commandStarted            time.Time
	commandRegionSelection    commandRegionSelection
	commandStructureSelection structureCommandSelection
	regions                   []messages.WorldRegion
}

type itemContainer struct {
	OpenPacket *packet.ContainerOpen
	Content    *packet.InventoryContent
}

func NewWorldsHandler(ctx context.Context, settings WorldSettings) func() *proxy.Handler {
	settings.ExcludedMobs = slices.DeleteFunc(settings.ExcludedMobs, func(mob string) bool {
		return mob == ""
	})

	if settings.ChunkRadius == 0 {
		settings.ChunkRadius = 76
	}

	return func() *proxy.Handler {
		w := &worldsHandler{
			ctx:      ctx,
			log:      logrus.WithField("part", "WorldsHandler"),
			settings: settings,
		}

		return &proxy.Handler{
			PickBlock: w.pickBlock,
			Name:      "Worlds",

			SessionStart: w.onSessionStart,

			GameDataModifier: func(s *proxy.Session, gd *minecraft.GameData) {
				gd.ClientSideGeneration = false
			},

			OnConnect: w.onConnect,

			PacketCallback: w.packetHandler,
			OnSessionEnd: func(s *proxy.Session, wg *sync.WaitGroup) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					w.SaveAndReset(true, nil)
					w.wg.Wait()
				}()
			},

			OnPlayerMove: func(s *proxy.Session) {
				playerPos := s.Player.Position
				w.currentWorld(func(world *worldstate.World) {
					world.PlayerMove(playerPos, w.serverState.getEntityRenderDistance(), 0)
				})
			},
		}
	}
}

func (w *worldsHandler) onSessionStart(session *proxy.Session, serverName string) error {
	w.session = session
	w.serverState = serverState{
		serverName:           serverName,
		worldCounter:         0,
		openItemContainers:   make(map[byte]*itemContainer),
		dimensions:           make(map[int]protocol.DimensionDefinition),
		playerSkins:          make(map[uuid.UUID]*protocol.Skin),
		playerUniqueIDs:      make(map[uuid.UUID]int64),
		playerXUIDs:          make(map[uuid.UUID]string),
		playerSaveOverrides:  make(map[uuid.UUID]bool),
		isHive:               isHiveServer(session.ServerAddress(), serverName),
		knownMaps:            make(map[int64]*worldstate.Map),
		requestedMaps:        make(map[int64]struct{}),
		biomes:               world.DefaultBiomes.Clone(),
		entityProperties:     make(map[string][]entity.EntityPropertyDef),
		behaviorPack:         behaviourpack.New(serverName),
		resourcePack:         resourcepack.New(),
		playerPropertyValues: make(map[string]any),
	}

	w.mapUI = NewMapUI(w)

	w.scripting = nil
	if w.settings.Script != "" {
		w.scripting = scripting.New()
		w.scripting.GetWorld = func() *worldstate.World {
			return w.worldState
		}
		w.scripting.DisplayChatMessage = func(msg string) {
			w.session.SendMessage(msg)
		}
		w.scripting.SetIngameMap = func(enabled bool) {
			w.mapUI.SetEnabled(enabled)
		}
		err := w.scripting.Load(w.settings.Script)
		if err != nil {
			return err
		}
	}

	w.registerCommands(session)

	worldState, err := worldstate.New(w.ctx, w.serverState.dimensions, w.mapUI.SetChunk)
	if err != nil {
		return err
	}
	worldState.VoidGen = w.settings.VoidGen
	w.worldState = worldState
	return nil
}

func (w *worldsHandler) onConnect(_ *proxy.Session) bool {
	messages.SendEvent(&messages.EventSetUIState{
		State: messages.UIStateMain,
	})
	messages.SendEvent(&messages.EventSetValue{
		Name:  "worldName",
		Value: w.worldState.Name,
	})

	w.session.ClientWritePacket(&packet.ChunkRadiusUpdated{
		ChunkRadius: w.settings.ChunkRadius,
	})

	w.session.Server.WritePacket(&packet.RequestChunkRadius{
		ChunkRadius: w.settings.ChunkRadius,
	})

	mapItemID, _ := world.ItemRidByName("minecraft:filled_map")
	mapItem.Stack.ItemType.NetworkID = mapItemID

	w.session.SendMessage(locale.Loc("use_setname", nil))
	w.mapUI.Start(w.ctx)
	return false
}

func (w *worldsHandler) currentWorld(fn func(world *worldstate.World)) {
	w.worldStateMu.Lock()
	if w.worldState != nil && !messages.WorldDeleted(w.worldState.ID) {
		fn(w.worldState)
	}
	w.worldStateMu.Unlock()
}

func (w *worldsHandler) setVoidGen(val bool) bool {
	changed := false
	w.currentWorld(func(world *worldstate.World) {
		world.VoidGen = val
		w.settings.VoidGen = val
		changed = true
	})
	if !changed {
		return w.commandError("No active world capture.", "/cb status")
	}
	var s = locale.Loc("void_generator_false", nil)
	if val {
		s = locale.Loc("void_generator_true", nil)
	}
	w.session.SendMessage(s)

	var voidGen = "false"
	if val {
		voidGen = "true"
	}

	messages.SendEvent(&messages.EventSetValue{
		Name:  "voidGen",
		Value: voidGen,
	})
	return true
}

func (w *worldsHandler) setWorldName(val string) bool {
	val = strings.TrimSpace(val)
	err := w.renameWorldState(val)
	if err != nil {
		return w.commandError(err.Error(), "/setname <world name>")
	}
	w.session.SendMessage(locale.Loc("worldname_set", locale.Strmap{"Name": val}))
	messages.SendEvent(&messages.EventSetValue{
		Name:  "worldName",
		Value: val,
	})
	return true
}

func (w *worldsHandler) SaveAndReset(end bool, dim world.Dimension) {
	w.menuWorldMu.Lock()
	defer w.menuWorldMu.Unlock()
	w.saveAndReset(end, dim)
}

// saveAndReset requires menuWorldMu, so a form action cannot race a capture switch.
func (w *worldsHandler) saveAndReset(end bool, dim world.Dimension) {

	w.worldStateMu.Lock()
	defer w.worldStateMu.Unlock()
	if w.worldState == nil {
		return
	}
	if dim == nil {
		dim = w.worldState.Dimension()
	}
	if w.session.Server == nil && len(w.worldState.StoredChunks) > 0 {
		return
	}

	worldState := w.worldState
	defer worldState.ReleaseScene()
	w.worldState = nil
	defer messages.UnregisterWorldEditor(worldState.ID)
	defer messages.UnregisterCaptureDiscarder(worldState.ID)
	defer func(regions []messages.WorldRegion) {
		for _, r := range regions {
			messages.UnregisterCaptureDiscarder(r.ID)
		}
	}(append([]messages.WorldRegion{}, w.regions...))

	if len(worldState.StoredChunks) == 0 {
		messages.MarkCaptureDiscarded(worldState.ID)
		if err := worldState.Discard(); err != nil {
			w.log.WithError(err).Warn("Could not close empty transfer state")
		}
		messages.SendEvent(&messages.EventWorldAbandoned{WorldID: worldState.ID})
	}

	if !messages.WorldDeleted(worldState.ID) && len(worldState.StoredChunks) > 0 {
		w.serverState.worldCounter += 1

		if err := w.saveWorldState(worldState, w.session.Player, w.serverState.behaviorPack); err != nil {
			w.log.Error(err)
			messages.SendEvent(&messages.EventProcessingWorldUpdate{WorldID: worldState.ID, WorldName: worldState.Name, State: "Saving failed"})
		}

	}

	if !end {
		w.serverState.openItemContainers = make(map[byte]*itemContainer)
		w.regions = nil
		worldState, err := worldstate.New(w.ctx, w.serverState.dimensions, w.mapUI.SetChunk)
		if err != nil {
			w.log.Error(err)
		}
		worldState.VoidGen = w.settings.VoidGen
		worldState.SetDimension(dim)
		w.worldState = worldState
		w.openWorldState()
	}
}

func (w *worldsHandler) saveWorldState(worldState *worldstate.World, player proxy.Player, behaviorPack *behaviourpack.Pack) error {
	text := locale.Loc("saving_world", locale.Strmap{"Name": worldState.Name, "Count": len(worldState.StoredChunks)})
	w.log.Info(text)
	w.session.SendMessage(text)

	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   worldState.ID,
		WorldName: worldState.Name,
		State:     "Saving",
	})

	var playerSkins = make(map[uuid.UUID]*protocol.Skin)
	maps.Copy(playerSkins, w.serverState.playerSkins)

	playerNBT, err := w.playerData(player)
	if err != nil {
		return err
	}
	if err = w.flushContainerInventories(worldState); err != nil {
		return err
	}
	err = worldState.Save(
		player, playerNBT,
		behaviorPack,
		w.excludedMobs(),
		worldstate.SaveOptions{KeepPlayer: w.keepPlayer(), PlayerSkins: playerSkins, Hive: w.serverState.isHive, KnownMaps: w.knownMapsSnapshot()},
		w.capturedGameData(), w.serverState.serverName,
		w.settings.EntityCulling,
		w.serverState.getEntityRenderDistance(),
	)
	if err != nil {
		return err
	}

	if w.settings.MapArt && len(w.regions) == 0 {
		if _, _, err := worldState.ExportMapArt(worldState.Folder + "_mapart"); err != nil {
			return err
		}
	}
	w.mapUI.redrawAt(player.Position)
	if len(w.regions) > 0 {
		return w.saveRegionExports(worldState, player, w.regions)
	}
	if err := applyCaptureMetadata(worldState); err != nil {
		return err
	}
	if w.settings.SaveImage {
		if err := w.saveMapImage(worldState); err != nil {
			w.log.Error(err)
		}
	}
	total, active := worldState.EntityCounts(w.serverState.getEntityRenderDistance())
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		if err := w.writeWorldArchive(worldState, total, active); err != nil {
			w.log.Error(err)
			messages.SendEvent(&messages.EventProcessingWorldUpdate{WorldID: worldState.ID, WorldName: worldState.Name, State: "Saving failed"})
		}
	}()
	return nil
}

func (w *worldsHandler) saveMapImage(worldState *worldstate.World) error {
	return w.saveMapImageWhere(worldState, nil)
}

func (w *worldsHandler) saveMapImageWhere(worldState *worldstate.World, keep func(x, z int32) bool) error {
	imagePath := worldState.Folder + ".png"
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		return err
	}
	f, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	encodeErr := png.Encode(f, w.mapUI.ToImageWhere(keep))
	closeErr := f.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	worldState.ImagePath = imagePath
	return nil
}

func (w *worldsHandler) writeWorldArchive(worldState *worldstate.World, total, active int) error {
	return messages.WithCaptureWrite(worldState.ID, func() error { return w.writeWorldArchiveLocked(worldState, total, active) })
}

func (w *worldsHandler) writeWorldArchiveLocked(worldState *worldstate.World, total, active int) error {

	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   worldState.ID,
		WorldName: worldState.Name,
		State:     "Writing mcworld file",
	})

	filename := worldState.Folder + ".mcworld"
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	utils.ZipCompressPool(zw)
	err = zw.AddFS(os.DirFS(worldState.Folder))
	if err != nil {
		return err
	}
	err = zw.Close()
	if err != nil {
		return err
	}
	w.log.WithFields(logrus.Fields{
		"Entities": fmt.Sprintf("%d (%d)", total, active),
		"Chunks":   len(worldState.StoredChunks),
	}).Info(locale.Loc("saved", locale.Strmap{"Name": filename}))
	messages.SendEvent(&messages.EventFinishedSavingWorld{
		WorldID:   worldState.ID,
		WorldName: worldState.Name,
		Filepath:  filename,
		ImagePath: worldState.ImagePath,
		Chunks:    len(worldState.StoredChunks),
		Entities:  total,
	})
	return nil
}

func (w *worldsHandler) defaultWorldName() string {
	worldName := "world"
	if w.serverState.worldCounter > 0 {
		worldName = fmt.Sprintf("world-%d", w.serverState.worldCounter)
	} else if w.serverState.worldName != "" {
		worldName = w.serverState.worldName
	}
	return worldName
}

func (w *worldsHandler) openWorldState() {
	worldName := utils.MakeValidFilename(w.defaultWorldName())
	serverName := utils.MakeValidFilename(w.serverState.serverName)
	folder := utils.PathData("worlds", serverName, worldName+"--"+w.worldState.ID)

	w.worldState.BiomeRegistry = w.serverState.biomes
	w.worldState.BlockRegistry = w.serverState.blocks
	w.worldState.ResourcePacks = w.session.ActiveResourcePacks()
	w.serverState.isHive = w.serverState.isHive || hasHivePacks(w.worldState.ResourcePacks)
	w.worldState.Hive = w.serverState.isHive
	w.worldState.OnMapSeen = w.onMapSeen
	w.worldState.BlockEntries = w.serverState.customBlocks
	w.worldState.SkipCustomEntities = !w.settings.SaveCustomEntities
	w.worldState.LocalPlayerUniqueID = w.session.Player.UniqueID
	for _, id := range w.serverState.playerUniqueIDs {
		w.worldState.RegisterScenePlayerID(id)
	}
	w.worldState.UseHashedRids = w.serverState.useHashedRids
	w.captureSettingsMu.Lock()
	rain, lightning, commands := w.serverState.rainLevel, w.serverState.lightningLevel, w.serverState.captureCommands
	w.captureSettingsMu.Unlock()
	w.worldState.SetWeather(rain, lightning)
	if commands != nil {
		w.worldState.SetCommandsEnabled(*commands)
	}
	w.worldState.Open(w.defaultWorldName(), folder)
	messages.RecordCaptureFolder(w.worldState.ID, w.worldState.Folder)
	// StartGame and dimension transitions can be followed immediately by another
	// transfer. Publish a library entry only after actual terrain arrives.
	state, serverName := w.worldState, w.serverState.serverName
	state.OnFirstChunk = func() {
		messages.SendEvent(&messages.EventWorldOpened{WorldID: state.ID, WorldName: state.Name, Folder: state.Folder, ServerName: serverName, HeightRange: [2]int(state.Range())})
	}
	w.mapUI.SwitchWorld(w.worldState.ID)
	parentID := w.worldState.ID
	discard := func() error {
		w.worldStateMu.Lock()
		defer w.worldStateMu.Unlock()
		if w.worldState != nil && w.worldState.ID == parentID && messages.WorldDeleted(parentID) {
			return w.worldState.Discard()
		}
		return nil
	}
	messages.RegisterCaptureDiscarder(parentID, discard)
	messages.RegisterMetadataEditor(parentID, func(request messages.EditWorld) error {
		if request.Export != nil {
			if err := request.Export.Validate(); err != nil {
				return err
			}
		}
		w.worldStateMu.Lock()
		defer w.worldStateMu.Unlock()
		if w.worldState == nil || w.worldState.ID != parentID {
			return fmt.Errorf("Warte auf das fertige Weltarchiv")
		}
		if request.Spawn != nil {
			bounds := w.worldState.Range()
			if int((*request.Spawn)[1]) < bounds[0] || int((*request.Spawn)[1])+1 > bounds[1] {
				return fmt.Errorf("Spawn point is outside the dimension height")
			}
		}
		messages.StoreWorldMetadata(request)
		return nil
	})
	messages.RegisterWorldEditor(parentID, func(request messages.CreateWorldRegion) (messages.WorldRegion, error) {
		w.worldStateMu.Lock()
		defer w.worldStateMu.Unlock()
		if w.worldState == nil || w.worldState.ID != parentID {
			return messages.WorldRegion{}, fmt.Errorf("This world is no longer being captured")
		}
		for _, region := range w.regions {
			if region.Bounds.Overlaps(request.Bounds) {
				return messages.WorldRegion{}, fmt.Errorf("This region overlaps with %s", region.Name)
			}
		}
		region := messages.WorldRegion{ID: uuid.NewString(), ParentID: parentID, Name: strings.TrimSpace(request.Name), Bounds: request.Bounds, Move: request.Move}
		w.regions = append(w.regions, region)
		messages.RegisterCaptureDiscarder(region.ID, discard)
		if edit := messages.WorldMetadata(parentID); request.Move && edit.Spawn != nil && request.Bounds.Contains((*edit.Spawn)[0]>>4, (*edit.Spawn)[2]>>4) {
			messages.ClearWorldSpawn(parentID)
		}
		messages.SendEvent(&messages.EventWorldRegionCreated{Region: region})
		return region, nil
	})
}

func (w *worldsHandler) renameWorldState(name string) error {
	if err := validateCaptureName(name); err != nil {
		return err
	}
	worldName := utils.MakeValidFilename(name)
	serverName := utils.MakeValidFilename(w.serverState.serverName)
	var err error = fmt.Errorf("No active world capture")
	w.currentWorld(func(world *worldstate.World) {
		folder := utils.PathData("worlds", serverName, worldName+"--"+world.ID)
		err = world.Rename(name, folder)
		if err == nil {
			messages.RecordCaptureFolder(world.ID, world.Folder)
			messages.StoreWorldMetadata(messages.EditWorld{WorldID: world.ID, Name: &name})
			if len(world.StoredChunks) > 0 {
				messages.SendEvent(&messages.EventWorldOpened{WorldID: world.ID, WorldName: name, Folder: world.Folder, ServerName: w.serverState.serverName, HeightRange: [2]int(world.Range())})
			}
		}
	})
	return err
}
