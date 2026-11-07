package scene

func (c *Catalog) composterModel(props map[string]any) blockModel {
	m := blockModel{name: "minecraft:composter", mode: "opaque"}
	side := c.texture(m.name, 0)
	outer := blockModel{mode: "opaque"}
	appendModelBox(&outer, unit, side)
	for _, face := range []int{0, 1, 4, 5} {
		m.quads = append(m.quads, outer.quads[face])
	}
	bottom := blockModel{mode: "opaque"}
	appendModelBox(&bottom, unit, c.texture(m.name, 3))
	m.quads = append(m.quads, bottom.quads[3])
	// The top sprite contains only the rim. It cannot supply the inner walls
	// or floor: a cube with this transparent top shows neighbouring blocks.
	rim := texturedQuad([4][3]float32{{0, 1, 1}, {1, 1, 1}, {1, 1, 0}, {0, 1, 0}}, c.texture(m.name, 2))
	m.quads = append(m.quads, rim)
	inner := blockModel{mode: "opaque"}
	appendModelBox(&inner, box{[3]float32{.125, .125, .125}, [3]float32{.875, 1, .875}}, side)
	for _, face := range []int{0, 1, 4, 5} {
		q := inner.quads[face]
		q.points[1], q.points[3] = q.points[3], q.points[1]
		q.uv[1], q.uv[3] = q.uv[3], q.uv[1]
		for a := range q.normal {
			q.normal[a] = -q.normal[a]
		}
		m.quads = append(m.quads, q)
	}
	addFloor := func(y float32, tex tile) {
		floor := blockModel{mode: "opaque"}
		appendModelBox(&floor, box{[3]float32{.125, 0, .125}, [3]float32{.875, y, .875}}, tex)
		m.quads = append(m.quads, floor.quads[2])
	}
	addFloor(.125, c.texture(m.name, 3))
	level := max(0, min(8, number(props["composter_fill_level"])))
	if level > 0 {
		key := "compost"
		if level == 8 {
			key = "compost_ready"
		}
		addFloor(float32(1+2*min(level, 7))/16, c.materialTexture(map[string]any{"texture": key}, 16, 16))
	}
	return m
}
