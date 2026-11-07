package scene

import (
	"math"
	"testing"
)

func TestBeaconSeparatesGlassCoreAndObsidianBase(t *testing.T) {
	c := mechanicalCatalog(t)
	m := mechanicalModel(c, "minecraft:beacon", nil)
	if m.full || len(m.boxes) != 0 || len(m.quads) != 18 {
		t.Fatalf("beacon still uses a single cube: full=%t boxes=%d surfaces=%d", m.full, len(m.boxes), len(m.quads))
	}
	core, base, shell := c.texture("minecraft:beacon", 2), c.texture("minecraft:beacon", 3), c.texture("minecraft:beacon", 0)
	for _, group := range []struct {
		tex    tile
		lo, hi [3]float32
		mode   string
	}{
		{base, [3]float32{.125, 0, .125}, [3]float32{.875, .1875, .875}, "opaque"},
		{core, [3]float32{.1875, .1875, .1875}, [3]float32{.8125, .8125, .8125}, "unlit"},
		{shell, [3]float32{0, 0, 0}, [3]float32{1, 1, 1}, "blend"},
	} {
		lo, hi := [3]float32{1, 1, 1}, [3]float32{}
		count := 0
		for _, q := range m.quads {
			if q.texture != group.tex {
				continue
			}
			count++
			if q.texture.missing || q.mode != group.mode {
				t.Fatalf("wrong texture/material for beacon layer: %s", q.mode)
			}
			for _, p := range q.points {
				for a := range p {
					lo[a] = min(lo[a], p[a])
					hi[a] = max(hi[a], p[a])
				}
			}
		}
		if count != 6 || lo != group.lo || hi != group.hi {
			t.Fatalf("wrong beacon layer bounds: %d faces %v..%v want %v..%v", count, lo, hi, group.lo, group.hi)
		}
	}
	// The small luminous cube must show the complete beacon sprite on every
	// side, rather than the cropped UVs of a full-block face.
	for _, q := range m.quads {
		if q.texture != core {
			continue
		}
		minU, minV, maxU, maxV := float32(1), float32(1), float32(0), float32(0)
		for _, uv := range q.uv {
			minU = min(minU, uv[0])
			minV = min(minV, uv[1])
			maxU = max(maxU, uv[0])
			maxV = max(maxV, uv[1])
		}
		if math.Abs(float64((maxU-minU)*atlasSize-float32(core.w))) > .001 || math.Abs(float64((maxV-minV)*atlasSize-float32(core.h))) > .001 {
			t.Fatal("beacon core sprite is cropped")
		}
	}
}
