package mcdb

import (
	"fmt"
	"testing"

	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestCapturedColumnHeightRoundTrip(t *testing.T) {
	for _, limits := range []cube.Range{{-64, 319}, {0, 255}, {80, 207}, {-128, 255}, {64, 447}} {
		t.Run(fmt.Sprint(limits), func(t *testing.T) {
			root := t.TempDir()
			reg := world.DefaultBlockRegistry
			reg.Finalize()
			db, err := (Config{Blocks: reg}).Open(root)
			if err != nil {
				t.Fatal(err)
			}
			ch := chunk.New(reg, limits)
			stone, _ := reg.StateToRuntimeID("minecraft:stone", nil)
			ch.SetBlock(1, int16(limits.Min()+1), 2, 0, stone)
			ch.SetBlock(3, int16(limits.Max()-1), 4, 0, stone)
			ch.SetBiome(1, int16(limits.Min()), 2, 42)
			ch.SetBiome(3, int16(limits.Max()), 4, 7)
			if err := db.StoreColumn(world.ChunkPos{}, world.Overworld, &chunk.Column{Chunk: ch}); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			db, err = (Config{Blocks: reg}).Open(root)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			col, err := db.LoadColumn(world.ChunkPos{}, world.Overworld)
			if err != nil {
				t.Fatal(err)
			}
			if got := col.Chunk.Range(); got != limits {
				t.Fatalf("decoded height %v, want %v", got, limits)
			}
			if col.Chunk.Block(1, int16(limits.Min()+1), 2, 0) != stone || col.Chunk.Block(3, int16(limits.Max()-1), 4, 0) != stone {
				t.Fatal("top or bottom block moved")
			}
			if col.Chunk.Biome(1, int16(limits.Min()), 2) != 42 || col.Chunk.Biome(3, int16(limits.Max()), 4) != 7 {
				t.Fatal("biomes shifted vertically")
			}
		})
	}
}

func TestSparseVanillaColumnKeepsDefaultHeight(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	db, err := (Config{Blocks: reg}).Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ch := chunk.New(reg, world.Overworld.Range())
	stone, _ := reg.StateToRuntimeID("minecraft:stone", nil)
	ch.SetBlock(1, 65, 2, 0, stone)
	if err := db.StoreColumn(world.ChunkPos{}, world.Overworld, &chunk.Column{Chunk: ch}); err != nil {
		t.Fatal(err)
	}
	key := dbKey{pos: world.ChunkPos{}, dim: world.Overworld}
	for y := -4; y < 20; y++ {
		if y != 4 {
			if err := db.ldb.Delete(key.Sum(keySubChunkData, byte(int8(y))), nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	col, err := db.LoadColumn(world.ChunkPos{}, world.Overworld)
	if err != nil {
		t.Fatal(err)
	}
	if col.Chunk.Range() != world.Overworld.Range() || col.Chunk.Block(1, 65, 2, 0) != stone {
		t.Fatal("sparse storage changed the world's vertical origin")
	}
}

func TestTruncatedBiomeDataIsNotAcceptedAsShorterWorld(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	db, err := (Config{Blocks: reg}).Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ch := chunk.New(reg, world.Overworld.Range())
	if err := db.StoreColumn(world.ChunkPos{}, world.Overworld, &chunk.Column{Chunk: ch}); err != nil {
		t.Fatal(err)
	}
	key := dbKey{pos: world.ChunkPos{}, dim: world.Overworld}.Sum(key3DData)
	data, err := db.ldb.Get(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ldb.Put(key, data[:len(data)-1], nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.LoadColumn(world.ChunkPos{}, world.Overworld); err == nil {
		t.Fatal("truncated biome sections were silently accepted")
	}
}

func TestInconsistentCaptureHeaderDoesNotChangeRange(t *testing.T) {
	for _, data := range [][]byte{{8, 0}, {9, 0, 1}} {
		t.Run(fmt.Sprint(data), func(t *testing.T) {
			reg := world.DefaultBlockRegistry
			reg.Finalize()
			db, err := (Config{Blocks: reg}).Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ch := chunk.New(reg, cube.Range{0, 255})
			if err := db.StoreColumn(world.ChunkPos{}, world.Overworld, &chunk.Column{Chunk: ch}); err != nil {
				t.Fatal(err)
			}
			key := dbKey{pos: world.ChunkPos{}, dim: world.Overworld}
			if err := db.ldb.Put(key.Sum(keySubChunkData, 0), data, nil); err != nil {
				t.Fatal(err)
			}
			biomes, err := db.biomes(key)
			if err != nil {
				t.Fatal(err)
			}
			limits, err := db.columnRange(key, biomes)
			if err != nil {
				t.Fatal(err)
			}
			if limits != world.Overworld.Range() {
				t.Fatal("legacy or inconsistent header was used to infer a new vertical origin")
			}
		})
	}
}
