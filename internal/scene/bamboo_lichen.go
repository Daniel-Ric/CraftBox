package scene

func (c *Catalog) bambooModel(props map[string]any) blockModel {
	m := blockModel{name: "minecraft:bamboo", mode: "mask"}
	width := float32(2)
	if props["bamboo_stalk_thickness"] == "thick" {
		width = 3
	}
	lo, hi := (8-width/2)/16, (8+width/2)/16
	stem := c.materialTexture(map[string]any{"texture": "bamboo_stem"}, 16, 16)
	appendSkinBox(&m, box{[3]float32{lo, 0, lo}, [3]float32{hi, 1, hi}}, stem, [2]float32{}, 16, 16)
	for face := range m.quads {
		q := &m.quads[face]
		q.mode = "opaque"
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			h := float32(16)
			if face == 2 || face == 3 {
				h = width
			}
			q.uv[i] = [2]float32{(float32(stem.x) + uv[0]*width*float32(stem.w)/16) / atlasSize, (float32(stem.y) + uv[1]*h*float32(stem.h)/16) / atlasSize}
		}
	}
	texture := ""
	switch props["bamboo_leaf_size"] {
	case "small_leaves":
		texture = "bamboo_small_leaf"
	case "large_leaves":
		texture = "bamboo_leaf"
	}
	if texture != "" {
		leaf := c.materialTexture(map[string]any{"texture": texture}, 16, 16)
		for _, points := range [][4][3]float32{{{0, 0, 0}, {1, 0, 1}, {1, 1, 1}, {0, 1, 0}}, {{1, 0, 0}, {0, 0, 1}, {0, 1, 1}, {1, 1, 0}}} {
			m.quads = append(m.quads, texturedQuad(points, leaf))
		}
	}
	return m
}

func (c *Catalog) glowLichenModel(bits int) blockModel {
	m := blockModel{name: "minecraft:glow_lichen", mode: "mask"}
	if bits == 0 {
		bits = 63
	}
	tex := c.texture(m.name, 0)
	for bit, face := range []int{3, 2, 4, 1, 5, 0} {
		if bits&(1<<bit) == 0 {
			continue
		}
		var points [4][3]float32
		for i, corner := range corners[face] {
			for axis, p := range corner {
				points[i][axis] = float32(p)
			}
			axis := face / 2
			if face%2 == 0 {
				points[i][axis] = 63.0 / 64
			} else {
				points[i][axis] = 1.0 / 64
			}
		}
		m.quads = append(m.quads, texturedQuad(points, tex))
	}
	return m
}
