package scene

func (c *Catalog) paneQuads(name, mode string, boxes []box) []modelQuad {
	m := blockModel{name: name, mode: mode}
	broad, edge := c.texture(name, 4), c.texture(name, 0)

	breaks := [4]float32{0, 7.0 / 16, 9.0 / 16, 1}
	var filled [3][3]bool
	for x := 0; x < 3; x++ {
		for z := 0; z < 3; z++ {
			cx, cz := (breaks[x]+breaks[x+1])/2, (breaks[z]+breaks[z+1])/2
			for _, b := range boxes {
				if cx > b.min[0] && cx < b.max[0] && cz > b.min[2] && cz < b.max[2] {
					filled[x][z] = true
				}
			}
		}
	}
	axisX, axisZ := filled[0][1] || filled[2][1], filled[1][0] || filled[1][2]
	for x := 0; x < 3; x++ {
		for z := 0; z < 3; z++ {
			if !filled[x][z] {
				continue
			}
			b := box{[3]float32{breaks[x], 0, breaks[z]}, [3]float32{breaks[x+1], 1, breaks[z+1]}}
			cell := blockModel{mode: mode}
			appendModelBox(&cell, b, broad)
			for face := 0; face < 6; face++ {
				d := directions[face]
				nx, nz := x+d[0], z+d[2]
				if d[1] == 0 && nx >= 0 && nx < 3 && nz >= 0 && nz < 3 && filled[nx][nz] {
					continue
				}
				q := cell.quads[face]
				wide := face < 2 && x == 1 && (axisZ || !axisX) || face >= 4 && z == 1 && (axisX || !axisZ)
				tex := broad
				if !wide {
					tex = edge
				}
				q.texture = tex
				if q.mode == "opaque" && tex.alpha {
					q.mode = "mask"
				}
				for i, p := range q.points {
					u, v := p[0], 1-p[1]
					if face < 2 {
						u = p[2]
					}
					if !wide {
						u = p[2]
						if face < 2 {
							u = p[2]
						} else {
							u = p[0]
						}
						if face == 2 || face == 3 {
							u, v = p[0], p[2]
							if x != 1 {
								u, v = p[2], p[0]
							}
						}
					}
					q.uv[i] = [2]float32{(float32(tex.x) + u*float32(tex.w)) / atlasSize, (float32(tex.y) + v*float32(tex.h)) / atlasSize}
				}
				m.quads = append(m.quads, q)
			}
		}
	}
	return m.quads
}

func readConnections(props map[string]any) *[4]bool {
	var out [4]bool
	known := false
	for i, d := range []string{"east", "west", "south", "north"} {
		v, ok := props["minecraft:connection_"+d]
		if ok {
			known = true
			out[i] = flag(v)
		}
	}
	if !known {
		return nil
	}
	return &out
}
func readWallConnections(props map[string]any) (*[4]float32, *bool) {
	var out [4]float32
	known := false
	for i, d := range []string{"east", "west", "south", "north"} {
		v, ok := props["wall_connection_type_"+d]
		if ok {
			known = true
		}
		switch v {
		case "short":
			out[i] = .8125
		case "tall":
			out[i] = 1
		}
	}
	var post *bool
	if v, ok := props["wall_post_bit"]; ok {
		b := flag(v)
		post = &b
	}
	if !known {
		return nil, post
	}
	return &out, post
}
