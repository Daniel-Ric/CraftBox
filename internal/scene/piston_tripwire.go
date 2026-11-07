package scene

import "strings"

func pistonHead(name string) bool {
	switch strings.TrimPrefix(name, "minecraft:") {
	case "piston_arm_collision", "sticky_piston_arm_collision", "pistonArmCollision", "stickyPistonArmCollision":
		return true
	}
	return false
}

func (c *Catalog) pistonWithNeighbour(m *blockModel, props map[string]any, head *blockModel) *blockModel {
	if pistonHead(head.name) && head.pistonFacing == mountedFace(props) && strings.Contains(strings.ToLower(head.name), "sticky") == strings.Contains(m.name, "sticky") {
		out := c.pistonModel(m.name, props, true)
		return &out
	}
	return m
}

// Build in the upward-facing orientation, then rotate geometry, normals and UVs
// together. Bedrock's piston_top is the exposed inner face, not the closed head.
func (c *Catalog) pistonModel(name string, props map[string]any, extended bool) blockModel {
	m := blockModel{name: name, mode: "opaque", pistonFacing: mountedFace(props)}
	texture := func(key string) tile { return c.materialTexture(map[string]any{"texture": key}, 16, 16) }
	frontKey := "piston_top_normal"
	if strings.Contains(strings.ToLower(name), "sticky") {
		frontKey = "piston_top_sticky"
	}
	front, side, bottom := texture(frontKey), texture("piston_side"), texture("piston_bottom")
	if pistonHead(name) {
		appendModelBox(&m, box{[3]float32{0, .75, 0}, [3]float32{1, 1, 1}}, side)
		for i := range m.quads {
			q := &m.quads[i]
			if i == 2 || i == 3 {
				setPistonFaceTexture(q, front)
				continue
			}
			// Position-based box UVs already select the upper wooden band.
		}
		start := len(m.quads)
		appendModelBox(&m, box{[3]float32{.375, -.25, .375}, [3]float32{.625, .75, .625}}, side)
		for i := start; i < len(m.quads); i++ {
			q := &m.quads[i]
			for j, p := range q.points {
				u := p[0] - .375
				if i-start < 2 {
					u = p[2] - .375
				}
				// Rotate the wooden band along the shaft, rather than stretching the
				// cobblestone part of the base texture over it.
				q.uv[j] = [2]float32{(float32(side.x) + (p[1]+.25)*16) / atlasSize, (float32(side.y) + u*16) / atlasSize}
				if i-start == 2 || i-start == 3 {
					q.uv[j] = [2]float32{(float32(side.x) + (p[0]-.375)*16) / atlasSize, (float32(side.y) + (p[2]-.375)*16) / atlasSize}
				}
			}
		}
	} else {
		m.entityVisual = "piston"
		height := float32(1)
		if extended {
			height = .75
			front = texture("piston_top")
		}
		appendModelBox(&m, box{[3]float32{}, [3]float32{1, height, 1}}, side)
		setPistonFaceTexture(&m.quads[2], front)
		setPistonFaceTexture(&m.quads[3], bottom)
	}
	mountModel(&m, mountedFace(props))
	return m
}

func setPistonFaceTexture(q *modelQuad, tex tile) {
	old := q.texture
	for i, uv := range q.uv {
		q.uv[i] = [2]float32{(float32(tex.x) + (uv[0]*atlasSize-float32(old.x))/float32(old.w)*float32(tex.w)) / atlasSize, (float32(tex.y) + (uv[1]*atlasSize-float32(old.y))/float32(old.h)*float32(tex.h)) / atlasSize}
	}
	q.texture = tex
}

func (c *Catalog) tripwireModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "mask", connect: "tripwire"}
	y := float32(1.0 / 64)
	if flag(props["suspended_bit"]) {
		y = 1.5 / 16
	}
	sides := [4]bool{}
	explicit := false
	for i, key := range []string{"minecraft:connection_east", "minecraft:connection_west", "minecraft:connection_south", "minecraft:connection_north"} {
		if v, ok := props[key]; ok {
			explicit = true
			sides[i] = flag(v)
		}
	}
	if explicit {
		m.connections = &sides
	}
	tex := c.materialTexture(map[string]any{"texture": "trip_wire"}, 16, 16)
	m.quads = tripwireQuads(tex, y, sides)
	return m
}

func tripwireQuads(tex tile, y float32, sides [4]bool) []modelQuad {
	if sides == [4]bool{} {
		sides = [4]bool{true, true, true, true}
	}
	quads := []modelQuad{}
	for i, on := range sides {
		if !on {
			continue
		}
		// A two-pixel wide, alpha-masked strand, not a collision/selection box.
		pts := [4][3]float32{{.5, y, .5625}, {1, y, .5625}, {1, y, .4375}, {.5, y, .4375}}
		for j, p := range pts {
			switch i {
			case 1:
				p[0] = 1 - p[0]
				p[2] = 1 - p[2]
			case 2:
				p[0], p[2] = 1-p[2], p[0]
			case 3:
				p[0], p[2] = p[2], 1-p[0]
			}
			pts[j] = p
		}
		q := texturedQuad(pts, tex)
		for j, uv := range [4][2]float32{{0, 2}, {8, 2}, {8, 0}, {0, 0}} {
			q.uv[j] = [2]float32{(float32(tex.x) + uv[0]/16*float32(tex.w)) / atlasSize, (float32(tex.y) + uv[1]/16*float32(tex.h)) / atlasSize}
		}
		quads = append(quads, q)
	}
	return quads
}

func connectedTripwireQuads(m *blockModel, neighbours [4]*blockModel) []modelQuad {
	sides := [4]bool{}
	if m.connections != nil {
		sides = *m.connections
	} else {
		for i, n := range neighbours {
			if n == nil {
				continue
			}
			sides[i] = n.connect == "tripwire"
			if n.name == "minecraft:tripwire_hook" {
				sides[i] = n.tripwireHookDirection == i^1
			}
		}
	}
	return tripwireQuads(m.quads[0].texture, m.quads[0].points[0][1], sides)
}
