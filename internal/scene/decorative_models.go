package scene

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func appendSkinBox(m *blockModel, b box, tex tile, uv [2]float32, tw, th float32) {
	dx, dy, dz := (b.max[0]-b.min[0])*16, (b.max[1]-b.min[1])*16, (b.max[2]-b.min[2])*16
	starts := [6][2]float32{{uv[0] + dz + dx, uv[1] + dz}, {uv[0], uv[1] + dz}, {uv[0] + dz, uv[1]}, {uv[0] + dz + dx, uv[1]}, {uv[0] + dz + dx + dz, uv[1] + dz}, {uv[0] + dz, uv[1] + dz}}
	for face, d := range directions {
		q := modelQuad{texture: tex, mode: m.mode, normal: [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}}
		w, h := dx, dy
		if face < 2 {
			w = dz
		}
		if face == 2 || face == 3 {
			h = dz
		}
		for i, p := range corners[face] {
			for a := 0; a < 3; a++ {
				q.points[i][a] = b.min[a]
				if p[a] != 0 {
					q.points[i][a] = b.max[a]
				}
			}
			u, v := float32(1-p[0]), float32(1-p[1])
			switch face {
			case 0:
				u = float32(1 - p[2])
			case 1:
				u = float32(p[2])
			case 2:
				u, v = float32(p[0]), float32(p[2])
			case 3:
				u, v = float32(p[0]), float32(1-p[2])
			case 5:
				u = float32(p[0])
			}
			q.uv[i] = [2]float32{(float32(tex.x) + (starts[face][0]+u*w)/tw*float32(tex.w)) / atlasSize, (float32(tex.y) + (starts[face][1]+v*h)/th*float32(tex.h)) / atlasSize}
		}
		m.quads = append(m.quads, q)
	}
}
func rotateFacing(m *blockModel, props map[string]any) {
	angle := float32(0)
	switch number(props["facing_direction"]) {
	case 3:
		angle = 180
	case 4:
		angle = 90
	case 5:
		angle = 270
	}
	switch props["minecraft:cardinal_direction"] {
	case "north":
		angle = 0
	case "south":
		angle = 180
	case "west":
		angle = 90
	case "east":
		angle = 270
	}
	transformQuads(m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, angle, 0}))
}
func (c *Catalog) chestModel(name string, props, data map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque", entityVisual: "chest"}
	if flag(data["render_secondary"]) {
		return m
	}
	skin := "normal"
	if name == "minecraft:trapped_chest" {
		skin = "trapped"
	}
	if name == "minecraft:ender_chest" {
		skin = "ender"
	}
	width, tw, center := float32(15.0/16), float32(64), float32(.5)
	if flag(data["render_double"]) && name != "minecraft:ender_chest" {
		width, tw, center = 31.0/16, 128, 1
		if skin == "normal" {
			skin = "double_normal"
		} else {
			skin = "trapped_double"
		}
	}
	tex := c.materialTexture(map[string]any{"texture": "textures/entity/chest/" + skin}, tw, 64)
	appendSkinBox(&m, box{[3]float32{1.0 / 16, 0, 1.0 / 16}, [3]float32{width, 10.0 / 16, 15.0 / 16}}, tex, [2]float32{0, 19}, tw, 64)
	appendSkinBox(&m, box{[3]float32{1.0 / 16, 10.0 / 16, 1.0 / 16}, [3]float32{width, 14.0 / 16, 15.0 / 16}}, tex, [2]float32{}, tw, 64)
	appendSkinBox(&m, box{[3]float32{center - 1.0/16, 8.0 / 16, 0}, [3]float32{center + 1.0/16, 12.0 / 16, 2.0 / 16}}, tex, [2]float32{}, tw, 64)
	rotateFacing(&m, props)
	return m
}

var bedColors = []string{"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray", "silver", "cyan", "purple", "blue", "brown", "green", "red", "black"}

func (c *Catalog) bedModel(name string, props, data map[string]any) blockModel {
	colorIndex := 14
	if v, ok := data["color"]; ok {
		colorIndex = max(0, min(15, number(v)))
	}
	tex := c.materialTexture(map[string]any{"texture": "textures/entity/bed/" + bedColors[colorIndex]}, 64, 64)
	m := blockModel{name: name, mode: "opaque", entityVisual: "bed"}
	head := flag(props["head_piece_bit"])
	uv := [2]float32{0, 22}
	if head {
		uv[1] = 0
	}
	appendSkinBox(&m, box{[3]float32{}, [3]float32{1, 1, 6.0 / 16}}, tex, uv, 64, 64)

	for qi := range m.quads {
		q := &m.quads[qi]
		for i, p := range q.points {
			q.points[i] = [3]float32{p[0], 9.0/16 - p[2], p[1]}
		}
		q.normal = [3]float32{q.normal[0], -q.normal[2], q.normal[1]}
	}
	z := float32(0)
	if head {
		z = 13.0 / 16
	}
	for i, x := range []float32{0, 13.0 / 16} {
		appendSkinBox(&m, box{[3]float32{x, 0, z}, [3]float32{x + 3.0/16, 3.0 / 16, z + 3.0/16}}, tex, [2]float32{float32(i * 6), 44}, 64, 64)
	}

	transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, -float32(number(props["direction"]) * 90), 0}))
	return m
}
func (c *Catalog) torchModel(name string, props map[string]any) blockModel {
	tex := c.texture(name, 4)
	m := blockModel{name: name, mode: "mask"}
	appendModelBox(&m, box{[3]float32{7.0 / 16, 0, 7.0 / 16}, [3]float32{9.0 / 16, 10.0 / 16, 9.0 / 16}}, tex)
	for face := range m.quads {
		q := &m.quads[face]
		for i, p := range corners[face] {
			u, v := float32(7+p[0]*2), float32(16-p[1]*10)
			if face < 2 {
				u = float32(7 + p[2]*2)
			}
			if face == 2 || face == 3 {
				u, v = float32(7+p[0]*2), float32(6+p[2]*2)
			}
			q.uv[i] = [2]float32{(float32(tex.x) + u/16*float32(tex.w)) / atlasSize, (float32(tex.y) + v/16*float32(tex.h)) / atlasSize}
		}
	}
	rotation := [3]float32{}
	offset := [3]float32{0, .2, 0}
	switch props["torch_facing_direction"] {
	case "north":
		rotation[0] = -22.5
		offset[2] = .25
	case "south":
		rotation[0] = 22.5
		offset[2] = -.25
	case "west":
		rotation[2] = 22.5
		offset[0] = .25
	case "east":
		rotation[2] = -22.5
		offset[0] = -.25
	default:
		return m
	}
	transformQuads(&m, rotationMatrix([3]float32{.5, 0, .5}, rotation))
	for qi := range m.quads {
		for i := range m.quads[qi].points {
			for a := range offset {
				m.quads[qi].points[i][a] += offset[a]
			}
		}
	}
	return m
}
func (c *Catalog) wallCoralModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "mask"}
	tex := c.coralTexture(name, props)
	for _, pts := range [][4][3]float32{{{0, .25, 0}, {1, .25, 0}, {1, .625, .5}, {0, .625, .5}}, {{0, .75, 0}, {1, .75, 0}, {1, .375, .5}, {0, .375, .5}}} {
		m.quads = append(m.quads, texturedQuad(pts, tex))
	}
	transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, float32(number(props["coral_direction"]) * 90), 0}))
	return m
}

var bannerDyes = []color.NRGBA{{29, 29, 33, 255}, {176, 46, 38, 255}, {94, 124, 22, 255}, {131, 84, 50, 255}, {60, 68, 170, 255}, {137, 50, 184, 255}, {22, 156, 156, 255}, {157, 157, 151, 255}, {71, 79, 82, 255}, {243, 139, 170, 255}, {128, 199, 31, 255}, {254, 216, 61, 255}, {58, 179, 218, 255}, {199, 78, 189, 255}, {249, 128, 29, 255}, {249, 255, 254, 255}}

func (c *Catalog) bannerTexture(data map[string]any) (tile, bool) {
	raw, _ := json.Marshal(map[string]any{"Base": data["Base"], "Patterns": data["Patterns"]})
	key := "banner-render:" + string(raw)
	if t, ok := c.tiles[key]; ok {
		return t, c.bannerApprox[key]
	}
	base := c.materialTexture(map[string]any{"texture": "textures/entity/banner/banner_base"}, 64, 64)
	if base.missing {
		return base, true
	}
	img := image.NewNRGBA(image.Rect(0, 0, base.w, base.h))
	dye := bannerDyes[max(0, min(15, number(data["Base"])))]
	for y := 0; y < base.h; y++ {
		for x := 0; x < base.w; x++ {
			v := c.atlas.NRGBAAt(base.x+x, base.y+y)
			img.SetNRGBA(x, y, color.NRGBA{uint8(uint16(v.R) * uint16(dye.R) / 255), uint8(uint16(v.G) * uint16(dye.G) / 255), uint8(uint16(v.B) * uint16(dye.B) / 255), v.A})
		}
	}
	patterns := []map[string]any{}
	switch p := data["Patterns"].(type) {
	case []map[string]any:
		patterns = p
	case []any:
		for _, v := range p {
			if m, ok := v.(map[string]any); ok {
				patterns = append(patterns, m)
			}
		}
	}
	approximate := false
	names := map[string]string{"bs": "stripe_bottom", "ts": "stripe_top", "ls": "stripe_left", "rs": "stripe_right", "cs": "stripe_center", "ms": "stripe_middle", "drs": "stripe_downright", "dls": "stripe_downleft", "ss": "small_stripes", "cr": "cross", "sc": "straight_cross", "bt": "triangle_bottom", "tt": "triangle_top", "bts": "triangles_bottom", "tts": "triangles_top", "ld": "diagonal_left", "rd": "diagonal_right", "lud": "diagonal_up_left", "rud": "diagonal_up_right", "mc": "circle", "mr": "rhombus", "vh": "half_vertical", "hh": "half_horizontal", "vhr": "half_vertical_right", "hhb": "half_horizontal_bottom", "bo": "border", "cbo": "curly_border", "bri": "bricks", "gra": "gradient", "gru": "gradient_up", "cre": "creeper", "sku": "skull", "flo": "flower", "moj": "mojang", "glb": "globe", "pig": "piglin", "flw": "flow", "gus": "guster"}
	for _, pattern := range patterns {
		code, _ := pattern["Pattern"].(string)
		name := names[code]
		if name == "" {
			approximate = true
			continue
		}
		layer := c.materialTexture(map[string]any{"texture": "textures/entity/banner/banner_" + name}, 64, 64)
		if layer.missing {
			approximate = true
			continue
		}
		col := bannerDyes[max(0, min(15, number(pattern["Color"])))]
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				mask := c.atlas.NRGBAAt(layer.x+x*layer.w/base.w, layer.y+y*layer.h/base.h)
				old := img.NRGBAAt(x, y)
				a := uint32(mask.A)
				old.R = uint8((uint32(old.R)*(255-a) + uint32(mask.R)*uint32(col.R)*a/255) / 255)
				old.G = uint8((uint32(old.G)*(255-a) + uint32(mask.G)*uint32(col.G)*a/255) / 255)
				old.B = uint8((uint32(old.B)*(255-a) + uint32(mask.B)*uint32(col.B)*a/255) / 255)
				img.SetNRGBA(x, y, old)
			}
		}
	}
	result := c.addTileSized(key, img, false, base.w, base.h)
	if c.bannerApprox == nil {
		c.bannerApprox = map[string]bool{}
	}
	c.bannerApprox[key] = approximate
	return result, approximate
}
func (c *Catalog) bannerModel(name string, props, data map[string]any) blockModel {
	tex, approx := c.bannerTexture(data)
	m := blockModel{name: name, mode: "mask", approx: approx, entityVisual: "banner"}
	wall := strings.Contains(name, "wall")
	top, z := float32(1.75), float32(.5)
	if wall {
		top, z = .9375, .875
	}
	appendSkinBox(&m, box{[3]float32{1.0 / 12, top - 5.0/3, z - 1.0/48}, [3]float32{11.0 / 12, top, z + 1.0/48}}, tex, [2]float32{}, 64, 64)
	for _, face := range []int{4, 5} {
		q := &m.quads[face]
		for i, p := range corners[face] {
			u := float32(p[0])
			if face == 4 {
				u = 1 - u
			}
			q.uv[i] = [2]float32{(float32(tex.x) + (1+u*20)/64*float32(tex.w)) / atlasSize, (float32(tex.y) + (1+float32(1-p[1])*40)/64*float32(tex.h)) / atlasSize}
		}
	}
	wood := c.texture("minecraft:oak_planks", 2)
	appendModelBox(&m, box{[3]float32{1.0 / 12, top - 1.0/24, z - 1.0/24}, [3]float32{11.0 / 12, top + 1.0/24, z + 1.0/24}}, wood)
	if !wall {
		appendModelBox(&m, box{[3]float32{11.0 / 24, 0, 11.0 / 24}, [3]float32{13.0 / 24, 1.75, 13.0 / 24}}, wood)
	}
	if wall {
		rotateFacing(&m, props)
	} else {
		transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, -float32(number(props["ground_sign_direction"])) * 22.5, 0}))
	}
	return m
}
func (s *Store) decorativeAt(m *blockModel, props map[string]any, pos cube.Pos) *blockModel {
	data := s.decorativeEntities[pos]
	if data == nil {
		return m
	}
	key := fmt.Sprint(pos, m.name, props)
	if cached := s.decorativeModels[key]; cached != nil {
		return cached
	}
	var out blockModel
	switch m.entityVisual {
	case "lectern":
		out = s.catalog.lecternModel(props, data)
	case "shulker":
		out = s.catalog.shulkerBoxModel(m.name, props, data)
	case "piston":
		out = s.catalog.pistonModel(m.name, props, number(data["State"]) != 0 || number(data["NewState"]) != 0)
	case "bed":
		out = s.catalog.bedModel(m.name, props, data)
	case "banner":
		out = s.catalog.bannerModel(m.name, props, data)
	case "chest":
		visual := map[string]any{}
		if px, ok := data["pairx"]; ok && m.name != "minecraft:ender_chest" {
			if pz, ok := data["pairz"]; ok {
				other := cube.Pos{number(px), pos[1], number(pz)}
				dx, dz := other[0]-pos[0], other[2]-pos[2]
				if abs(dx)+abs(dz) == 1 {
					ch := s.chunks[world.ChunkPos{int32(other[0] >> 4), int32(other[2] >> 4)}]
					if ch != nil {
						name, _, _ := s.registry.RuntimeIDToState(ch.Block(uint8(other[0]&15), int16(other[1]), uint8(other[2]&15), 0))
						if name == m.name {
							unitX, unitZ := 1, 0
							switch props["minecraft:cardinal_direction"] {
							case "south":
								unitX = -1
							case "west":
								unitX, unitZ = 0, -1
							case "east":
								unitX, unitZ = 0, 1
							default:
								switch number(props["facing_direction"]) {
								case 3:
									unitX = -1
								case 4:
									unitX, unitZ = 0, -1
								case 5:
									unitX, unitZ = 0, 1
								}
							}
							if dx == unitX && dz == unitZ {
								visual["render_double"] = true
							} else if dx == -unitX && dz == -unitZ {
								visual["render_secondary"] = true
							}
						}
					}
				}
			}
		}
		out = s.catalog.chestModel(m.name, props, visual)
	default:
		return m
	}
	if s.decorativeModels == nil {
		s.decorativeModels = map[string]*blockModel{}
	}
	s.decorativeModels[key] = &out
	return &out
}
