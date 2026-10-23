package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
)

func TestFallingFluidColumnsHaveNoInternalShelves(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	for _, name := range []string{"water", "flowing_water", "lava", "flowing_lava"} {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t, reg, NewCatalog())
			c := chunk.New(reg, cube.Range{0, 15})
			for y := 1; y <= 3; y++ {
				rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"liquid_depth": int32(8)})
				if !ok {
					t.Fatal(name)
				}
				c.SetBlock(1, int16(y), 1, 0, rid)
			}
			s.Put(world.ChunkPos{}, c)
			snap, err := s.Snapshot(0, nil)
			if err != nil {
				t.Fatal(err)
			}
			horizontal := 0
			for _, p := range snap.Chunks[0].Parts {
				for i := 0; i < len(p.Positions); i += 12 {
					if p.Normals[i+1] != 0 {
						horizontal++
						y := p.Positions[i+1]
						if y != 1 && y != 4 {
							t.Errorf("internal horizontal fluid shelf at y=%v", y)
						}
					}
				}
			}
			if horizontal != 2 {
				t.Fatalf("want only column top and bottom, got %d", horizontal)
			}
		})
	}
}

func TestPressurePlatesHaveInsetThinPressedAndUnpressedModels(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for _, name := range []string{"wooden_pressure_plate", "stone_pressure_plate", "light_weighted_pressure_plate", "heavy_weighted_pressure_plate"} {
		for _, signal := range []int32{0, 1, 15} {
			rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"redstone_signal": signal})
			if !ok {
				t.Fatal(name, signal)
			}
			m := s.model(rid)
			h := float32(1.0 / 16)
			if signal > 0 {
				h = 1.0 / 32
			}
			want := box{[3]float32{1.0 / 16, 0, 1.0 / 16}, [3]float32{15.0 / 16, h, 15.0 / 16}}
			if m.full || len(m.boxes) != 1 || m.boxes[0] != want {
				t.Errorf("%s signal=%d: full=%v boxes=%v", name, signal, m.full, m.boxes)
			}
		}
	}
}

func TestFluidColumnAcrossLayersAndChunkBorder(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	left, right := chunk.New(reg, cube.Range{0, 15}), chunk.New(reg, cube.Range{0, 15})
	water, _ := reg.StateToRuntimeID("minecraft:water", map[string]any{"liquid_depth": int32(0)})
	flow, _ := reg.StateToRuntimeID("minecraft:flowing_water", map[string]any{"liquid_depth": int32(8)})
	left.SetBlock(15, 1, 1, 1, water)
	left.SetBlock(15, 2, 1, 0, flow)
	right.SetBlock(0, 1, 1, 0, flow)
	right.SetBlock(0, 2, 1, 1, water)
	s.Put(world.ChunkPos{}, left)
	s.Put(world.ChunkPos{1, 0}, right)
	snap, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range snap.Chunks {
		for _, p := range ch.Parts {
			for i := 0; i < len(p.Positions); i += 12 {
				if p.Normals[i+1] != 0 && p.Positions[i+1] == 2 {
					t.Fatal("internal shelf between different fluid aliases/layers")
				}
				if ch.X == 0 && p.Normals[i] == 1 && p.Positions[i] == 16 && p.Positions[i+1] < 2 {
					t.Fatal("internal face at chunk border")
				}
				if ch.X == 1 && p.Normals[i] == -1 && p.Positions[i] == 0 && p.Positions[i+1] < 2 {
					t.Fatal("internal face at chunk border")
				}
			}
		}
	}
}

func TestLanternBodyAndHangingHandleUseModelUVs(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	for _, name := range []string{"lantern", "soul_lantern"} {
		for _, hanging := range []bool{false, true} {
			rid, ok := reg.StateToRuntimeID("minecraft:"+name, map[string]any{"hanging": hanging})
			if !ok {
				t.Fatal(name)
			}
			m := s.model(rid)
			if m.full || len(m.boxes) != 0 || len(m.quads) < 14 {
				t.Errorf("%s hanging=%v still a cube: %+v", name, hanging, m)
				continue
			}
			minY, maxY := float32(2), float32(-1)
			for _, q := range m.quads {
				for _, p := range q.points {
					minY = min(minY, p[1])
					maxY = max(maxY, p[1])
					if p[0] < 5.0/16 || p[0] > 11.0/16 || p[2] < 5.0/16 || p[2] > 11.0/16 {
						t.Fatal("lantern exceeds six-pixel body width", p)
					}
				}
			}
			if hanging {
				if minY != 1.0/16 || maxY != 1 {
					t.Errorf("hanging bounds %v..%v", minY, maxY)
				}
			} else if minY != 0 || maxY != 11.0/16 {
				t.Errorf("standing bounds %v..%v", minY, maxY)
			}
		}
	}
}
