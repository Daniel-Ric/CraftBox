package scene

import "github.com/go-gl/mathgl/mgl32"

func (c *Catalog) lanternModel(name string, hanging bool) blockModel {
	m := blockModel{name: name, mode: "mask"}
	tex := c.texture(name, 0)
	add := func(points [4][3]float32, uv [4]float32) {
		q := modelQuad{texture: tex, mode: "mask"}
		coords := [4][2]float32{{uv[0], uv[3]}, {uv[2], uv[3]}, {uv[2], uv[1]}, {uv[0], uv[1]}}
		for i, p := range points {
			for a := range p {
				q.points[i][a] = p[a] / 16
			}
			q.uv[i] = [2]float32{(float32(tex.x) + coords[i][0]) / atlasSize, (float32(tex.y) + coords[i][1]) / atlasSize}
		}
		normal := mgl32.Vec3(q.points[1]).Sub(mgl32.Vec3(q.points[0])).Cross(mgl32.Vec3(q.points[2]).Sub(mgl32.Vec3(q.points[0]))).Normalize()
		q.normal = [3]float32(normal)
		m.quads = append(m.quads, q)
	}
	offset := float32(0)
	if hanging {
		offset = 1
	}
	for i, b := range []box{{[3]float32{5, offset, 5}, [3]float32{11, 7 + offset, 11}}, {[3]float32{6, 7 + offset, 6}, [3]float32{10, 9 + offset, 10}}} {
		for face := range directions {
			uv := [4]float32{0, 2, 6, 9}
			if i == 1 {
				uv = [4]float32{0, 0, 4, 2}
			}
			if face == 2 || face == 3 {
				uv = [4]float32{0, 9, 6, 15}
				if i == 1 {
					uv = [4]float32{0, 9, 4, 13}
				}
			}
			var q [4][3]float32
			for v, p := range corners[face] {
				for a := range p {
					q[v][a] = b.min[a]
					if p[a] != 0 {
						q[v][a] = b.max[a]
					}
				}
			}
			add(q, uv)
		}
	}
	bottom, top := float32(9), float32(11)
	uv := [4]float32{11, 1, 14, 3}
	if hanging {
		bottom = 10
		top = 16
		uv = [4]float32{11, 0, 14, 6}
	}
	add([4][3]float32{{6.5, bottom, 8}, {9.5, bottom, 8}, {9.5, top, 8}, {6.5, top, 8}}, uv)
	add([4][3]float32{{8, bottom, 6.5}, {8, bottom, 9.5}, {8, top, 9.5}, {8, top, 6.5}}, uv)
	return m
}
