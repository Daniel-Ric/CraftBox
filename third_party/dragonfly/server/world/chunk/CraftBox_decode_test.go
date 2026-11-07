package chunk_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestHistoricalDiskSubChunkFormatsAndLegacyPaletteMappings(t *testing.T) {
	r := world.DefaultBlockRegistry
	r.Finalize()
	for _, version := range []byte{1, 8, 9} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			var data bytes.Buffer
			data.WriteByte(version)
			if version != 1 {
				data.WriteByte(1)
			}
			if version == 9 {
				data.WriteByte(0)
			}
			data.WriteByte(0)
			if err := nbt.NewEncoderWithEncoding(&data, nbt.LittleEndian).Encode(map[string]any{"name": "minecraft:wool", "val": int16(14), "version": int32(0)}); err != nil {
				t.Fatal(err)
			}
			decoded, err := chunk.DiskDecode(r, chunk.SerialisedData{SubChunks: [][]byte{data.Bytes()}}, cube.Range{0, 15})
			if err != nil {
				t.Fatal(err)
			}
			name, _, _ := r.RuntimeIDToState(decoded.Block(3, 7, 9, 0))
			if name != "minecraft:red_wool" {
				t.Fatal("legacy wool metadata mapped incorrectly", name)
			}
		})
	}
}

func TestDiskPaletteRejectsOversizedCountBeforeReadingNBT(t *testing.T) {
	r := world.DefaultBlockRegistry
	r.Finalize()
	raw := []byte{9, 1, 0, 2}
	raw = append(raw, make([]byte, 128*4)...)
	raw = binary.LittleEndian.AppendUint32(raw, 65537)
	if _, err := chunk.DiskDecode(r, chunk.SerialisedData{SubChunks: [][]byte{raw}}, cube.Range{0, 15}); err == nil || !strings.Contains(err.Error(), "palette entry count") {
		t.Fatalf("palette count was not bounded before decoding: %v", err)
	}
}

func TestDiskPaletteRejectsIndicesOutsidePalette(t *testing.T) {
	r := world.DefaultBlockRegistry
	r.Finalize()
	var data bytes.Buffer
	data.Write([]byte{9, 1, 0, 2})
	for i := 0; i < 128; i++ {
		binary.Write(&data, binary.LittleEndian, uint32(0xffffffff))
	}
	binary.Write(&data, binary.LittleEndian, uint32(1))
	if err := nbt.NewEncoderWithEncoding(&data, nbt.LittleEndian).Encode(map[string]any{"name": "minecraft:stone", "states": map[string]any{}, "version": int32(chunk.CurrentBlockVersion)}); err != nil {
		t.Fatal(err)
	}
	if _, err := chunk.DiskDecode(r, chunk.SerialisedData{SubChunks: [][]byte{data.Bytes()}}, cube.Range{0, 15}); err == nil {
		t.Fatal("out-of-range palette indices accepted")
	}
}

func TestDiskSubChunkUsesEncodedVerticalIndex(t *testing.T) {
	br := world.DefaultBlockRegistry
	br.Finalize()
	stone, _ := br.StateToRuntimeID("minecraft:stone", nil)
	source := chunk.New(br, cube.Range{-64, 319})
	source.SetBlock(2, 17, 3, 0, stone)
	data := chunk.SerialisedData{SubChunks: [][]byte{chunk.EncodeSubChunk(source, chunk.DiskEncoding, 5)}}
	decoded, err := chunk.DiskDecode(br, data, cube.Range{-64, 319})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Block(2, 17, 3, 0) != stone || decoded.Block(2, -63, 3, 0) != br.AirRuntimeID() {
		t.Fatal("encoded Y index ignored; block moved vertically")
	}
}

func TestDiskSubChunkRejectsOutOfRangeAndDuplicateIndices(t *testing.T) {
	br := world.DefaultBlockRegistry
	br.Finalize()
	for _, subs := range [][][]byte{{{9, 0, 127}}, {{9, 0, 0}, {9, 0, 0}}} {
		if _, err := chunk.DiskDecode(br, chunk.SerialisedData{SubChunks: subs}, cube.Range{-64, 319}); err == nil {
			t.Fatalf("accepted invalid sub-chunks %#v", subs)
		}
	}
}

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
