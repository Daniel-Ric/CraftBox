package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"testing"
)

func TestLecternHasSlopedDeskAndBookState(t *testing.T) {
	c := mechanicalCatalog(t)
	m := mechanicalModel(c, "minecraft:lectern", map[string]any{"minecraft:cardinal_direction": "north"})
	if m.full || len(m.boxes) > 0 || len(m.quads) < 18 {
		t.Fatal("lectern still rendered as a cube")
	}
	sloped := false
	for _, q := range m.quads {
		if q.normal[1] > .5 && q.normal[1] < .99 {
			sloped = true
		}
	}
	if !sloped {
		t.Fatal("lectern desk is not inclined")
	}
	pos := cube.Pos{}
	var entries []potPlant
	_ = json.Unmarshal(encodePotPlants(map[cube.Pos]map[string]any{pos: {"id": "Lectern", "hasBook": uint8(1)}}), &entries)
	if len(entries) != 1 || !flag(entries[0].Visual["hasBook"]) {
		t.Fatal("lectern book presence discarded")
	}
	s := &Store{catalog: c, decorativeEntities: map[cube.Pos]map[string]any{pos: entries[0].Visual}}
	withBook := s.decorativeAt(&m, map[string]any{}, pos)
	if len(withBook.quads) <= len(m.quads) {
		t.Fatal("lectern book missing")
	}
	empty := c.lecternModel(nil, map[string]any{"hasBook": uint8(0), "book": map[string]any{}})
	if len(empty.quads) != len(m.quads) {
		t.Fatal("empty lectern inventory still shows a book")
	}
}
func TestEnchantingTableIsTwelvePixelsHighAndHasFloatingBook(t *testing.T) {
	c := mechanicalCatalog(t)
	m := mechanicalModel(c, "minecraft:enchanting_table", nil)
	if m.full || len(m.boxes) > 0 || len(m.quads) <= 6 {
		t.Fatal("enchanting table still rendered as a cube")
	}
	top := c.texture(m.name, 2)
	found := false
	bookAbove := false
	for _, q := range m.quads {
		if q.texture.missing {
			t.Fatal("missing table/book texture")
		}
		if q.texture == top {
			found = true
			for _, p := range q.points {
				if p[1] != .75 {
					t.Fatalf("table top at %v rather than 12/16", p[1])
				}
			}
		}
		for _, p := range q.points {
			if p[1] > .9 {
				bookAbove = true
			}
		}
	}
	if !found || !bookAbove {
		t.Fatal("tabletop or floating book missing")
	}
}
func TestStonecutterHasLowBaseAndUprightSaw(t *testing.T) {
	c := mechanicalCatalog(t)
	m := mechanicalModel(c, "minecraft:stonecutter_block", map[string]any{"minecraft:cardinal_direction": "north"})
	if m.full || len(m.boxes) > 0 || len(m.quads) != 7 {
		t.Fatal("stonecutter still rendered as a cube")
	}
	saw := false
	for _, q := range m.quads {
		if q.texture.missing {
			t.Fatal("missing stonecutter texture")
		}
		if q.mode == "mask" {
			saw = true
			u, v := float32(0), float32(0)
			for _, uv := range q.uv {
				u += uv[0] / 4
				v += uv[1] / 4
			}
			if c.atlas.NRGBAAt(int(u*atlasSize), int(v*atlasSize)).A == 0 {
				t.Fatal("saw samples the transparent half of its sprite")
			}
			for _, p := range q.points {
				if p[0] != .5 {
					t.Fatal("saw is not a vertical center plane")
				}
			}
		} else {
			for _, p := range q.points {
				if p[1] > .5625 {
					t.Fatal("stonecutter base too tall")
				}
			}
		}
	}
	if !saw {
		t.Fatal("saw blade missing")
	}
}
func TestRedstoneStraightWireUsesLineSprite(t *testing.T) {
	c := mechanicalCatalog(t)
	props := map[string]any{"redstone_east": "side", "redstone_west": "side", "redstone_north": "none", "redstone_south": "none"}
	m := mechanicalModel(c, "minecraft:redstone_wire", props)
	// The cross texture has vertical branches through its centre. A straight
	// east/west wire must not sample that sprite and leave a square junction.
	cross := c.redstoneTexture(0)
	if len(m.quads) != 1 || m.quads[0].texture == cross {
		t.Fatal("straight redstone wire still samples cropped cross texture")
	}
}
