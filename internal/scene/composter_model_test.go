package scene

import "testing"

func TestComposterHasClosedFloorAndInteriorWalls(t *testing.T) {
	c := mechanicalCatalog(t)
	m := mechanicalModel(c, "minecraft:composter", map[string]any{"composter_fill_level": int32(0)})
	if m.full || len(m.boxes) > 0 || len(m.quads) < 11 {
		t.Fatalf("composter has no hollow container model: full=%t boxes=%d quads=%d", m.full, len(m.boxes), len(m.quads))
	}
	floor := false
	inside := 0
	for _, q := range m.quads {
		if q.texture.missing {
			t.Fatal("missing composter texture")
		}
		if q.normal[1] > .9 && q.points[0][1] == .125 {
			floor = true
			if q.texture.alpha {
				t.Fatal("interior floor must block the view of blocks below")
			}
		}
		if q.normal[0] < -.9 && q.points[0][0] == .875 || q.normal[0] > .9 && q.points[0][0] == .125 || q.normal[2] < -.9 && q.points[0][2] == .875 || q.normal[2] > .9 && q.points[0][2] == .125 {
			inside++
		}
	}
	if !floor || inside != 4 {
		t.Fatalf("missing floor or inner walls: floor=%t walls=%d", floor, inside)
	}
}

func TestComposterFillLevelControlsSurfaceHeightAndReadyTexture(t *testing.T) {
	c := mechanicalCatalog(t)
	for level := 1; level <= 8; level++ {
		m := mechanicalModel(c, "minecraft:composter", map[string]any{"composter_fill_level": int32(level)})
		key := "compost"
		if level == 8 {
			key = "compost_ready"
		}
		tex := c.materialTexture(map[string]any{"texture": key}, 16, 16)
		found := false
		for _, q := range m.quads {
			if q.texture == tex && q.normal[1] > .9 {
				found = true
				want := float32(1+2*min(level, 7)) / 16
				for _, p := range q.points {
					if p[1] != want {
						t.Fatalf("level %d: wrong fill height %v, want %v", level, p[1], want)
					}
				}
			}
		}
		if !found {
			t.Fatalf("level %d: missing compost surface", level)
		}
	}
}
