package scene

func (c *Catalog) cakeModel(name string, props map[string]any) blockModel {
	bites := max(0, min(6, number(props["bite_counter"])))
	b := box{[3]float32{float32(1+bites*2) / 16, 0, 1.0 / 16}, [3]float32{15.0 / 16, .5, 15.0 / 16}}
	m := blockModel{name: name, mode: "mask"}
	for face, d := range directions {
		tex := c.texture(name, face)
		if face == 1 && bites > 0 {
			tex = c.materialTexture(map[string]any{"texture": "cake_inner"}, 16, 16)
		}
		var points [4][3]float32
		for i, p := range corners[face] {
			for a := range p {
				points[i][a] = b.min[a]
				if p[a] == 1 {
					points[i][a] = b.max[a]
				}
			}
		}
		part := newPart("mask")
		emitBoxQuad(part, points, [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, [3]float32{}, tex, face)
		q := modelQuad{points: points, normal: [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, texture: tex, mode: "mask"}
		for i := range q.uv {
			q.uv[i] = [2]float32{part.UVs[i*2], part.UVs[i*2+1]}
		}
		m.quads = append(m.quads, q)
	}
	return m
}
func (c *Catalog) pickleModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "mask"}
	count := max(1, min(4, number(props["cluster_count"])+1))
	tex := c.texture(name, 0)
	centers := [][3]float32{{.5, .5, 6}}
	if count > 1 {
		centers = [][3]float32{{.3125, .3125, 6}, {.6875, .6875, 7}, {.3125, .75, 5}, {.75, .3125, 6}}
	}
	for i := 0; i < count; i++ {
		p := centers[i]
		start := len(m.quads)
		appendModelBox(&m, box{[3]float32{p[0] - .125, 0, p[1] - .125}, [3]float32{p[0] + .125, p[2] / 16, p[1] + .125}}, tex)
		for face := 0; face < 6; face++ {
			u, v, w, h := float32(face%4*4), float32(5), float32(4), p[2]
			if face == 2 {
				u = 4
				v = 1
				h = 4
			}
			if face == 3 {
				u = 8
				v = 1
				h = 4
			}
			for j, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
				m.quads[start+face].uv[j] = [2]float32{(float32(tex.x) + u + uv[0]*w) / atlasSize, (float32(tex.y) + v + uv[1]*h) / atlasSize}
			}
		}
		if !flag(props["dead_bit"]) {
			for axis := 0; axis < 2; axis++ {
				points := [4][3]float32{{p[0] - .125, p[2] / 16, p[1]}, {p[0] + .125, p[2] / 16, p[1]}, {p[0] + .125, (p[2] + 2) / 16, p[1]}, {p[0] - .125, (p[2] + 2) / 16, p[1]}}
				if axis == 1 {
					for j := range points {
						points[j][0], points[j][2] = p[0], p[1]+points[j][0]-p[0]
					}
				}
				q := texturedQuad(points, tex)
				for j, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
					q.uv[j] = [2]float32{(float32(tex.x) + uv[0]*4) / atlasSize, (float32(tex.y) + uv[1]*5) / atlasSize}
				}
				m.quads = append(m.quads, q)
			}
		}
	}
	return m
}
