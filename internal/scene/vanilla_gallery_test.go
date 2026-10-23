package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"os"
	"testing"
)

func TestVanillaMaterialGallery(t *testing.T) {
	output := os.Getenv("CRAFT_BOX_VANILLA_GALLERY_OUT")
	if output == "" {
		t.Skip("set CRAFT_BOX_VANILLA_GALLERY_OUT to record the visual fixture")
	}
	c, err := LoadCatalog(os.Getenv("CRAFT_BOX_3D_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	ch := chunk.New(reg, cube.Range{0, 15})
	put := func(x, y, z int, name string, props map[string]any) {
		rid, ok := reg.StateToRuntimeID("minecraft:"+name, props)
		if !ok {
			t.Fatalf("unknown fixture block %s %v", name, props)
		}
		ch.SetBlock(uint8(x), int16(y), uint8(z), 0, rid)
	}
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			put(x, 0, z, "grass_block", nil)
		}
	}
	for i, name := range []string{"sunflower", "rose_bush", "peony", "lilac", "tall_grass", "large_fern"} {
		put(1+i*2, 1, 1, name, map[string]any{"upper_block_bit": false})
		put(1+i*2, 2, 1, name, map[string]any{"upper_block_bit": true})
	}
	for i := 0; i < 4; i++ {
		put(1+i*3, 1, 4, "trapdoor", map[string]any{"direction": int32(i), "open_bit": i >= 2, "upside_down_bit": i%2 != 0})
	}
	for i, name := range []string{"oak_leaves", "birch_leaves", "spruce_leaves", "azalea_leaves", "pale_oak_leaves"} {
		put(1+i*3, 1, 7, name, nil)
	}
	for x := 10; x < 15; x++ {
		for z := 10; z < 15; z++ {
			put(x, 1, z, "water", map[string]any{"liquid_depth": int32(0)})
		}
	}
	for x := 1; x < 8; x++ {
		put(x, 1, 10, "oak_fence", nil)
		put(x, 1, 13, "glass_pane", nil)
	}
	put(14, 1, 1, "light_block_15", nil)
	put(2, 1, 6, "lantern", map[string]any{"hanging": false})
	put(5, 3, 6, "lantern", map[string]any{"hanging": true})
	put(8, 1, 6, "soul_lantern", map[string]any{"hanging": false})
	put(11, 3, 6, "soul_lantern", map[string]any{"hanging": true})
	put(2, 1, 9, "wooden_pressure_plate", map[string]any{"redstone_signal": int32(0)})
	put(5, 1, 9, "stone_pressure_plate", map[string]any{"redstone_signal": int32(15)})
	for y := 2; y <= 5; y++ {
		put(13, y, 12, "flowing_water", map[string]any{"liquid_depth": int32(8)})
	}
	s := newTestStore(t, reg, c)
	s.Put(world.ChunkPos{}, ch)
	snapshot, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Chunks[0].MissingTextures != 0 {
		t.Fatalf("fixture missing textures: %v", c.MissingTexturePaths())
	}
	page := Page{Sequence: snapshot.Sequence, Reset: true, Complete: true, Total: 1, Chunks: []PackedChunk{packChunk(snapshot.Chunks[0])}, AtlasRevision: c.revision, AtlasSize: atlasSize, Patches: c.atlasPatches(0), Source: "Flowers, trapdoors, foliage, fences, glass and water verification"}
	data, err := json.Marshal([]Page{page})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(output, data, 0600); err != nil {
		t.Fatal(err)
	}
	glb, err := EncodeGLB(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(output+".glb", glb, 0600); err != nil {
		t.Fatal(err)
	}
}
