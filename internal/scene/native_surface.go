package scene

func (c *Catalog) texturedAnvilModel(name string, props map[string]any) blockModel {
	m := anvilModel(name, props)
	canonical := anvilModel(name, nil)
	turns := 0
	switch props["minecraft:cardinal_direction"] {
	case "west":
		turns = 1
	case "south":
		turns = 2
	case "east":
		turns = 3
	}
	for bi, b := range canonical.boxes {
		for face, d := range directions {
			tex := c.texture(name, 0)
			if bi == 3 && face == 2 {
				tex = c.texture(name, 2)
			}
			var points [4][3]float32
			for i, p := range corners[face] {
				for a := range p {
					points[i][a] = b.min[a]
					if p[a] != 0 {
						points[i][a] = b.max[a]
					}
				}
			}

			p := &Part{}
			emitBoxQuad(p, points, [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, [3]float32{}, tex, face)
			q := modelQuad{texture: tex, mode: "opaque", points: points, normal: [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}}
			for i := range q.uv {
				q.uv[i] = [2]float32{p.UVs[2*i], p.UVs[2*i+1]}
			}
			for n := 0; n < turns; n++ {
				for i, v := range q.points {
					q.points[i] = [3]float32{v[2], v[1], 1 - v[0]}
				}
				q.normal = [3]float32{q.normal[2], q.normal[1], -q.normal[0]}
			}
			m.quads = append(m.quads, q)
		}
	}
	return m
}

func (c *Catalog) saplingModel(name string) blockModel {
	const lo, hi = float32(.18180195), float32(.81819805)
	m := blockModel{name: name, mode: "mask", cross: true}
	tex := c.texture(name, 0)
	for _, pts := range [][4][3]float32{{{lo, 0, lo}, {hi, 0, hi}, {hi, 1, hi}, {lo, 1, lo}}, {{hi, 0, lo}, {lo, 0, hi}, {lo, 1, hi}, {hi, 1, lo}}} {
		m.quads = append(m.quads, texturedQuad(pts, tex))
	}
	return m
}

func (c *Catalog) vineModel(bits int) blockModel {
	m := blockModel{name: "minecraft:vine", mode: "mask"}
	tex := c.texture(m.name, 0)
	for i, face := range []int{4, 1, 5, 0} {
		if bits&(1<<i) == 0 {
			continue
		}
		var pts [4][3]float32
		for j, p := range corners[face] {
			for a := range p {
				pts[j][a] = float32(p[a])
			}
			axis := face / 2
			pts[j][axis] = 1.0 / 64
			if face%2 == 0 {
				pts[j][axis] = 63.0 / 64
			}
		}
		m.quads = append(m.quads, texturedQuad(pts, tex))
	}
	if bits == 0 {
		m.quads = append(m.quads, texturedQuad([4][3]float32{{0, 63.0 / 64, 1}, {1, 63.0 / 64, 1}, {1, 63.0 / 64, 0}, {0, 63.0 / 64, 0}}, tex))
	}
	return m
}
