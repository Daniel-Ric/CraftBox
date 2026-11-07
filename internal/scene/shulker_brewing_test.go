package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"testing"
)

func TestShulkerBoxesUseClosedSkinGeometryWithoutTransparentHoles(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, colour := range []string{"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray", "light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black", "undyed"} {
		m := mechanicalModel(c, "minecraft:"+colour+"_shulker_box", nil)
		if len(m.boxes) > 0 || len(m.quads) != 12 || m.entityVisual != "shulker" {
			t.Fatalf("%s shulker still uses a block sprite: %d surfaces", colour, len(m.quads))
		}
		for _, q := range m.quads {
			if q.texture.missing {
				t.Fatalf("missing %s shulker skin", colour)
			}
		}
		for face, normal := range directions {
			if face == 2 || face == 3 {
				continue
			}
			axis := face / 2
			for u := 0; u < 16; u++ {
				for v := 0; v < 16; v++ {
					sample := [3]float32{}
					a, b := 0, 1
					if axis == 0 {
						a, b = 2, 1
					}
					sample[a] = (float32(u) + .5) / 16
					sample[b] = (float32(v) + .5) / 16
					covered := false
					for _, q := range m.quads {
						if q.normal != [3]float32{float32(normal[0]), float32(normal[1]), float32(normal[2])} {
							continue
						}
						fraction := [2]float32{}
						valid := true
						for i, j := range []int{1, 3} {
							dot, length := float32(0), float32(0)
							for dim := 0; dim < 3; dim++ {
								if dim == axis {
									continue
								}
								delta := q.points[j][dim] - q.points[0][dim]
								dot += (sample[dim] - q.points[0][dim]) * delta
								length += delta * delta
							}
							if length == 0 {
								valid = false
								break
							}
							fraction[i] = dot / length
							if fraction[i] < 0 || fraction[i] > 1 {
								valid = false
							}
						}
						if !valid {
							continue
						}
						uv := q.uv[0]
						for dim := 0; dim < 2; dim++ {
							uv[dim] += fraction[0]*(q.uv[1][dim]-q.uv[0][dim]) + fraction[1]*(q.uv[3][dim]-q.uv[0][dim])
						}
						if c.atlas.NRGBAAt(int(uv[0]*atlasSize), int(uv[1]*atlasSize)).A == 255 {
							covered = true
							break
						}
					}
					if !covered {
						t.Fatalf("%s shulker hole on face %d pixel %d,%d", colour, face, u, v)
					}
				}
			}
		}
	}
}

func TestShulkerLidCutoutsHaveOpaqueBaseBehindThem(t *testing.T) {
	c := mechanicalCatalog(t)
	m := c.shulkerBoxModel("minecraft:red_shulker_box", nil, nil)
	// The lid's skin has openings in its lower four pixels. The original
	// eight-pixel base must remain behind those openings, below y=8/16.
	for face := 0; face < 6; face++ {
		if face == 2 || face == 3 {
			continue
		}
		found := false
		for _, q := range m.quads {
			if q.normal != [3]float32{float32(directions[face][0]), float32(directions[face][1]), float32(directions[face][2])} {
				continue
			}
			lo, hi := float32(1), float32(0)
			for _, p := range q.points {
				lo = min(lo, p[1])
				hi = max(hi, p[1])
			}
			if lo == 0 && hi == .5 {
				found = true
			}
		}
		if !found {
			t.Fatalf("no full-height base behind lid cutouts, face %d", face)
		}
	}
}

func TestShulkerFacingSurvivesCaptureMetadata(t *testing.T) {
	pos := cube.Pos{1, 2, 3}
	raw := encodePotPlants(map[cube.Pos]map[string]any{pos: {"id": "ShulkerBox", "facing": uint8(5)}})
	var entries []potPlant
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || number(entries[0].Visual["facing"]) != 5 {
		t.Fatal("shulker mounting direction discarded")
	}
}

func TestBrewingStandHasFeetPostAndSlotPlanes(t *testing.T) {
	c := mechanicalCatalog(t)
	m := mechanicalModel(c, "minecraft:brewing_stand", nil)
	if m.full || len(m.boxes) > 0 || len(m.quads) < 27 {
		t.Fatalf("brewing stand still a cube: full=%t boxes=%d quads=%d", m.full, len(m.boxes), len(m.quads))
	}
	for _, q := range m.quads {
		if q.texture.missing {
			t.Fatal("missing brewing texture")
		}
	}
	filled := mechanicalModel(c, "minecraft:brewing_stand", map[string]any{"brewing_stand_slot_a_bit": uint8(1)})
	changes := 0
	for i, q := range m.quads {
		if q.uv != filled.quads[i].uv {
			changes++
		}
	}
	if changes != 1 {
		t.Fatalf("one filled slot should change only its bottle plane, changed %d surfaces", changes)
	}
}
