package scene

func (c *Catalog) orientPillar(m *blockModel, props map[string]any) {
	axis, _ := props["pillar_axis"].(string)
	if axis != "x" && axis != "z" || len(m.quads) != 0 || m.cross || m.connect != "" || len(m.boxes) != 1 || m.boxes[0] != unit {
		return
	}
	rotate := func(p [3]float32, point bool) [3]float32 {
		offset := float32(0)
		if point {
			offset = 1
		}
		if axis == "x" {
			return [3]float32{p[1], offset - p[0], p[2]}
		}
		return [3]float32{p[0], offset - p[2], p[1]}
	}
	var textures [6]tile
	for face := range directions {
		textures[face] = c.texture(m.name, face)
	}
	texturedCubeSurface(m, textures, rotate)
}

func texturedCubeSurface(m *blockModel, textures [6]tile, rotate func([3]float32, bool) [3]float32) {
	for face, d := range directions {
		t := textures[face]
		q := modelQuad{texture: t, mode: m.mode, normal: rotate([3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, false)}
		if t.alpha && q.mode == "opaque" {
			q.mode = "mask"
		}
		for i, p := range corners[face] {
			q.points[i] = rotate([3]float32{float32(p[0]), float32(p[1]), float32(p[2])}, true)
		}
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			q.uv[i] = [2]float32{(float32(t.x) + uv[0]*float32(t.w)) / atlasSize, (float32(t.y) + uv[1]*float32(t.h)) / atlasSize}
		}
		for i, n := range directions {
			if q.normal == [3]float32{float32(n[0]), float32(n[1]), float32(n[2])} {
				q.cullFace = i + 1
				break
			}
		}
		m.quads = append(m.quads, q)
	}
}
