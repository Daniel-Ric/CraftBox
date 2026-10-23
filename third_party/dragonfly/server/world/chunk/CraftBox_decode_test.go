package chunk_test

import (
	"bytes"
	"testing"

	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestCapturedChunkPreservesBlocksAndBlockNBT(t *testing.T) {
	br := world.DefaultBlockRegistry
	br.Finalize()
	stone, ok := br.StateToRuntimeID("minecraft:stone", nil)
	if !ok {
		t.Fatal("stone missing")
	}
	c := chunk.New(br, cube.Range{0, 15})
	c.SetBlock(1, 5, 2, 0, stone)
	encoded := chunk.Encode(c, chunk.NetworkEncoding)
	var payload bytes.Buffer
	payload.Write(encoded.SubChunks[0])
	payload.Write(encoded.Biomes)
	payload.WriteByte(0)
	if err := nbt.NewEncoderWithEncoding(&payload, nbt.NetworkLittleEndian).Encode(map[string]any{"x": int32(1), "y": int32(5), "z": int32(2), "id": "Chest"}); err != nil {
		t.Fatal(err)
	}
	decoded, entities, err := chunk.DecodeCapturedNetwork(br, payload.Bytes(), 1, cube.Range{0, 15}, false)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Block(1, 5, 2, 0) != stone {
		t.Fatal("block changed")
	}
	if len(entities) != 1 || entities[0]["id"] != "Chest" {
		t.Fatalf("block NBT = %#v", entities)
	}
}

func TestCapturedSubChunkResolvesHashedRuntimeID(t *testing.T) {
	br := world.DefaultBlockRegistry
	br.Finalize()
	stone, _ := br.StateToRuntimeID("minecraft:stone", nil)
	hash, ok := br.RuntimeIDToHash(stone)
	if !ok {
		t.Fatal("stone hash missing")
	}
	buf := bytes.NewBuffer([]byte{9, 1, 0, 1})
	if err := protocol.WriteVarint32(buf, int32(hash)); err != nil {
		t.Fatal(err)
	}
	index := byte(0)
	sub, err := chunk.DecodeCapturedSubChunk(buf, br, cube.Range{0, 15}, &index, chunk.NetworkEncoding, true)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Layer(0).At(3, 4, 5) != stone {
		t.Fatal("hash not converted to runtime ID")
	}
}

func TestCapturedSubChunkReadsPersistentBlockPalette(t *testing.T) {
	br := world.DefaultBlockRegistry
	br.Finalize()
	stone, _ := br.StateToRuntimeID("minecraft:stone", nil)
	buf := bytes.NewBuffer([]byte{9, 1, 0, 0})
	state := map[string]any{"name": "minecraft:stone", "states": map[string]any{}, "version": chunk.CurrentBlockVersion}
	if err := nbt.NewEncoderWithEncoding(buf, nbt.NetworkLittleEndian).Encode(state); err != nil {
		t.Fatal(err)
	}
	index := byte(0)
	sub, err := chunk.DecodeCapturedSubChunk(buf, br, cube.Range{0, 15}, &index, chunk.NetworkEncoding, true)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Layer(0).At(3, 4, 5) != stone {
		t.Fatal("persistent state was interpreted as runtime hash")
	}
}
