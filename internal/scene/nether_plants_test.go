package scene

import "testing"

func TestNetherGroundPlantsUseCrossedMaskedPlanes(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, name := range []string{"minecraft:warped_roots", "minecraft:crimson_roots", "minecraft:nether_sprouts"} {
		t.Run(name, func(t *testing.T) {
			m := mechanicalModel(c, name, nil)
			if m.full || len(m.boxes) != 0 || !m.cross || m.mode != "mask" {
				t.Fatalf("ground plant still uses block surfaces: full=%t boxes=%d cross=%t mode=%s", m.full, len(m.boxes), m.cross, m.mode)
			}
			tex := c.texture(name, m.plantFace)
			if tex.missing || !tex.alpha {
				t.Fatal("plant must use its transparent vegetation texture")
			}
			// Root block definitions also have a potted-only texture on a side.
			// Ground plants must consistently use the full, unpotted sprite.
			if m.plantFace != 2 {
				t.Fatal("wrong texture slot for ground vegetation")
			}
		})
	}
}
