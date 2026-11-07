package worlds

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
)

func copyRegionFiles(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("Unsupported file in world folder")
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.Create(target)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		return errors.Join(copyErr, output.Close())
	})
}

func (w *worldsHandler) saveRegionExports(source *worldstate.World, player proxy.Player, regions []messages.WorldRegion) error {
	dimension, _ := world.DimensionID(source.Dimension())
	type preparedRegion struct {
		state    *worldstate.World
		region   messages.WorldRegion
		entities int
	}
	prepared := make([]preparedRegion, 0, len(regions))

	for _, region := range regions {
		if messages.WorldDeleted(region.ID) {
			continue
		}
		selected := make(map[world.ChunkPos]struct{})
		for pos := range source.StoredChunks {
			if region.Bounds.Contains(pos[0], pos[1]) {
				selected[pos] = struct{}{}
			}
		}
		if len(selected) == 0 {
			messages.SendEvent(&messages.EventProcessingWorldUpdate{WorldID: region.ID, WorldName: region.Name, State: "Leer"})
			continue
		}
		folder := filepath.Join(filepath.Dir(source.Folder), utils.MakeValidFilename(region.Name)+"--"+region.ID)
		messages.RecordCaptureFolder(region.ID, folder)
		if err := copyRegionFiles(source.Folder, folder); err != nil {
			return err
		}
		points := messages.CapturePointsInBounds(messages.ReadCapturePoints(source.Folder), region.Bounds)
		if data, e := json.MarshalIndent(points, "", "  "); e != nil {
			return e
		} else if e = os.WriteFile(filepath.Join(folder, "craftbox-capture.json"), data, 0600); e != nil {
			return e
		}
		count, err := filterRegionDatabase(filepath.Join(folder, "db"), func(x, z, dim int32) bool { return dim == int32(dimension) && region.Bounds.Contains(x, z) })
		if err != nil {
			return err
		}
		if err := normalizeRegionSpawn(folder, region.Name, source, player, selected); err != nil {
			return err
		}
		state := &worldstate.World{ID: region.ID, Name: region.Name, Folder: folder, StoredChunks: selected}
		prepared = append(prepared, preparedRegion{state, region, count})
	}
	keepSource := func(x, z, dim int32) bool {
		for _, region := range regions {
			if region.Move && dim == int32(dimension) && region.Bounds.Contains(x, z) {
				return false
			}
		}
		return true
	}
	remaining := make(map[world.ChunkPos]struct{})
	for pos := range source.StoredChunks {
		if keepSource(pos[0], pos[1], int32(dimension)) {
			remaining[pos] = struct{}{}
		}
	}

	if len(remaining) > 0 {
		if err := normalizeRegionSpawn(source.Folder, source.Name, source, player, remaining); err != nil {
			return err
		}
	}
	count, err := filterRegionDatabase(filepath.Join(source.Folder, "db"), keepSource)
	if err != nil {
		return err
	}
	for pos := range source.StoredChunks {
		if !keepSource(pos[0], pos[1], int32(dimension)) {
			delete(source.StoredChunks, pos)
		}
	}
	if len(source.StoredChunks) > 0 {
		if w.settings.SaveImage {
			if err := w.saveMapImageWhere(source, func(x, z int32) bool { return keepSource(x, z, int32(dimension)) }); err != nil {
				return err
			}
		}
		w.queueRegionArchive(source, count)
	} else {
		messages.SendEvent(&messages.EventProcessingWorldUpdate{WorldID: source.ID, WorldName: source.Name, State: "Leer"})
	}
	for _, item := range prepared {
		if w.settings.SaveImage {
			if err := w.saveMapImageWhere(item.state, item.region.Bounds.Contains); err != nil {
				return err
			}
		}
		w.queueRegionArchive(item.state, item.entities)
	}
	return nil
}

func normalizeRegionSpawn(folder, name string, source *worldstate.World, player proxy.Player, selected map[world.ChunkPos]struct{}) (result error) {
	db, err := mcdb.Config{Blocks: source.BlockRegistry}.Open(folder)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, db.Close()) }()
	spawn := cube.Pos{int(math.Floor(float64(player.Position.X()))), int(math.Floor(float64(player.Position.Y()))), int(math.Floor(float64(player.Position.Z())))}
	_, inside := selected[world.ChunkPos{int32(spawn.X() >> 4), int32(spawn.Z() >> 4)}]
	inside = inside && spawn.Y() >= source.Dimension().Range()[0] && spawn.Y()+1 <= source.Dimension().Range()[1]
	if !inside {
		spawn, err = findRegionSpawn(db, source, player, selected)
		if err != nil {
			return err
		}
	}
	settings := db.Settings()
	settings.Name = name
	settings.Spawn = spawn
	db.SaveSettings(settings)
	raw, err := db.LDB().Get([]byte("~local_player"), nil)
	if err != nil {
		return err
	}
	var local map[string]any
	if err := nbt.UnmarshalEncoding(raw, &local, nbt.LittleEndian); err != nil {
		return err
	}
	if !inside {
		local["Pos"] = []float32{float32(spawn.X()) + 0.5, float32(spawn.Y()), float32(spawn.Z()) + 0.5}
	}
	local["SpawnX"], local["SpawnY"], local["SpawnZ"] = int32(spawn.X()), int32(spawn.Y()), int32(spawn.Z())
	return db.SaveLocalPlayerData(local)
}

func findRegionSpawn(db *mcdb.DB, source *worldstate.World, player proxy.Player, selected map[world.ChunkPos]struct{}) (cube.Pos, error) {
	positions := make([]world.ChunkPos, 0, len(selected))
	for pos := range selected {
		positions = append(positions, pos)
	}
	distance := func(pos world.ChunkPos) float64 {
		dx, dz := float64(pos[0])*16+8-float64(player.Position.X()), float64(pos[1])*16+8-float64(player.Position.Z())
		return dx*dx + dz*dz
	}
	sort.Slice(positions, func(i, j int) bool {
		a, b := positions[i], positions[j]
		if da, db := distance(a), distance(b); da != db {
			return da < db
		}
		return a[0] < b[0] || (a[0] == b[0] && a[1] < b[1])
	})
	air := source.BlockRegistry.AirRuntimeID()
	for _, pos := range positions {
		column, err := db.LoadColumn(pos, source.Dimension())
		if err != nil {
			return cube.Pos{}, err
		}
		terrain := column.Chunk

		for index := -1; index < 256; index++ {
			x, z := uint8(8), uint8(8)
			if index >= 0 {
				x, z = uint8(index%16), uint8(index/16)
			}
			for y := terrain.Range()[1] - 1; y > terrain.Range()[0]; y-- {
				floor := terrain.Block(x, int16(y-1), z, 0)
				if floor == air || source.BlockRegistry.LiquidBlock(floor) || source.BlockRegistry.FilteringBlock(floor) < 15 {
					continue
				}
				clear := true
				for layer := uint8(0); layer < 2; layer++ {
					if terrain.Block(x, int16(y), z, layer) != air || terrain.Block(x, int16(y+1), z, layer) != air {
						clear = false
						break
					}
				}
				if clear {
					return cube.Pos{int(pos[0])*16 + int(x), y, int(pos[1])*16 + int(z)}, nil
				}
			}
		}
	}
	return cube.Pos{}, errors.New("The selected region has no clear spawn point with solid ground and two air blocks above it. Select a larger region.")
}

func (w *worldsHandler) queueRegionArchive(state *worldstate.World, entities int) {
	if err := applyCaptureMetadata(state); err != nil {
		w.log.Error(err)
		messages.SendEvent(&messages.EventProcessingWorldUpdate{WorldID: state.ID, WorldName: state.Name, State: "Saving failed"})
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		if err := w.writeWorldArchive(state, entities, entities); err != nil {
			w.log.Error(err)
			messages.SendEvent(&messages.EventProcessingWorldUpdate{WorldID: state.ID, WorldName: state.Name, State: "Saving failed"})
		}
	}()
}
func applyCaptureMetadata(state *worldstate.World) error {
	edit := messages.WorldMetadata(state.ID)
	if edit.Name == nil && edit.Spawn == nil && edit.Export == nil {
		return nil
	}
	if edit.Name != nil {
		state.Name = *edit.Name
	}
	options := worldexport.Options{}
	if edit.Export != nil {
		options = *edit.Export
	}
	options.Settings.Name = &state.Name
	options.Settings.Spawn = edit.Spawn
	return worldexport.UpdateMetadata(context.Background(), state.Folder, options)
}
