package scene

import "strings"

func (c *Catalog) shulkerBoxModel(name string, props, data map[string]any) blockModel {
	colour := strings.TrimSuffix(strings.TrimPrefix(name, "minecraft:"), "_shulker_box")
	if colour == "shulker_box" || colour == "" {
		colour, _ = props["color"].(string)
		if colour == "" {
			colour = "undyed"
		}
	}
	if colour == "light_gray" {
		colour = "silver"
	}
	tex := c.materialTexture(map[string]any{"texture": "textures/entity/shulker/shulker_" + colour}, 64, 64)
	m := blockModel{name: name, mode: "opaque", entityVisual: "shulker", full: true}
	// Use the official shulker skin layout: base 16x8x16 at UV 0,28;
	// lid 16x12x16 at UV 0,0. The lower lid has skin cutouts, so its
	// eight-pixel base must remain underneath as the solid inner layer.
	base := blockModel{mode: "opaque"}
	appendSkinBox(&base, box{[3]float32{}, [3]float32{1, .5, 1}}, tex, [2]float32{0, 28}, 64, 64)
	for _, q := range base.quads {
		for j, p := range q.points {
			for _, a := range []int{0, 2} {
				if p[a] == 0 {
					q.points[j][a] = .0001
				} else if p[a] == 1 {
					q.points[j][a] = .9999
				}
			}
		}
		m.quads = append(m.quads, q)
	}
	lid := blockModel{mode: "mask"}
	appendSkinBox(&lid, box{[3]float32{0, .25, 0}, [3]float32{1, 1, 1}}, tex, [2]float32{}, 64, 64)
	m.quads = append(m.quads, lid.quads...)
	facing := 1
	if value, ok := props["facing_direction"]; ok {
		facing = mountedFace(map[string]any{"facing_direction": value})
	}
	if value, ok := data["facing"]; ok {
		facing = number(value)
	}
	mountModel(&m, facing)
	return m
}

func (c *Catalog) brewingStandModel(props map[string]any) blockModel {
	m := blockModel{name: "minecraft:brewing_stand", mode: "opaque"}
	base := c.materialTexture(map[string]any{"texture": "brewing_stand_base"}, 16, 16)
	for _, b := range []box{
		{[3]float32{1.0 / 16, 0, 1.0 / 16}, [3]float32{7.0 / 16, 2.0 / 16, 7.0 / 16}},
		{[3]float32{1.0 / 16, 0, 9.0 / 16}, [3]float32{7.0 / 16, 2.0 / 16, 15.0 / 16}},
		{[3]float32{9.0 / 16, 0, 5.0 / 16}, [3]float32{15.0 / 16, 2.0 / 16, 11.0 / 16}},
	} {
		appendModelBox(&m, b, base)
	}
	tex := c.materialTexture(map[string]any{"texture": "brewing_stand"}, 16, 16)
	start := len(m.quads)
	appendModelBox(&m, box{[3]float32{7.0 / 16, 0, 7.0 / 16}, [3]float32{9.0 / 16, 14.0 / 16, 9.0 / 16}}, tex)
	for face := 0; face < 6; face++ {
		q := &m.quads[start+face]
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			v0, vh := float32(2), float32(14)
			if face == 2 || face == 3 {
				vh = 2
			}
			q.uv[i] = [2]float32{(float32(tex.x) + 7 + uv[0]*2) / atlasSize, (float32(tex.y) + v0 + uv[1]*vh) / atlasSize}
		}
	}
	for slot, key := range []string{"brewing_stand_slot_a_bit", "brewing_stand_slot_b_bit", "brewing_stand_slot_c_bit"} {
		arm := blockModel{mode: "mask"}
		q := texturedQuad([4][3]float32{{.5, 0, .5}, {1, 0, .5}, {1, 1, .5}, {.5, 1, .5}}, tex)
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			u := 8 + uv[0]*8
			if flag(props[key]) {
				u = 8 - uv[0]*8
			}
			q.uv[i] = [2]float32{(float32(tex.x) + u) / atlasSize, (float32(tex.y) + uv[1]*16) / atlasSize}
		}
		arm.quads = []modelQuad{q}
		transformQuads(&arm, rotationMatrix([3]float32{.5, 0, .5}, [3]float32{0, float32(slot) * 120, 0}))
		m.quads = append(m.quads, arm.quads...)
	}
	return m
}
