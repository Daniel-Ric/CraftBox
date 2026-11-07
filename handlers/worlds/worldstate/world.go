package worldstate

import (
	"context"
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"log/slog"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/Daniel-Ric/CraftBox/utils/resourcepack"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

type resourcePackDependency struct {
	UUID    string `json:"pack_id"`
	Version [3]int `json:"version"`
}

type World struct {
	discarded     bool
	scene         *scene.Store
	sceneRetained bool
	ID            string
	ImagePath     string

	ctx       context.Context
	cancelCtx context.CancelCauseFunc

	ChunkFunc func(pos world.ChunkPos, ch *chunk.Chunk)
	// OnFirstChunk promotes a connection's provisional state to a terrain capture.
	// It runs once, synchronously, before the first non-air chunk update.
	OnFirstChunk func()

	BlockRegistry world.BlockRegistry
	BiomeRegistry *world.BiomeRegistry

	dimension            world.Dimension
	dimRange             cube.Range
	dimensionDefinitions map[int]protocol.DimensionDefinition
	StoredChunks         map[world.ChunkPos]struct{}

	stateLock sync.Mutex
	memState  *memoryState
	provider  *mcdb.DB
	onceOpen  sync.Once
	opened    bool

	ResourcePacks         []*resource.Pack
	BlockEntries          []protocol.BlockEntry
	LocalPlayerUniqueID   int64
	SkipCustomEntities    bool
	scenePlayerIDs        map[int64]bool
	exportedResourcePacks []*resource.Pack
	resourcePacksDone     chan error

	players map[uuid.UUID]*player

	VoidGen                   bool
	timeSync                  time.Time
	time                      int
	rainLevel, lightningLevel float32
	haveWeather               bool
	commandsEnabled           *bool
	Name                      string
	Folder                    string

	UseHashedRids    bool
	blockUpdatesLock sync.Mutex
	blockUpdates     map[world.ChunkPos][]blockUpdate
	onChunkUpdate    func(pos world.ChunkPos, chunk *chunk.Chunk)
	IgnoredChunks    map[world.ChunkPos]bool

	log *logrus.Entry
}

type blockUpdate struct {
	rid   uint32
	pos   protocol.BlockPos
	layer uint8
}

type Map struct {
	Decorations       []any            `nbt:"decorations"`
	Dimension         uint8            `nbt:"dimension"`
	Height            int16            `nbt:"height"`
	Width             int16            `nbt:"width"`
	MapID             int64            `nbt:"mapId"`
	Scale             uint8            `nbt:"scale"`
	UnlimitedTracking bool             `nbt:"unlimitedTracking"`
	ZCenter           int32            `nbt:"zCenter"`
	XCenter           int32            `nbt:"xCenter"`
	FullyExplored     bool             `nbt:"fullyExplored"`
	ParentMapId       int64            `nbt:"parentMapId"`
	Colors            [0xffff + 1]byte `nbt:"colors"`
	MapLocked         bool             `nbt:"mapLocked"`
}

func New(ctx context.Context, dimensionDefinitions map[int]protocol.DimensionDefinition, onChunkUpdate func(pos world.ChunkPos, chunk *chunk.Chunk)) (*World, error) {
	ctxw, cancel := context.WithCancelCause(ctx)
	w := &World{
		ID:        uuid.NewString(),
		ctx:       ctxw,
		cancelCtx: cancel,

		StoredChunks:         make(map[world.ChunkPos]struct{}),
		dimensionDefinitions: dimensionDefinitions,
		memState:             newWorldState(),
		players:              make(map[uuid.UUID]*player),
		blockUpdates:         make(map[world.ChunkPos][]blockUpdate),
		onChunkUpdate:        onChunkUpdate,
		IgnoredChunks:        make(map[world.ChunkPos]bool),
		log:                  logrus.WithFields(logrus.Fields{"part": "world"}),
	}

	return w, nil
}

func (w *World) storeMemToProvider() error {
	if w.discarded {
		return nil
	}
	if len(w.memState.chunks) == 0 {
		return nil
	}
	if w.provider == nil {
		// Keep annotations in place, including when database creation fails or a
		// reader has the notes open. Only reset other owned capture files.
		if strings.TrimSpace(w.Folder) == "" || filepath.Clean(w.Folder) == "." {
			return fmt.Errorf("Capture folder is not ready")
		}
		root, e := filepath.Abs(w.Folder)
		if e != nil {
			return e
		}
		if filepath.Dir(root) == root {
			return fmt.Errorf("Capture folder cannot be a filesystem root")
		}
		entries, e := os.ReadDir(root)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), "craftbox-capture.json") {
				continue
			}
			target := filepath.Join(root, entry.Name())
			relative, e := filepath.Rel(root, target)
			if e != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
				return fmt.Errorf("Invalid capture cleanup path")
			}
			if e = os.RemoveAll(target); e != nil {
				return e
			}
		}
		w.log.Debugf("Opening provider in %s", w.Folder)
		if e := os.MkdirAll(w.Folder, 0o777); e != nil {
			return e
		}
		provider, err := mcdb.Config{
			Log: slog.Default(),
			LDBOptions: &opt.Options{
				Compression: opt.DefaultCompression,
			},
			Blocks: w.BlockRegistry,
		}.Open(w.Folder)
		if err != nil {
			return err
		}
		w.provider = provider

		w.resourcePacksDone = make(chan error, 1)
		go func() {
			defer close(w.resourcePacksDone)
			err := w.addResourcePacks()
			if err != nil {
				w.resourcePacksDone <- err
			}
		}()
	}

	for pos, ch := range w.memState.chunks {

		empty := true
		for _, sc := range ch.Sub() {
			if !sc.Empty() {
				empty = false
				break
			}
		}
		if empty {
			continue
		}

		var blockEntities []chunk.BlockEntity
		for pos, ent := range ch.BlockEntities {
			blockEntities = append(blockEntities, chunk.BlockEntity{
				Pos:  pos,
				Data: ent,
			})
		}

		err := w.provider.StoreColumn(pos, w.dimension, &chunk.Column{
			Chunk:         ch.Chunk,
			BlockEntities: blockEntities,
		})
		if err != nil {
			w.log.Error("StoreColumn", err)
		}
		delete(w.memState.chunks, pos)
	}
	return nil
}

func (w *World) Dimension() world.Dimension {
	return w.dimension
}

func (w *World) SetDimension(dim world.Dimension) {
	w.dimension = dim

	w.dimRange = dim.Range()
	id, _ := world.DimensionID(dim)
	if d, ok := w.dimensionDefinitions[id]; ok {
		w.dimRange = cube.Range{
			int(d.MinimumY), int(d.MinimumY+d.HeightRange) - 1,
		}
	}
}

func (w *World) Range() cube.Range {
	return w.dimRange
}

func (w *World) SetTime(real time.Time, ingame int) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	w.timeSync = real
	w.time = ingame
}
func (w *World) SetWeather(rain, lightning float32) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if math.IsNaN(float64(rain)) || math.IsInf(float64(rain), 0) {
		rain = 0
	}
	if math.IsNaN(float64(lightning)) || math.IsInf(float64(lightning), 0) {
		lightning = 0
	}
	w.rainLevel = max(0, min(1, rain))
	w.lightningLevel = max(0, min(1, lightning))
	w.haveWeather = true
}
func (w *World) SetCommandsEnabled(enabled bool) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	w.commandsEnabled = &enabled
}

func (w *World) StoreChunk(pos world.ChunkPos, ch *Chunk) (err error) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	return w.storeChunkLocked(pos, ch)
}

func (w *World) storeChunkLocked(pos world.ChunkPos, ch *Chunk) (err error) {
	if w.discarded {
		return nil
	}
	if w.scene != nil {
		w.scene.Put(pos, ch.Chunk, ch.BlockEntities)
	}
	var empty = true
	for _, sub := range ch.Sub() {
		if !sub.Empty() {
			empty = false
			break
		}
	}
	if !empty {
		w.StoredChunks[pos] = struct{}{}

		w.onceOpen.Do(func() {
			if w.OnFirstChunk != nil {
				w.OnFirstChunk()
			}
			go func() {
				t := time.NewTicker(10 * time.Second)
				defer t.Stop()
				for {
					select {
					case <-w.ctx.Done():
						return
					case <-t.C:
						w.stateLock.Lock()
						w.applyBlockUpdates()
						w.storeMemToProvider()
						w.stateLock.Unlock()
					}
				}
			}()
		})
		w.onChunkUpdate(pos, ch.Chunk)
	}
	w.memState.StoreChunk(pos, ch)
	return nil
}

func (w *World) LoadChunk(pos world.ChunkPos) (*Chunk, bool, error) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	return w.loadChunkLocked(pos)
}

func (w *World) loadChunkLocked(pos world.ChunkPos) (*Chunk, bool, error) {
	if ch, ok := w.memState.chunks[pos]; ok {
		return ch, true, nil
	}

	if w.provider != nil {
		ch, err := w.provider.LoadColumn(pos, w.dimension)
		if err != nil {
			if errors.Is(err, leveldb.ErrNotFound) {
				return nil, false, nil
			}
			return nil, false, err
		}

		blockEntities := make(map[cube.Pos]map[string]any)
		for _, ent := range ch.BlockEntities {
			blockEntities[ent.Pos] = ent.Data
		}
		ret := &Chunk{
			Chunk:         ch.Chunk,
			BlockEntities: blockEntities,
		}
		w.memState.chunks[pos] = ret
		return ret, true, nil
	}

	return nil, false, nil
}

func (w *World) QueueBlockUpdate(pos protocol.BlockPos, ridTo uint32, layer uint8) {
	cp := world.ChunkPos{pos.X() >> 4, pos.Z() >> 4}
	w.blockUpdatesLock.Lock()
	defer w.blockUpdatesLock.Unlock()
	w.blockUpdates[cp] = append(w.blockUpdates[cp], blockUpdate{rid: ridTo, pos: pos, layer: layer})
}

func (w *World) SetBlockNBT(pos cube.Pos, nbt map[string]any, merge bool) error {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	chunkPos, _ := cubePosInChunk(pos)

	ch, ok, err := w.loadChunkLocked(chunkPos)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if merge {
		if pos.Y() < ch.Range().Min() || pos.Y() > ch.Range().Max() {
			return fmt.Errorf("block entity Y %d is outside captured terrain height", pos.Y())
		}
		if ch.BlockEntities == nil {
			ch.BlockEntities = make(map[cube.Pos]map[string]any)
		}
		if ch.BlockEntities[pos] == nil {
			if block, found := w.BlockRegistry.BlockByRuntimeID(ch.Block(uint8(pos.X()&15), int16(pos.Y()), uint8(pos.Z()&15), 0)); found {
				if native, ok := block.(world.NBTer); ok {
					ch.BlockEntities[pos] = native.EncodeNBT()
				}
			}
			if ch.BlockEntities[pos] == nil {
				ch.BlockEntities[pos] = map[string]any{}
			}
		}
		maps.Copy(ch.BlockEntities[pos], nbt)
	} else {
		ch.BlockEntities[pos] = nbt
	}

	return w.storeChunkLocked(chunkPos, ch)
}

func (w *World) StoreEntity(id entity.RuntimeID, es *entity.Entity) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if w.SkipCustomEntities && entity.IsCustomType(es.EntityType) {
		return
	}
	w.memState.StoreEntity(id, es)
	w.updateSceneActor(es)
}

func (w *World) StoreMap(m *packet.ClientBoundMapItemData) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	w.memState.StoreMap(m)
}

func (w *World) GetEntityRuntimeID(id entity.UniqueID) entity.RuntimeID {
	return w.memState.uniqueIDsToRuntimeIDs[id]
}

var ErrIgnoreEntity = errors.New("ignore entity")

func (w *World) ActEntity(id entity.RuntimeID, create bool, fn func(ent *entity.Entity) error) error {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	var new bool
	ent := w.memState.GetEntity(id)
	if ent == nil {
		if !create {
			return nil
		}
		ent = &entity.Entity{
			RuntimeID:            id,
			Inventory:            make(map[byte]map[byte]protocol.ItemInstance),
			Metadata:             make(map[uint32]any),
			Properties:           make(map[string]any),
			PropertiesOverridden: make(map[string]any),
			DeletedDistance:      -1,
		}
		new = true
	}
	err := fn(ent)
	if err == nil && w.SkipCustomEntities && entity.IsCustomType(ent.EntityType) {
		return nil
	}
	if err == ErrIgnoreEntity {
		return nil
	}
	if err != nil {
		return err
	}
	if new {
		w.memState.StoreEntity(id, ent)
	}
	w.updateSceneActor(ent)
	return nil
}

func (w *World) PlayerMove(playerPos mgl32.Vec3, entityRenderDistance float32, teleport int) {

	for _, ent := range w.memState.entities {
		if teleport > 0 && ent.DeletedDistance == -1 {
			ent.LastTeleport = teleport
		}

		if ent.DeletedDistance > 0 && entityRenderDistance != 0 {
			dist3d := ent.Position.Sub(playerPos).Len()
			if dist3d < entityRenderDistance+3 {
				ent.DeletedDistance = min(dist3d, ent.DeletedDistance)
			}
		}
	}
}

func entityCull(ent *entity.Entity, entityRenderDistance float32, diffOut *float32) bool {
	if ent.DeletedDistance > 0 && entityRenderDistance != 0 {
		diff := math.Abs(float64(ent.DeletedDistance - entityRenderDistance))
		return diff > 5
	}
	return false
}

func (w *World) EntityCounts(entityRenderDistance float32) (total, active int) {
	total = len(w.memState.entities)
	active = total
	for _, ent := range w.memState.entities {
		if entityCull(ent, entityRenderDistance, nil) {
			active -= 1
		}
	}
	return
}

func (w *World) AddEntityLink(el protocol.EntityLink) {
	w.memState.AddEntityLink(el)
	if w.scene != nil {
		w.scene.LinkActor(el.RiddenEntityUniqueID, el.RiderEntityUniqueID, el.Type != protocol.EntityLinkRemove)
	}
}

func blockPosInChunk(bp protocol.BlockPos) (x uint8, y int16, z uint8) {
	return uint8(bp.X() % 16), int16(bp.Y()), uint8(bp.Z() % 16)
}

func (w *World) BlockByID(rid uint32) (runtimeID uint32, name string, properties map[string]any, found bool) {
	if w.UseHashedRids {
		var ok bool
		rid, ok = w.BlockRegistry.HashToRuntimeID(rid)
		if !ok {
			w.log.Warn("couldnt find block hash for block update")
		}
	}
	name, properties, found = w.BlockRegistry.RuntimeIDToState(rid)
	return rid, name, properties, found
}

func (w *World) applyBlockUpdates() {
	w.blockUpdatesLock.Lock()
	defer w.blockUpdatesLock.Unlock()
	for pos, updates := range w.blockUpdates {
		delete(w.blockUpdates, pos)
		ch, ok, err := w.loadChunkLocked(world.ChunkPos(pos))
		if !ok {
			w.log.Warnf("Chunk updates for a chunk we dont have pos = %v", pos)
			continue
		}
		if err != nil {
			w.log.Warnf("Failed loading chunk %v %s", pos, err)
			continue
		}

		for _, update := range updates {
			x, y, z := blockPosInChunk(update.pos)
			ch.SetBlock(x, y, z, update.layer, update.rid)
		}
		err = w.storeChunkLocked(pos, ch)
		if err != nil {
			w.log.Warnf("Failed storing chunk %v %s", pos, err)
			continue
		}
	}
}

func (w *World) Open(name string, folder string) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	w.Name = name
	w.Folder = folder
	w.initScene()

	if w.opened {
		panic("trying to open already opened world")
	}
	w.opened = true
}

func (w *World) Rename(name, folder string) error {
	if w.Name == name && w.Folder == folder {
		return nil
	}
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	os.RemoveAll(folder)
	if w.provider != nil {
		err := w.provider.Close()
		if err != nil {
			return err
		}
		err = os.Rename(w.Folder, folder)
		if err != nil {
			return err
		}
		provider, err := mcdb.Config{
			Log: slog.Default(),
			LDBOptions: &opt.Options{
				Compression: opt.DefaultCompression,
			},
			Blocks: w.BlockRegistry,
		}.Open(folder)
		if err != nil {
			return err
		}
		w.provider = provider
	}
	if w.provider == nil {
		if _, err := os.Stat(w.Folder); err == nil {
			if err := os.Rename(w.Folder, folder); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	w.Folder = folder
	w.Name = name
	return nil
}

var errFinished = errors.New("finished")

func (w *World) Save(
	player proxy.Player, playerData map[string]any,
	behaviorPack *behaviourpack.Pack,
	excludedMobs []string,
	withPlayers bool, playerSkins map[uuid.UUID]*protocol.Skin,
	gameData minecraft.GameData, serverName string,
	entityCulling bool, entityRenderDistance float32,
) error {
	spawnPos := cube.Pos{int(player.Position.X()), int(player.Position.Y()), int(player.Position.Z())}
	behaviorPack = behaviorPack.WorldSnapshot(w.ID, withPlayers)

	var playerResourcePack *resourcepack.Pack
	if withPlayers {
		playerEntities := w.playersToEntities()

		for _, player := range playerEntities {
			behaviorPack.AddEntity(player.Identifier, nil, protocol.EntityMetadata{}, nil)
		}

		playerResourcePack = resourcepack.New()
		err := playerResourcePack.MakePlayers(playerEntities, playerSkins)
		if err != nil {
			return err
		}
	}

	err := w.finalizeProvider(
		playerData,
		excludedMobs,
		withPlayers,
		spawnPos,
		gameData,
		behaviorPack.HasContent(),
		entityCulling,
		entityRenderDistance,
	)
	if err != nil {
		return err
	}

	additionalPacks := func(fs utils.WriterFS) ([]addedPack, error) {
		var headers []addedPack
		for _, pack := range w.exportedResourcePacks {
			if err := behaviorPack.IncludeClientEntities(pack); err != nil {
				return nil, err
			}
		}

		if behaviorPack.HasContent() {
			packFolder := path.Join("behavior_packs", utils.FormatPackName(serverName))
			for _, p := range w.exportedResourcePacks {
				behaviorPack.CheckAddLink(p)
			}
			if err = behaviorPack.Save(utils.SubFS(fs, packFolder)); err != nil {
				return nil, err
			}
			headers = append(headers, addedPack{
				BehaviorPack: true,
				Header:       &behaviorPack.Manifest.Header,
			})
		}

		if playerResourcePack != nil {
			packFolder := path.Join("resource_packs", "CraftBox_players")
			if err := playerResourcePack.WriteToFS(utils.SubFS(fs, packFolder)); err != nil {
				return nil, err
			}
			headers = append(headers, addedPack{
				BehaviorPack: false,
				Header:       &playerResourcePack.Manifest.Header,
			})
		}

		return headers, nil
	}

	if err = w.finalizePacks(additionalPacks); err != nil {
		return err
	}

	return nil
}

func (w *World) finalizeProvider(
	playerData map[string]any,
	excludedMobs []string,
	withPlayers bool,
	spawn cube.Pos,
	gd minecraft.GameData,
	experimental bool,
	entityCulling bool,
	entityRenderDistance float32,
) error {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	w.cancelCtx(errFinished)

	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   w.ID,
		WorldName: w.Name,
		State:     "Storing Chunks",
	})

	w.applyBlockUpdates()
	err := w.storeMemToProvider()
	if err != nil {
		return err
	}

	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   w.ID,
		WorldName: w.Name,
		State:     "Storing Entities",
	})

	logrus.Tracef("entityRenderDistance: %.5f", entityRenderDistance)

	chunkEntities := make(map[world.ChunkPos][]chunk.Entity)
	for _, ent := range w.memState.entities {
		if ent.EntityType == "minecraft:player" || (!withPlayers && strings.HasPrefix(ent.EntityType, "CraftBox_player:")) {
			continue
		}
		if w.SkipCustomEntities && entity.IsCustomType(ent.EntityType) {
			continue
		}
		var ignore bool
		for _, ex := range excludedMobs {
			if ok, err := path.Match(ex, ent.EntityType); ok {
				w.log.Debugf("Excluding: %s %v", ent.EntityType, ent.Position)
				ignore = true
				break
			} else if err != nil {
				w.log.Warn(err)
			}
		}
		if ignore {
			continue
		}

		var diff float32
		shouldCull := entityCull(ent, entityRenderDistance, &diff)

		if shouldCull {
			if entityCulling {
				logrus.Tracef("dropping entity %s, dist: %.5f diff: %.5f", ent.EntityType, ent.DeletedDistance, diff)
				continue
			} else {
				ent.Tags = append(ent.Tags, "removed")
			}
		}
		cp := world.ChunkPos{int32(math.Floor(float64(ent.Position.X()) / 16)), int32(math.Floor(float64(ent.Position.Z()) / 16))}
		links := maps.Keys(w.memState.entityLinks[ent.UniqueID])
		stored := ent.ToChunkEntity(links)
		if w.scene != nil && w.scene.ActorExcluded(ent.UniqueID) {
			stored.Data["CraftBoxExcludeFrom3D"] = true
		}
		chunkEntities[cp] = append(chunkEntities[cp], stored)
	}

	for cp, v := range chunkEntities {
		err := w.provider.StoreEntities(cp, w.dimension, v)
		if err != nil {
			w.log.Error(err)
		}
	}

	playerData = maps.Clone(playerData)
	if playerData == nil {
		playerData = map[string]any{}
	}
	if _, present := playerData["DimensionId"]; !present {
		dimensionID, _ := world.DimensionID(w.dimension)
		playerData["DimensionId"] = int32(dimensionID)
	}
	if _, present := playerData["SpawnDimension"]; !present {
		playerData["SpawnDimension"] = playerData["DimensionId"]
	}
	err = w.provider.SaveLocalPlayerData(playerData)
	if err != nil {
		return err
	}

	ldb := w.provider.LDB()
	for id, m := range w.memState.maps {
		d, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
		if err != nil {
			return err
		}
		err = ldb.Put([]byte(fmt.Sprintf("map_%d", id)), d, nil)
		if err != nil {
			return err
		}
	}

	s := w.provider.Settings()
	s.Spawn = spawn
	s.Name = w.Name
	modeID := int(gd.WorldGameMode)
	if modeID == 6 {
		modeID = 3
	}
	if mode, ok := world.GameModeByID(modeID); ok {
		s.DefaultGameMode = mode
	}
	if difficulty, ok := world.DifficultyByID(int(gd.Difficulty)); ok {
		s.Difficulty = difficulty
	}

	ld := w.provider.LevelDat()
	ld.CheatsEnabled = true
	if w.commandsEnabled != nil {
		ld.CheatsEnabled = *w.commandsEnabled
		ld.CommandsEnabled = *w.commandsEnabled
	}
	ld.RandomSeed = int64(gd.WorldSeed)
	for _, gr := range gd.GameRules {
		switch strings.ToLower(gr.Name) {
		case "commandblockoutput":
			ld.CommandBlockOutput = gr.Value.(bool)
		case "maxcommandchainlength":
			ld.MaxCommandChainLength = int32(gr.Value.(uint32))
		case "commandblocksenabled":
			ld.CommandBlocksEnabled = gr.Value.(bool)
		case "dodaylightcycle":
			ld.DoDayLightCycle = gr.Value.(bool)
		case "doentitydrops":
			ld.DoEntityDrops = gr.Value.(bool)
		case "dofiretick":
			ld.DoFireTick = gr.Value.(bool)
		case "domobloot":
			ld.DoMobLoot = gr.Value.(bool)
		case "domobspawning":
			ld.DoMobSpawning = gr.Value.(bool)
		case "dotiledrops":
			ld.DoTileDrops = gr.Value.(bool)
		case "doweathercycle":
			ld.DoWeatherCycle = gr.Value.(bool)
		case "drowningdamage":
			ld.DrowningDamage = gr.Value.(bool)
		case "doinsomnia":
			ld.DoInsomnia = gr.Value.(bool)
		case "falldamage":
			ld.FallDamage = gr.Value.(bool)
		case "firedamage":
			ld.FireDamage = gr.Value.(bool)
		case "locatorbar":
			ld.LocatorBar = gr.Value.(bool)
		case "keepinventory":
			ld.KeepInventory = gr.Value.(bool)
		case "mobgriefing":
			ld.MobGriefing = gr.Value.(bool)
		case "pvp":
			ld.PVP = gr.Value.(bool)
		case "showcoordinates":
			ld.ShowCoordinates = gr.Value.(bool)
		case "naturalregeneration":
			ld.NaturalRegeneration = gr.Value.(bool)
		case "tntexplodes":
			ld.TNTExplodes = gr.Value.(bool)
		case "sendcommandfeedback":
			ld.SendCommandFeedback = gr.Value.(bool)
		case "randomtickspeed":
			ld.RandomTickSpeed = int32(gr.Value.(uint32))
		case "doimmediaterespawn":
			ld.DoImmediateRespawn = gr.Value.(bool)
		case "showdeathmessages":
			ld.ShowDeathMessages = gr.Value.(bool)
		case "functioncommandlimit":
			ld.FunctionCommandLimit = int32(gr.Value.(uint32))
		case "spawnradius":
			ld.SpawnRadius = int32(gr.Value.(uint32))
		case "showtags":
			ld.ShowTags = gr.Value.(bool)
		case "freezedamage":
			ld.FreezeDamage = gr.Value.(bool)
		case "respawnblocksexplode":
			ld.RespawnBlocksExplode = gr.Value.(bool)
		case "showbordereffect":
			ld.ShowBorderEffect = gr.Value.(bool)
		case "recipesunlock":
			ld.RecipesUnlock = gr.Value.(bool)
		case "dolimitedcrafting":
			ld.DoLimitedCrafting = gr.Value.(bool)
		case "showdaysplayed":
			ld.ShowDaysPlayed = gr.Value.(bool)
		case "showrecipemessages":
			ld.ShowRecipeMessages = gr.Value.(bool)
		case "playerssleepingpercentage":
			ld.PlayersSleepingPercentage = int32(gr.Value.(uint32))
		case "projectilescanbreakblocks":
			ld.ProjectilesCanBreakBlocks = gr.Value.(bool)
		case "tntexplosiondropdecay":
			ld.TNTExplosionDropDecay = gr.Value.(bool)
		default:
			w.log.Warn(locale.Loc("unknown_gamerule", locale.Strmap{"Name": gr.Name}))
		}
	}

	if w.VoidGen {
		ld.FlatWorldLayers = `{"biome_id":1,"block_layers":[{"block_data":0,"block_id":0,"count":1},{"block_data":0,"block_id":0,"count":2},{"block_data":0,"block_id":0,"count":1}],"encoding_version":3,"structure_options":null}`
		ld.Generator = 2
	}

	s.CurrentTick = gd.Time
	s.WeatherCycle = ld.DoWeatherCycle
	if w.haveWeather {
		s.Raining = w.rainLevel > 0
		s.Thundering = w.lightningLevel > 0
	}

	ticksSince := int64(time.Since(w.timeSync)/time.Millisecond) / 50
	s.Time = int64(w.time)
	s.TimeCycle = ld.DoDayLightCycle
	if ld.DoDayLightCycle {
		s.Time += ticksSince
		s.TimeCycle = true
	}

	if experimental {
		if ld.Experiments == nil {
			ld.Experiments = map[string]any{}
		}
		ld.Experiments["data_driven_items"] = true
		ld.Experiments["experiments_ever_used"] = true
		ld.Experiments["saved_with_toggled_experiments"] = true
		ld.Experiments["upcoming_creator_features"] = true
	}

	w.provider.SaveSettings(s)
	if w.haveWeather {
		ld.RainLevel = w.rainLevel
		ld.LightningLevel = w.lightningLevel
	}
	return w.provider.Close()
}
