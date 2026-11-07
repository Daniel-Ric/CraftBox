package scene

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type mechanicalRegistry struct {
	world.BlockRegistry
	name  string
	props map[string]any
}

func (r mechanicalRegistry) RuntimeIDToState(uint32) (string, map[string]any, bool) {
	return r.name, r.props, true
}
func mechanicalModel(c *Catalog, name string, props map[string]any) blockModel {
	s := &Store{catalog: c, registry: mechanicalRegistry{name: name, props: props}}
	return s.model(1)
}
func mechanicalCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := LoadCatalog("../../assets/vanilla-3d")
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestPistonFacesFollowAllSixDirections(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, name := range []string{"minecraft:piston", "minecraft:sticky_piston"} {
		key := "piston_top_normal"
		if name == "minecraft:sticky_piston" {
			key = "piston_top_sticky"
		}
		front := c.materialTexture(map[string]any{"texture": key}, 16, 16)
		for facing, normal := range [][3]float32{{0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}, {-1, 0, 0}, {1, 0, 0}} {
			m := mechanicalModel(c, name, map[string]any{"facing_direction": int32(facing)})
			if len(m.quads) != 6 || len(m.boxes) != 0 {
				t.Fatalf("%s facing %d: expected textured surfaces, got %d quads, %d boxes", name, facing, len(m.quads), len(m.boxes))
			}
			found := false
			for _, q := range m.quads {
				if q.texture.missing {
					t.Fatal("missing piston texture")
				}
				if q.texture == front {
					found = true
					for a := range normal {
						if math.Abs(float64(q.normal[a]-normal[a])) > 0.00001 {
							t.Fatalf("front facing %d: normal %v", facing, q.normal)
						}
					}
				}
			}
			if !found {
				t.Fatalf("%s: missing proper retracted front", name)
			}
		}
	}
}
func TestPistonEntityStateSurvivesCaptureAndShortensBase(t *testing.T) {
	c := mechanicalCatalog(t)
	pos := cube.Pos{2, 4, 6}
	for _, state := range []uint8{0, 2} {
		raw := encodePotPlants(map[cube.Pos]map[string]any{pos: {"id": "PistonArm", "State": state, "Progress": float32(state / 2)}})
		var entries []potPlant
		if err := json.Unmarshal(raw, &entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Visual == nil {
			t.Fatal("piston extension state discarded from capture")
		}
		s := &Store{catalog: c, decorativeEntities: map[cube.Pos]map[string]any{pos: entries[0].Visual}}
		m := mechanicalModel(c, "minecraft:piston", map[string]any{"facing_direction": int32(1)})
		out := s.decorativeAt(&m, map[string]any{"facing_direction": int32(1)}, pos)
		maxY := float32(0)
		for _, q := range out.quads {
			for _, p := range q.points {
				maxY = max(maxY, p[1])
			}
		}
		want := float32(1)
		if state == 2 {
			want = .75
		}
		if maxY != want {
			t.Fatalf("state %d: base height %v, want %v", state, maxY, want)
		}
	}
}
func TestPistonHeadsHavePlateAndConnectingShaft(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, name := range []string{"minecraft:piston_arm_collision", "minecraft:sticky_piston_arm_collision", "minecraft:pistonArmCollision", "minecraft:stickyPistonArmCollision"} {
		m := mechanicalModel(c, name, map[string]any{"facing_direction": int32(1)})
		if m.full || len(m.boxes) > 0 || len(m.quads) != 12 {
			t.Fatalf("%s must have a plate and shaft, got %d quads, full=%t", name, len(m.quads), m.full)
		}
		low, high := float32(1), float32(0)
		for _, q := range m.quads {
			if q.texture.missing {
				t.Fatal("missing head texture")
			}
			for _, uv := range q.uv {
				if uv[0]*atlasSize < float32(q.texture.x)-.001 || uv[0]*atlasSize > float32(q.texture.x+q.texture.w)+.001 || uv[1]*atlasSize < float32(q.texture.y)-.001 || uv[1]*atlasSize > float32(q.texture.y+q.texture.h)+.001 {
					t.Fatalf("head UV samples outside its texture: %v", uv)
				}
			}
			for _, p := range q.points {
				low = min(low, p[1])
				high = max(high, p[1])
			}
		}
		if low != -.25 || high != 1 {
			t.Fatalf("shaft must reach the recessed base: %v..%v", low, high)
		}
	}
}
func TestTripwireIsTransparentFlatString(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, name := range []string{"minecraft:trip_wire", "minecraft:tripWire", "minecraft:tripwire"} {
		m := mechanicalModel(c, name, nil)
		if m.full || m.mode != "mask" || len(m.boxes) > 0 || len(m.quads) == 0 {
			t.Fatalf("%s still rendered as a cube", name)
		}
		for _, q := range m.quads {
			if q.texture.missing || !q.texture.alpha {
				t.Fatal("string must use its transparent trip_wire texture")
			}
			for _, p := range q.points {
				if p[1] > .1 {
					t.Fatalf("string obstructs the carpet: %v", p)
				}
			}
		}
	}
}

func TestPistonIgnoresHeadBelongingToDifferentPiston(t *testing.T) {
	c := mechanicalCatalog(t)
	props := map[string]any{"facing_direction": int32(5)}
	base := mechanicalModel(c, "minecraft:piston", props)
	for _, facing := range []int32{2, 5} {
		head := mechanicalModel(c, "minecraft:piston_arm_collision", map[string]any{"facing_direction": facing})
		out := c.pistonWithNeighbour(&base, props, &head)
		minX := float32(1)
		maxX := float32(0)
		for _, q := range out.quads {
			for _, p := range q.points {
				minX = min(minX, p[0])
				maxX = max(maxX, p[0])
			}
		}
		want := float32(1)
		if facing == 5 {
			want = .75
		}
		if math.Abs(float64(maxX-minX-want)) > .00001 {
			t.Fatalf("head facing %d shortened wrong base: %v..%v", facing, minX, maxX)
		}
	}
}
