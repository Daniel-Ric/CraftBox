package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"os"
	"testing"
)

func TestNativeDetailGallery(t *testing.T) {
	output := os.Getenv("CRAFT_BOX_NATIVE_GALLERY_OUT")
	if output == "" {
		t.Skip("optional visual fixture")
	}
	catalog, err := LoadCatalog(os.Getenv("CRAFT_BOX_3D_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	ch := chunk.New(reg, cube.Range{0, 15})
	put := func(x, y, z int, name string, props map[string]any) {
		rid, ok := reg.StateToRuntimeID("minecraft:"+name, props)
		if !ok {
			t.Fatal(name, props)
		}
		ch.SetBlock(uint8(x), int16(y), uint8(z), 0, rid)
	}
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			put(x, 0, z, "smooth_stone", nil)
		}
	}
	for i, facing := range []string{"north", "east", "south", "west"} {
		put(2+3*i, 1, 2, "anvil", map[string]any{"minecraft:cardinal_direction": facing})
	}
	for i, face := range []string{"up", "down", "east", "north"} {
		put(2+3*i, 2, 5, "amethyst_cluster", map[string]any{"minecraft:block_face": face})
	}
	entities := map[cube.Pos]map[string]any{}
	for i, plant := range []string{"blue_orchid", "poppy", "allium", "jungle_sapling"} {
		x := 2 + 3*i
		put(x, 1, 8, "flower_pot", map[string]any{"update_bit": false})
		entities[cube.Pos{x, 1, 8}] = map[string]any{"id": "FlowerPot", "PlantBlock": map[string]any{"name": "minecraft:" + plant}}
	}
	for i, name := range []string{"jungle_leaves", "oak_leaves", "jungle_sapling", "pale_oak_sapling"} {
		props := map[string]any{}
		if i < 2 {
			props = map[string]any{"persistent_bit": false, "update_bit": false}
		} else {
			props = map[string]any{"age_bit": false}
		}
		put(2+3*i, 1, 11, name, props)
	}
	for z := 2; z <= 10; z++ {
		name := "birch_fence_gate"
		props := map[string]any{"minecraft:cardinal_direction": "east", "open_bit": false, "in_wall_bit": false}
		if z == 2 || z == 10 {
			name = "birch_fence"
			props = nil
		}
		put(14, 1, z, name, props)
	}
	put(2, 1, 14, "birch_fence_gate", map[string]any{"minecraft:cardinal_direction": "north", "open_bit": true, "in_wall_bit": false})
	put(5, 1, 14, "birch_fence_gate", map[string]any{"minecraft:cardinal_direction": "east", "open_bit": true, "in_wall_bit": false})
	for y := 1; y <= 3; y++ {
		put(8, y, 14, "vine", map[string]any{"vine_direction_bits": int32(2)})
	}
	s := newTestStore(t, reg, catalog)
	s.Put(world.ChunkPos{}, ch, entities)
	snapshot, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Chunks[0].MissingTextures != 0 {
		t.Fatal(catalog.MissingTexturePaths())
	}
	page := Page{Sequence: snapshot.Sequence, Reset: true, Complete: true, Total: 1, Chunks: []PackedChunk{packChunk(snapshot.Chunks[0])}, AtlasRevision: catalog.revision, AtlasSize: atlasSize, Patches: catalog.atlasPatches(0), Source: "Crystal orientations, anvils, flower-pot NBT, saplings and mixed-mask jungle leaves"}
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
