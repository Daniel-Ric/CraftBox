package scene

import (
	"github.com/go-gl/mathgl/mgl32"
	"strings"
)

func texturedQuad(points [4][3]float32, tex tile) modelQuad {
	q := modelQuad{points: points, texture: tex, mode: "mask"}
	for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
		q.uv[i] = [2]float32{(float32(tex.x) + 16*uv[0]) / atlasSize, (float32(tex.y) + 16*uv[1]) / atlasSize}
	}
	n := mgl32.Vec3(points[1]).Sub(mgl32.Vec3(points[0])).Cross(mgl32.Vec3(points[2]).Sub(mgl32.Vec3(points[0]))).Normalize()
	q.normal = [3]float32(n)
	return q
}

func (c *Catalog) crystalModel(name, face string) blockModel {
	m := blockModel{name: name, mode: "mask"}

	const lo, hi = float32(.14644661), float32(.85355339)
	for _, points := range [][4][3]float32{{{lo, 0, lo}, {hi, 0, hi}, {hi, 1, hi}, {lo, 1, lo}}, {{hi, 0, lo}, {lo, 0, hi}, {lo, 1, hi}, {hi, 1, lo}}} {
		for i, p := range points {
			x, y, z := p[0]-.5, p[1]-.5, p[2]-.5
			switch face {
			case "down":
				y, z = -y, -z
			case "north":
				y, z = z, -y
			case "south":
				y, z = -z, y
			case "west":
				x, y = -y, x
			case "east":
				x, y = y, -x
			}
			points[i] = [3]float32{x + .5, y + .5, z + .5}
		}
		m.quads = append(m.quads, texturedQuad(points, c.texture(name, 0)))
	}
	return m
}

func anvilModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque", boxes: []box{
		{[3]float32{2.0 / 16, 0, 2.0 / 16}, [3]float32{14.0 / 16, 4.0 / 16, 14.0 / 16}},
		{[3]float32{4.0 / 16, 4.0 / 16, 3.0 / 16}, [3]float32{12.0 / 16, 5.0 / 16, 13.0 / 16}},
		{[3]float32{6.0 / 16, 5.0 / 16, 4.0 / 16}, [3]float32{10.0 / 16, 10.0 / 16, 12.0 / 16}},
		{[3]float32{3.0 / 16, 10.0 / 16, 0}, [3]float32{13.0 / 16, 1, 1}},
	}}
	if props["minecraft:cardinal_direction"] == "east" || props["minecraft:cardinal_direction"] == "west" || number(props["direction"])%2 == 1 {
		for i := range m.boxes {
			b := &m.boxes[i]
			b.min[0], b.min[2] = b.min[2], b.min[0]
			b.max[0], b.max[2] = b.max[2], b.max[0]
		}
	}
	return m
}

func (c *Catalog) flowerPotModel(plant string) blockModel {
	m := blockModel{name: "minecraft:flower_pot", mode: "mask"}
	b := box{[3]float32{5.0 / 16, 0, 5.0 / 16}, [3]float32{11.0 / 16, 6.0 / 16, 11.0 / 16}}
	tex := c.texture(m.name, 0)
	for face := range directions {
		if face == 2 {
			continue
		}
		var pts [4][3]float32
		for i, p := range corners[face] {
			for a := range p {
				pts[i][a] = b.min[a]
				if p[a] != 0 {
					pts[i][a] = b.max[a]
				}
			}
		}
		q := texturedQuad(pts, tex)

		for i := range q.uv {
			u, v := [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}[i][0], [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}[i][1]
			q.uv[i] = [2]float32{(float32(tex.x) + 5 + 6*u) / atlasSize, (float32(tex.y) + 10 + 6*v) / atlasSize}
		}
		m.quads = append(m.quads, q)
	}

	for _, r := range [][4]float32{{5, 5, 11, 6}, {5, 10, 11, 11}, {5, 6, 6, 10}, {10, 6, 11, 10}} {
		q := texturedQuad([4][3]float32{{r[0] / 16, 6.0 / 16, r[3] / 16}, {r[2] / 16, 6.0 / 16, r[3] / 16}, {r[2] / 16, 6.0 / 16, r[1] / 16}, {r[0] / 16, 6.0 / 16, r[1] / 16}}, tex)
		for i := range q.uv {
			q.uv[i] = [2]float32{(float32(tex.x) + 5.5) / atlasSize, (float32(tex.y) + 10.5) / atlasSize}
		}
		m.quads = append(m.quads, q)
	}
	m.quads = append(m.quads, texturedQuad([4][3]float32{{6.0 / 16, 5.0 / 16, 10.0 / 16}, {10.0 / 16, 5.0 / 16, 10.0 / 16}, {10.0 / 16, 5.0 / 16, 6.0 / 16}, {6.0 / 16, 5.0 / 16, 6.0 / 16}}, c.texture("minecraft:dirt", 2)))
	if plant != "" && plant != "minecraft:air" {
		tex := c.texture(plant, 0)
		for _, pts := range [][4][3]float32{{{.2, 5.0 / 16, .2}, {.8, 5.0 / 16, .8}, {.8, 21.0 / 16, .8}, {.2, 21.0 / 16, .2}}, {{.8, 5.0 / 16, .2}, {.2, 5.0 / 16, .8}, {.2, 21.0 / 16, .8}, {.8, 21.0 / 16, .2}}} {
			m.quads = append(m.quads, texturedQuad(pts, tex))
		}
	}
	return m
}

func isCrystal(name string) bool {
	return name == "amethyst_cluster" || strings.HasSuffix(name, "_amethyst_bud")
}
