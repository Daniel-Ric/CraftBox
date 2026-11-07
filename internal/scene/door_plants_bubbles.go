package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl32"
	"strings"
)

type doorState struct {
	facing             cube.Direction
	open, right, upper bool
}

func readDoorState(props map[string]any) doorState {
	d := doorState{facing: cube.North, open: flag(props["open_bit"]), right: flag(props["door_hinge_bit"]), upper: flag(props["upper_block_bit"])}
	if cardinal, ok := props["minecraft:cardinal_direction"].(string); ok {
		for _, direction := range cube.Directions() {
			if direction.String() == cardinal {
				d.facing = direction.RotateLeft()
				break
			}
		}
	} else {
		d.facing = cube.Directions()[number(props["direction"])&3]
	}
	return d
}

func (c *Catalog) doorTexture(name string, upper bool) tile {
	face := 2
	if upper {
		face = 4
	}
	short := strings.TrimPrefix(name, "minecraft:")
	faces := c.blocks[short]
	key := faces[faceNames[face]]
	if key == "" {
		key = faces["side"]
	}
	if key == "door_upper" || key == "door_lower" {
		variant := map[string]int{"wooden_door": 0, "spruce_door": 1, "birch_door": 2, "jungle_door": 3, "acacia_door": 4, "dark_oak_door": 5, "iron_door": 6}[short]
		if paths := c.terrainVariants[key]; variant < len(paths) && paths[variant] != "" {
			return c.materialTexture(map[string]any{"texture": paths[variant]}, 16, 16)
		}
	}
	return c.texture(name, face)
}

func (c *Catalog) doorModel(name string, state doorState) blockModel {
	m := blockModel{name: name, mode: "mask", door: &state}
	tex := c.doorTexture(name, state.upper)
	appendModelBox(&m, box{[3]float32{0, 0, 13.0 / 16}, [3]float32{1, 1, 1}}, tex)
	for face := range m.quads {
		q := &m.quads[face]
		for i, p := range q.points {
			u, v := p[0], 1-p[1]
			switch face {
			case 0, 1:
				u = 1 - p[2]
			case 2, 3:
				v = 1 - p[2]
			}
			if state.right {
				u = 1 - u
			}
			q.uv[i] = [2]float32{(float32(tex.x) + u*float32(tex.w)) / atlasSize, (float32(tex.y) + v*float32(tex.h)) / atlasSize}
		}
	}
	angle := float32(0)
	switch state.facing {
	case cube.West:
		angle = 90
	case cube.South:
		angle = 180
	case cube.East:
		angle = -90
	}
	if state.open {
		if state.right {
			angle += 90
		} else {
			angle -= 90
		}
	}
	transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, angle, 0}))
	return m
}

func (c *Catalog) azaleaModel(name string) blockModel {
	m := blockModel{name: name, mode: "mask"}

	appendModelBox(&m, box{[3]float32{0, .5, 0}, [3]float32{1, 1, 1}}, c.texture(name, 5))
	for face := 2; face <= 3; face++ {
		q := &m.quads[face]
		tex := c.texture(name, 2)
		q.texture = tex
		for i, p := range q.points {
			q.uv[i] = [2]float32{(float32(tex.x) + 16*p[0]) / atlasSize, (float32(tex.y) + 16*p[2]) / atlasSize}
		}
	}
	for _, points := range [][4][3]float32{{{0, 0, 0}, {1, 0, 1}, {1, .5, 1}, {0, .5, 0}}, {{1, 0, 0}, {0, 0, 1}, {0, .5, 1}, {1, .5, 0}}} {
		tex := c.texture(name, 0)
		q := texturedQuad(points, tex)
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, .5}, {0, .5}} {
			q.uv[i] = [2]float32{(float32(tex.x) + 16*uv[0]) / atlasSize, (float32(tex.y) + 16*uv[1]) / atlasSize}
		}
		m.quads = append(m.quads, q)
	}
	return m
}

func (c *Catalog) bubbleQuads(x, y, z int) []modelQuad {
	tex := c.materialTexture(map[string]any{"texture": "textures/particle/bubble_white"}, 16, 16)
	if tex.missing {
		return nil
	}
	seed := uint32(x)*73856093 ^ uint32(y)*19349663 ^ uint32(z)*83492791
	random := func() float32 { seed = seed*1664525 + 1013904223; return float32(seed>>8) / 16777216 }
	var quads []modelQuad
	for i := 0; i < 3; i++ {
		cx, cy, cz := .15+.7*random(), .12+.76*random(), .15+.7*random()
		r := .025 + .025*random()
		for axis := 0; axis < 3; axis++ {
			points := [4][3]float32{{-r, -r, 0}, {r, -r, 0}, {r, r, 0}, {-r, r, 0}}
			for j, p := range points {
				v := mgl32.Vec3(p)
				if axis == 1 {
					v = mgl32.Vec3{p[2], p[1], p[0]}
				} else if axis == 2 {
					v = mgl32.Vec3{p[0], p[2], p[1]}
				}
				points[j] = [3]float32{cx + v[0], cy + v[1], cz + v[2]}
			}
			q := texturedQuad(points, tex)
			q.mode = "blend"
			for j, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
				q.uv[j] = [2]float32{(float32(tex.x) + float32(tex.w)*uv[0]) / atlasSize, (float32(tex.y) + float32(tex.h)*uv[1]) / atlasSize}
			}
			quads = append(quads, q)
		}
	}
	return quads
}
