package scene

import (
	"github.com/go-gl/mathgl/mgl32"
	"io/fs"
	"strings"
)

func transformQuads(m *blockModel, matrix mgl32.Mat4) {
	for qi := range m.quads {
		q := &m.quads[qi]
		for i, v := range q.points {
			p := matrix.Mul4x1(mgl32.Vec4{v[0], v[1], v[2], 1})
			q.points[i] = [3]float32{p[0], p[1], p[2]}
		}
		n := matrix.Mul4x1(mgl32.Vec4{q.normal[0], q.normal[1], q.normal[2], 0}).Vec3()
		if n.Len() > 0 {
			q.normal = [3]float32(n.Normalize())
		}
	}
}
func (c *Catalog) chainModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "mask"}
	tex := c.texture(name, 0)
	for _, pts := range [][4][3]float32{{{.40625, 0, .5}, {.59375, 0, .5}, {.59375, 1, .5}, {.40625, 1, .5}}, {{.5, 0, .40625}, {.5, 0, .59375}, {.5, 1, .59375}, {.5, 1, .40625}}} {
		q := texturedQuad(pts, tex)
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			q.uv[i] = [2]float32{(float32(tex.x) + uv[0]*3) / atlasSize, (float32(tex.y) + uv[1]*16) / atlasSize}
		}
		m.quads = append(m.quads, q)
	}
	rotation := [3]float32{}
	transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, 45, 0}))
	if props["pillar_axis"] == "x" {
		rotation[2] = 90
	} else if props["pillar_axis"] == "z" {
		rotation[0] = 90
	}
	transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, rotation))
	return m
}

func signMatrix(name string, props map[string]any) mgl32.Mat4 {
	angle := -float32(number(props["ground_sign_direction"])) * 22.5
	if strings.Contains(name, "wall_sign") {
		switch number(props["facing_direction"]) {
		case 2:
			angle = 180
		case 3:
			angle = 0
		case 4:
			angle = 270
		case 5:
			angle = 90
		}
	}
	if strings.Contains(name, "hanging_sign") {
		angle = -float32(number(props["ground_sign_direction"])) * 22.5
		if props["ground_sign_direction"] == nil {
			switch number(props["facing_direction"]) {
			case 2:
				angle = 180
			case 4:
				angle = 270
			case 5:
				angle = 90
			}
		}
	}
	return rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, angle, 0})
}
func (c *Catalog) signModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	wall := strings.Contains(name, "wall_sign")
	hanging := strings.Contains(name, "hanging_sign")
	y0, y1, z0, z1 := float32(.5), float32(1), float32(7.0/16), float32(9.0/16)
	if wall {
		y0, y1, z0, z1 = .28125, .78125, 0, .125
	}
	if hanging {
		y0, y1 = .25, .6875
	}
	boxes := []box{{[3]float32{0, y0, z0}, [3]float32{1, y1, z1}}}
	if !wall && !hanging {
		boxes = append(boxes, box{[3]float32{.4375, 0, .4375}, [3]float32{.5625, .5, .5625}})
	}
	wood := strings.TrimPrefix(name, "minecraft:")
	for _, suffix := range []string{"_standing_sign", "_wall_sign", "_hanging_sign", "_sign"} {
		wood = strings.TrimSuffix(wood, suffix)
	}
	if wood == "standing" || wood == "wall" || wood == "" {
		wood = "oak"
	}
	if wood == "darkoak" {
		wood = "dark_oak"
	}
	texture := c.texture("minecraft:"+wood+"_planks", 4)
	if texture.missing {
		texture = c.materialTexture(map[string]any{"texture": "textures/blocks/planks_" + wood}, 16, 16)
	}
	entityTexture := "textures/entity/sign_" + wood
	if wood == "oak" {
		entityTexture = "textures/entity/sign"
	}
	if wood == "dark_oak" {
		entityTexture = "textures/entity/sign_darkoak"
	}
	if wood == "bamboo" || wood == "cherry" || wood == "mangrove" || wood == "pale_oak" {
		entityTexture = "textures/entity/" + wood + "_sign"
	}
	if hanging {
		entityTexture = "textures/entity/" + wood + "_hanging_sign"
	}
	var original tile
	hasOriginal := false
	for _, source := range c.sources {
		if _, err := fs.Stat(source, entityTexture+".png"); err == nil {
			original = c.materialTexture(map[string]any{"texture": entityTexture}, 64, 32)
			hasOriginal = !original.missing
			break
		}
	}
	for bi, b := range boxes {
		for f := range directions {
			var pts [4][3]float32
			for i, v := range corners[f] {
				for a := range v {
					pts[i][a] = b.min[a] + float32(v[a])*(b.max[a]-b.min[a])
				}
			}
			q := texturedQuad(pts, texture)
			if hasOriginal && !hanging {
				q.texture = original
				rect := [4]float32{2, 2, 24, 12}
				switch f {
				case 0:
					rect = [4]float32{26, 2, 2, 12}
				case 1:
					rect = [4]float32{0, 2, 2, 12}
				case 2:
					rect = [4]float32{2, 0, 24, 2}
				case 3:
					rect = [4]float32{26, 0, 24, 2}
				case 5:
					rect = [4]float32{28, 2, 24, 12}
				}
				if bi == 1 {
					rect = [4]float32{2, 16, 2, 14}
				}
				for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
					q.uv[i] = [2]float32{(float32(original.x) + (rect[0]+uv[0]*rect[2])/64*float32(original.w)) / atlasSize, (float32(original.y) + (rect[1]+uv[1]*rect[3])/32*float32(original.h)) / atlasSize}
				}
			}
			m.quads = append(m.quads, q)
		}
	}
	if hanging {
		for _, x := range []float32{.125, .875} {
			chain := c.chainModel("minecraft:iron_chain", nil)
			for _, q := range chain.quads {
				for i, p := range q.points {
					q.points[i] = [3]float32{x + (p[0]-.5)*.65, .6875 + p[1]*.3125, .5 + (p[2]-.5)*.65}
				}
				m.quads = append(m.quads, q)
			}
		}
	}
	transformQuads(&m, signMatrix(name, props))
	return m
}
