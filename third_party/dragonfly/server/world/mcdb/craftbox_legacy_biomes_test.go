package mcdb

import (
	"bytes"
	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

func TestLegacy2DBiomesRetainEveryHorizontalColumn(t *testing.T) {
	db, err := Config{}.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := dbKey{pos: world.ChunkPos{-1, 2}, dim: world.Overworld}
	raw := append(make([]byte, 512), bytes.Repeat([]byte{1}, 256)...)
	raw[512+3+4*16] = 2
	ch := chunk.New(world.DefaultBlockRegistry, cube.Range{0, 255})
	stone, _ := world.DefaultBlockRegistry.StateToRuntimeID("minecraft:stone", nil)
	ch.SetBlock(3, 70, 4, 0, stone)
	data := chunk.Encode(ch, chunk.DiskEncoding)
	if err = db.ldb.Put(key.Sum(keyVersionOld), []byte{19}, nil); err != nil {
		t.Fatal(err)
	}
	if err = db.ldb.Put(key.Sum(key2DData), raw, nil); err != nil {
		t.Fatal(err)
	}
	for i, sub := range data.SubChunks {
		if err = db.ldb.Put(key.Sum(keySubChunkData, byte(i)), sub, nil); err != nil {
			t.Fatal(err)
		}
	}
	col, err := db.LoadColumn(key.pos, world.Overworld)
	if err != nil {
		t.Fatal(err)
	}
	if col.Chunk.Range() != (cube.Range{0, 255}) {
		t.Fatal("legacy range", col.Chunk.Range())
	}
	for _, y := range []int16{0, 70, 255} {
		if col.Chunk.Biome(3, y, 4) != 2 || col.Chunk.Biome(4, y, 3) != 1 {
			t.Fatal("legacy biome axes/height lost", y)
		}
	}
	if col.Chunk.Block(3, 70, 4, 0) != stone {
		t.Fatal("legacy terrain moved")
	}
}
func TestFutureChunkVersionAndTruncated2DDataFailExplicitly(t *testing.T) {
	for _, future := range []bool{true, false} {
		db, err := Config{}.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		key := dbKey{dim: world.Overworld}
		version := byte(19)
		if future {
			version = chunkVersion + 1
		}
		db.ldb.Put(key.Sum(keyVersion), []byte{version}, nil)
		if !future {
			db.ldb.Put(key.Sum(key2DData), []byte{1, 2, 3}, nil)
		}
		if _, err = db.LoadColumn(world.ChunkPos{}, world.Overworld); err == nil {
			t.Fatal("invalid world format silently accepted")
		}
		db.Close()
	}
}
