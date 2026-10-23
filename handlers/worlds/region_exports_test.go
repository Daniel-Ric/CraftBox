package worlds

import (
	"archive/zip"
	"context"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sirupsen/logrus"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegionCaptureIndexesActorsAtNegativeFractionalCoordinates(t *testing.T) {
	messages.SetEventHandler(func(any) error { return nil })
	blocks := world.DefaultBlockRegistry.Clone()
	blocks.Finalize()
	stone, _ := blocks.StateToRuntimeID("minecraft:stone", map[string]any{})
	capture, err := worldstate.New(context.Background(), nil, func(world.ChunkPos, *chunk.Chunk) {})
	if err != nil {
		t.Fatal(err)
	}
	capture.SetDimension(world.Overworld)
	capture.BlockRegistry = blocks
	capture.SetTime(time.Now(), 0)
	capture.Open("test", t.TempDir())
	terrain := chunk.New(blocks, world.Overworld.Range())
	terrain.SetBlock(8, 70, 8, 0, stone)
	if err := capture.StoreChunk(world.ChunkPos{-1, 2}, &worldstate.Chunk{Chunk: terrain, BlockEntities: map[cube.Pos]map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	capture.StoreEntity(1, &entity.Entity{RuntimeID: 1, UniqueID: 1, EntityType: "minecraft:pig", Position: mgl32.Vec3{-.5, 71, 40}, Metadata: protocol.NewEntityMetadata()})
	if err := capture.Save(proxy.Player{Position: mgl32.Vec3{-.5, 71, 40}}, map[string]any{"Pos": []float32{-.5, 71, 40}}, behaviourpack.New("test"), nil, false, nil, minecraft.GameData{}, "test", false, 0); err != nil {
		t.Fatal(err)
	}
	db, err := mcdb.Config{Blocks: blocks}.Open(capture.Folder)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	column, err := db.LoadColumn(world.ChunkPos{-1, 2}, world.Overworld)
	if err != nil {
		t.Fatal(err)
	}
	if len(column.Entities) != 1 {
		t.Fatal("actor was assigned to another area at a negative fractional coordinate")
	}
}

func TestRegionExportsSeparateRealArchivesImagesInventoriesAndSpawn(t *testing.T) {
	t.Run("move", func(t *testing.T) { testRegionExports(t, true) })
	t.Run("copy", func(t *testing.T) { testRegionExports(t, false) })
}

func TestRegionSpawnStaysBelowNetherRoof(t *testing.T) {
	blocks := world.DefaultBlockRegistry.Clone()
	blocks.Finalize()
	stone, _ := blocks.StateToRuntimeID("minecraft:stone", map[string]any{})
	folder := t.TempDir()
	source := &worldstate.World{BlockRegistry: blocks}
	source.SetDimension(world.Nether)
	db, err := mcdb.Config{Blocks: blocks}.Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	terrain := chunk.New(blocks, world.Nether.Range())
	terrain.SetBlock(8, 60, 8, 0, stone)
	terrain.SetBlock(8, 127, 8, 0, stone)
	if err := db.StoreColumn(world.ChunkPos{0, 0}, world.Nether, &chunk.Column{Chunk: terrain}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveLocalPlayerData(map[string]any{"Pos": []float32{100, 61, 100}}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := normalizeRegionSpawn(folder, "Nether region", source, proxy.Player{Position: mgl32.Vec3{100, 61, 100}}, map[world.ChunkPos]struct{}{{0, 0}: {}}); err != nil {
		t.Fatal(err)
	}
	db, err = mcdb.Config{Blocks: blocks}.Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := db.Settings().Spawn; got != (cube.Pos{8, 61, 8}) {
		t.Fatalf("unsafe Nether spawn: %v", got)
	}
	raw, err := db.LDB().Get([]byte("~local_player"), nil)
	if err != nil {
		t.Fatal(err)
	}
	var local map[string]any
	if err := nbt.UnmarshalEncoding(raw, &local, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	pos := local["Pos"].([]any)
	if pos[0] != float32(8.5) || pos[2] != float32(8.5) {
		t.Fatalf("player must be centred in the free block: %v", pos)
	}
}

func TestRegionSpawnRejectsAreaWithoutStandingSpace(t *testing.T) {
	blocks := world.DefaultBlockRegistry.Clone()
	blocks.Finalize()
	stone, _ := blocks.StateToRuntimeID("minecraft:stone", map[string]any{})
	source := &worldstate.World{BlockRegistry: blocks}
	source.SetDimension(world.Nether)
	db, err := mcdb.Config{Blocks: blocks}.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	terrain := chunk.New(blocks, world.Nether.Range())
	for x := uint8(0); x < 16; x++ {
		for z := uint8(0); z < 16; z++ {
			for y := int16(0); y < 128; y++ {
				terrain.SetBlock(x, y, z, 0, stone)
			}
		}
	}
	if err := db.StoreColumn(world.ChunkPos{0, 0}, world.Nether, &chunk.Column{Chunk: terrain}); err != nil {
		t.Fatal(err)
	}
	if _, err := findRegionSpawn(db, source, proxy.Player{}, map[world.ChunkPos]struct{}{{0, 0}: {}}); err == nil {
		t.Fatal("solid area must not produce a spawn outside the dimension")
	}
}

func testRegionExports(t *testing.T, move bool) {
	messages.SetEventHandler(func(any) error { return nil })
	sourceFolder := filepath.Join(t.TempDir(), "Lobby--source")
	blocks := world.DefaultBlockRegistry.Clone()
	blocks.Finalize()
	stone, found := blocks.StateToRuntimeID("minecraft:stone", map[string]any{})
	if !found {
		t.Fatal("stone definition missing")
	}
	source := &worldstate.World{ID: "source", Name: "Lobby", Folder: sourceFolder, BlockRegistry: blocks, StoredChunks: map[world.ChunkPos]struct{}{{-1, 2}: {}, {9, 2}: {}}}
	source.SetDimension(world.Overworld)
	db, err := mcdb.Config{Blocks: blocks}.Open(sourceFolder)
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range []world.ChunkPos{{-1, 2}, {9, 2}} {
		terrain := chunk.New(blocks, world.Overworld.Range())
		terrain.SetBlock(8, 70, 8, 0, stone)
		column := &chunk.Column{Chunk: terrain, BlockEntities: []chunk.BlockEntity{{Pos: cube.Pos{int(pos[0])*16 + 8, 71, 40}, Data: map[string]any{"id": "Chest", "x": pos[0]*16 + 8, "y": int32(71), "z": int32(40), "Items": []map[string]any{{"Name": "minecraft:diamond", "Count": uint8(2)}}}}}}
		if err := db.StoreColumn(pos, world.Overworld, column); err != nil {
			t.Fatal(err)
		}
		if err := db.StoreEntities(pos, world.Overworld, []chunk.Entity{{ID: int64(pos[0] + 100), Data: map[string]any{"identifier": "minecraft:zombie", "Pos": []float32{float32(pos[0]*16 + 8), 71, 40}, "Inventory": []map[string]any{{"Name": "minecraft:diamond", "Count": uint8(4)}}}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.SaveLocalPlayerData(map[string]any{"Pos": []float32{-8, 80, 40}, "SpawnX": int32(-8), "SpawnY": int32(80), "SpawnZ": int32(40), "Inventory": []map[string]any{{"Name": "minecraft:diamond", "Count": uint8(3)}}}); err != nil {
		t.Fatal(err)
	}
	db.LevelDat().KeepInventory = true
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceFolder, "resource_packs", "custom"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceFolder, "resource_packs", "custom", "manifest.json"), []byte("preserved pack"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &worldsHandler{session: &proxy.Session{}, log: logrus.NewEntry(logrus.New()), settings: WorldSettings{SaveImage: true}}
	w.mapUI = NewMapUI(w)
	close(w.mapUI.haveColors)
	for index, pos := range []protocol.ChunkPos{{-1, 2}, {9, 2}} {
		tile := image.NewRGBA(image.Rect(0, 0, 16, 16))
		pixel := []color.RGBA{{R: 255, A: 255}, {B: 255, A: 255}}[index]
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				tile.SetRGBA(x, y, pixel)
			}
		}
		w.mapUI.renderedChunks[pos] = tile
	}
	region := messages.WorldRegion{ID: "region", ParentID: source.ID, Name: "Island", Move: move, Bounds: messages.ChunkBounds{MinX: -1, MaxX: -1, MinZ: 2, MaxZ: 2}}
	if err := w.saveRegionExports(source, proxy.Player{Position: mgl32.Vec3{-8, 80, 40}}, []messages.WorldRegion{region}); err != nil {
		t.Fatal(err)
	}
	w.wg.Wait()
	for _, tc := range []struct {
		folder       string
		pos, outside world.ChunkPos
		pixel        color.RGBA
	}{{sourceFolder, world.ChunkPos{9, 2}, world.ChunkPos{-1, 2}, color.RGBA{B: 255, A: 255}}, {filepath.Join(filepath.Dir(sourceFolder), "Island--region"), world.ChunkPos{-1, 2}, world.ChunkPos{9, 2}, color.RGBA{R: 255, A: 255}}} {
		archive, err := zip.OpenReader(tc.folder + ".mcworld")
		if err != nil {
			t.Fatal(err)
		}
		archive.Close()
		db, err := mcdb.Config{Blocks: blocks}.Open(tc.folder)
		if err != nil {
			t.Fatal(err)
		}
		column, err := db.LoadColumn(tc.pos, world.Overworld)
		if err != nil {
			db.Close()
			t.Fatal(err)
		}
		if len(column.BlockEntities) != 1 || len(column.Entities) != 1 {
			db.Close()
			t.Fatalf("missing block/actor inventory: %+v", column)
		}
		isOriginal := tc.folder == sourceFolder
		if _, err := db.LoadColumn(tc.outside, world.Overworld); (err == nil) != (isOriginal && !move) {
			db.Close()
			t.Fatal("other area's terrain retained")
		}
		spawn := db.Settings().Spawn
		spawnChunk := world.ChunkPos{int32(spawn.X() >> 4), int32(spawn.Z() >> 4)}
		if spawnChunk != tc.pos && !(isOriginal && !move && spawnChunk == tc.outside) {
			db.Close()
			t.Fatal("spawn outside retained terrain")
		}
		raw, err := db.LDB().Get([]byte("~local_player"), nil)
		if err != nil {
			db.Close()
			t.Fatal(err)
		}
		var player map[string]any
		if err := nbt.UnmarshalEncoding(raw, &player, nbt.LittleEndian); err != nil {
			db.Close()
			t.Fatal(err)
		}
		if player["SpawnX"] != int32(spawn.X()) || player["SpawnY"] != int32(spawn.Y()) || player["SpawnZ"] != int32(spawn.Z()) {
			db.Close()
			t.Fatal("respawn fields disagree")
		}
		if _, ok := player["Inventory"]; !ok {
			db.Close()
			t.Fatal("local player inventory lost")
		}
		if !db.LevelDat().KeepInventory {
			db.Close()
			t.Fatal("game rule lost")
		}
		db.Close()
		file, err := os.Open(tc.folder + ".png")
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		width, pixelX := 16, 0
		if isOriginal && !move {
			width, pixelX = 176, 160
		}
		if img.Bounds().Dx() != width || img.Bounds().Dy() != 16 || color.RGBAModel.Convert(img.At(pixelX, 0)) != tc.pixel {
			t.Fatal("PNG mixes regions")
		}
		if data, err := os.ReadFile(filepath.Join(tc.folder, "resource_packs", "custom", "manifest.json")); err != nil || string(data) != "preserved pack" {
			t.Fatal("pack not preserved")
		}
	}
}
