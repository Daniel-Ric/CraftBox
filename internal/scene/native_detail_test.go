package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

func TestCrystalAndSaplingModelsAreCutoutPlanes(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for _, name := range []string{"amethyst_cluster", "small_amethyst_bud", "medium_amethyst_bud", "large_amethyst_bud"} {
		for _, face := range []string{"up", "down", "north", "south", "west", "east"} {
			rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"minecraft:block_face": face})
			if !ok {
				t.Fatal(name, face)
			}
			m := s.model(rid)
			if m.full || len(m.boxes) > 0 || len(m.quads) != 2 || m.mode != "mask" {
				t.Errorf("crystal is cube-mapped: %s %s", name, face)
			}
		}
	}
	for _, name := range []string{"oak_sapling", "jungle_sapling", "birch_sapling", "spruce_sapling", "cherry_sapling", "pale_oak_sapling"} {
		rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"age_bit": false})
		if !ok {
			t.Fatal(name)
		}
		m := s.model(rid)
		if m.full || !m.cross || m.mode != "mask" {
			t.Errorf("sapling %s still solid", name)
		}
	}
}

func TestAnvilHasFootNeckAndOrientedTop(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for _, name := range []string{"anvil", "chipped_anvil", "damaged_anvil"} {
		for _, facing := range []string{"north", "south", "west", "east"} {
			rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"minecraft:cardinal_direction": facing})
			if !ok {
				t.Fatal(name, facing)
			}
			m := s.model(rid)
			if m.full || len(m.boxes) != 4 {
				t.Errorf("anvil is cube %s %s", name, facing)
				continue
			}
			top := m.boxes[3]
			if (facing == "north" || facing == "south") && top.max[2]-top.min[2] != 1 {
				t.Fatal("top points wrong way", facing, top)
			}
			if (facing == "east" || facing == "west") && top.max[0]-top.min[0] != 1 {
				t.Fatal("top points wrong way", facing, top)
			}
		}
	}
}

func TestFlowerPotNBTChangesLiveMeshAndClearsContents(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	ch := chunk.New(reg, cube.Range{0, 15})
	rid, ok := reg.StateToRuntimeID("minecraft:flower_pot", map[string]any{"update_bit": false})
	if !ok {
		t.Fatal("pot state")
	}
	ch.SetBlock(2, 1, 2, 0, rid)
	s.Put(world.ChunkPos{}, ch)
	empty, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.model(rid).full {
		t.Error("pot is a full cube")
	}
	s.Put(world.ChunkPos{}, ch, map[cube.Pos]map[string]any{{2, 1, 2}: {"id": "FlowerPot", "PlantBlock": map[string]any{"name": "minecraft:blue_orchid", "states": map[string]any{}}}})
	full, err := s.Snapshot(empty.Sequence, nil)
	if err != nil {
		t.Fatal(err)
	}
	tris := func(s Snapshot) int {
		n := 0
		for _, c := range s.Chunks {
			for _, p := range c.Parts {
				n += len(p.Indices) / 3
			}
		}
		return n
	}
	if tris(full) != tris(empty)+4 {
		t.Fatalf("pot flower missing: empty=%d planted=%d", tris(empty), tris(full))
	}
	s.Put(world.ChunkPos{}, ch)
	cleared, err := s.Snapshot(full.Sequence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tris(cleared) != tris(empty) {
		t.Fatal("removed plant remained cached")
	}
}
