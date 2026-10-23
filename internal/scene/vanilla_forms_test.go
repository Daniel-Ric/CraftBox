package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
	"image/color"
	"testing"
	"testing/fstest"
)

func TestTrapdoorStatesMatchBedrockThinBounds(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for direction := 0; direction < 4; direction++ {
		for _, open := range []bool{false, true} {
			for _, top := range []bool{false, true} {
				rid, ok := reg.StateToRuntimeID("minecraft:trapdoor", map[string]any{"direction": int32(direction), "open_bit": open, "upside_down_bit": top})
				if !ok {
					t.Fatal("missing fixture state")
				}
				m := s.model(rid)
				if m.full || len(m.boxes) != 1 {
					t.Fatal("trapdoor rendered as full cube", direction, open, top)
				}
				want := (model.Trapdoor{Facing: cube.Direction(3 - direction), Open: open, Top: top}).BBox(cube.Pos{}, nil)[0]
				for a := 0; a < 3; a++ {
					if m.boxes[0].min[a] != float32(want.Min()[a]) || m.boxes[0].max[a] != float32(want.Max()[a]) {
						t.Fatal("wrong hinge or half", direction, open, top, m.boxes[0], want)
					}
				}
			}
		}
	}
}

func TestCutoutCubeDoesNotHideNeighboursThroughItsHoles(t *testing.T) {
	c := NewCatalog()
	c.AddPack(fstest.MapFS{"blocks.json": {Data: []byte(`{"stone":{"textures":"cutout"}}`)}, "textures/blocks/cutout.png": {Data: materialImage(16, 16, color.NRGBA{200, 200, 200, 0})}})
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	rid, _ := reg.StateToRuntimeID("minecraft:stone", nil)
	s := newTestStore(t, reg, c)
	if m := s.model(rid); m.full {
		t.Fatal("cutout texture incorrectly hides a solid neighbour")
	}
}

func TestIronChainUsesLegacyMojangTexturesAndThinBounds(t *testing.T) {
	c := NewCatalog()
	c.AddPack(fstest.MapFS{"textures/blocks/chain1.png": {Data: materialImage(16, 16, color.NRGBA{100, 100, 100, 255})}})
	if tex := c.texture("minecraft:iron_chain", 0); tex.missing {
		t.Fatal("renamed iron chain lost its texture")
	}
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	rid, ok := reg.StateToRuntimeID("minecraft:iron_chain", map[string]any{"pillar_axis": "y"})
	if !ok {
		t.Fatal("missing chain state")
	}
	s := newTestStore(t, reg, c)
	m := s.model(rid)
	if m.full || len(m.quads) != 2 {
		t.Fatal("chain rendered as full cube")
	}
}

func TestConnectedFenceBoxesAndPartialBlockUVs(t *testing.T) {
	m := &blockModel{connect: "fence", boxes: []box{{[3]float32{.375, 0, .375}, [3]float32{.625, 1, .625}}}}
	neighbours := [4]*blockModel{&emptyBlockModel, &emptyBlockModel, &emptyBlockModel, &emptyBlockModel}
	if boxes := connectedBoxes(m, neighbours); len(boxes) != 1 {
		t.Fatal("fence connected through missing chunks")
	}
	neighbours[0] = &blockModel{full: true}
	if boxes := connectedBoxes(m, neighbours); len(boxes) != 3 || boxes[1].max[0] != 1 {
		t.Fatal("fence did not connect to neighbour")
	}
	p := newPart("opaque")
	emitBoxQuad(p, [4][3]float32{{0, 0, 1}, {1, 0, 1}, {1, .5, 1}, {0, .5, 1}}, [3]float32{0, 0, 1}, [3]float32{}, tile{x: 1, y: 1}, 4)
	if p.UVs[1] != 17.0/atlasSize || p.UVs[5] != 9.0/atlasSize {
		t.Fatal("partial block texture stretched instead of cropped", p.UVs)
	}
}

func TestTallFlowersAndGrassAreCrossedWithCorrectHalfTextures(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for _, name := range []string{"sunflower", "rose_bush", "peony", "lilac", "tall_grass", "large_fern"} {
		for _, upper := range []bool{false, true} {
			rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"upper_block_bit": upper})
			if !ok {
				t.Fatal("missing fixture state", name)
			}
			m := s.model(rid)
			if !m.cross || m.full || len(m.boxes) != 0 {
				t.Fatal("plant rendered as cube", name)
			}
			want := 3
			if upper {
				want = 2
			}
			if m.plantFace != want {
				t.Fatal("wrong flower half texture", name, upper)
			}
		}
	}
}

func TestInvisibleLightAndSmallPlantsDoNotBecomeOccludingCubes(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for _, name := range []string{"light_block_15", "closed_eyeblossom", "open_eyeblossom", "firefly_bush", "seagrass", "vine", "web"} {
		rid, ok := reg.StateToRuntimeID("minecraft:"+name, nil)
		if !ok {
			t.Fatal("missing fixture", name)
		}
		m := s.model(rid)
		if m.full {
			t.Fatal("invisible/small block occludes whole cube", name)
		}
		if name == "light_block_15" && len(m.boxes) != 0 {
			t.Fatal("light block must be invisible")
		}
	}
}
