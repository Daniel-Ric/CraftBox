package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl32"
	"io/fs"
	"math"
	"strings"
)

func (c *Catalog) frameModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	appendModelBox(&m, box{[3]float32{2.0 / 16, 0, 2.0 / 16}, [3]float32{14.0 / 16, 1.0 / 32, 14.0 / 16}}, c.texture(name, 2))
	wood := c.texture("minecraft:birch_planks", 2)
	for _, b := range []box{{[3]float32{2.0 / 16, 0, 2.0 / 16}, [3]float32{3.0 / 16, 1.0 / 16, 14.0 / 16}}, {[3]float32{13.0 / 16, 0, 2.0 / 16}, [3]float32{14.0 / 16, 1.0 / 16, 14.0 / 16}}, {[3]float32{3.0 / 16, 0, 2.0 / 16}, [3]float32{13.0 / 16, 1.0 / 16, 3.0 / 16}}, {[3]float32{3.0 / 16, 0, 13.0 / 16}, [3]float32{13.0 / 16, 1.0 / 16, 14.0 / 16}}} {
		appendModelBox(&m, b, wood)
	}
	mountFrameModel(&m, mountedFace(props))
	return m
}
func (c *Catalog) framedItemTexture(name string, damage int) tile {
	short := strings.TrimPrefix(name, "minecraft:")
	key := "textures/items/" + short
	for _, source := range c.sources {
		raw, e := fs.ReadFile(source, "textures/item_texture.json")
		if e != nil {
			continue
		}
		raw, e = standardizePackJSON(raw)
		if e != nil {
			continue
		}
		var data struct {
			Textures map[string]struct {
				Texture json.RawMessage `json:"textures"`
			} `json:"texture_data"`
		}
		if json.Unmarshal(raw, &data) == nil {
			texture := data.Textures[short].Texture
			index := damage
			if len(texture) == 0 {
				alias := map[string]string{"bow": "bow_standby", "crossbow": "crossbow_standby", "beef": "beef_raw", "cooked_beef": "beef_cooked", "chicken": "chicken_raw", "cooked_chicken": "chicken_cooked", "mutton": "mutton_raw", "cooked_mutton": "mutton_cooked", "porkchop": "porkchop_raw", "cooked_porkchop": "porkchop_cooked", "cod": "fish", "cooked_cod": "cooked_fish", "golden_apple": "apple_golden", "enchanted_golden_apple": "apple_golden", "tropical_fish": "clownfish"}[short]
				if alias != "" {
					texture = data.Textures[alias].Texture
				}
				for i, wood := range []string{"oak", "spruce", "birch", "jungle", "acacia", "dark_oak", "mangrove", "bamboo", "cherry", "pale_oak", "poplar"} {
					if short == wood+"_boat" || short == wood+"_raft" {
						texture = data.Textures["boat"].Texture
						index = i
						break
					}
				}
			}
			if len(texture) == 0 && strings.HasSuffix(short, "_dye") {
				if dye, ok := map[string]int{"black": 16, "red": 1, "green": 2, "brown": 17, "blue": 18, "purple": 5, "cyan": 6, "light_gray": 7, "gray": 8, "pink": 9, "lime": 10, "yellow": 11, "light_blue": 12, "magenta": 13, "orange": 14, "white": 19}[strings.TrimSuffix(short, "_dye")]; ok {
					texture = data.Textures["dye_powder"].Texture
					index = dye
				}
			}
			if len(texture) == 0 && short == "dye" {
				texture = data.Textures["dye_powder"].Texture
			}
			var variants []json.RawMessage
			if json.Unmarshal(texture, &variants) == nil && len(variants) > 0 {
				texture = variants[max(0, min(len(variants)-1, index))]
			}
			if p := texturePath(texture); p != "" {
				key = p
				break
			}
		}
	}
	return c.materialTexture(map[string]any{"texture": key}, 16, 16)
}
func (s *Store) frameContents(props, data map[string]any) []modelQuad {
	item, _ := data["Item"].(map[string]any)
	if item == nil {
		return nil
	}
	if count, ok := item["Count"]; ok && number(count) == 0 {
		return nil
	}
	name, _ := item["Name"].(string)
	if name == "" {
		name, _ = item["name"].(string)
	}
	block, _ := item["Block"].(map[string]any)
	states, _ := block["states"].(map[string]any)
	if n, ok := block["name"].(string); ok && n != "" {
		name = n
	}
	if name == "" {
		return nil
	}
	normalized := map[string]any{}
	defaults, _ := s.registry.DefaultProperties(name)
	for k, v := range defaults {
		normalized[k] = v
	}
	for k, v := range states {
		if n, ok := v.(float64); ok {
			switch defaults[k].(type) {
			case uint8:
				v = uint8(n)
			case bool:
				v = n != 0
			default:
				v = int32(n)
			}
		}
		normalized[k] = v
	}
	var content blockModel
	isBlock := false
	if name != "minecraft:frame" && name != "minecraft:glow_frame" {
		if _, known := s.registry.DefaultProperties(name); known {
			if rid, ok := s.registry.StateToRuntimeID(name, normalized); ok {
				content = s.model(rid)
				isBlock = true
			}
		}
		if !isBlock {
			content, isBlock = s.catalog.customModel(name, normalized)
		}
	}
	if isBlock {
		short := strings.TrimPrefix(name, "minecraft:")
		if strings.Contains(short, "glass_pane") || short == "iron_bars" || content.cross || short == "ladder" || short == "vine" || short == "rail" || strings.HasSuffix(short, "_rail") {
			face := 4
			if strings.HasSuffix(short, "rail") {
				face = 3
			}
			tex := s.catalog.texture(name, face)
			q := frameSprite(tex)
			if strings.Contains(short, "glass_pane") {
				q.mode = "blend"
			}
			content.quads = []modelQuad{q}
			isBlock = false
		}
	}
	if isBlock {
		if content.cross && len(content.quads) == 0 {
			tex := s.catalog.texture(name, content.plantFace)
			for _, pts := range [][4][3]float32{{{0, 0, 0}, {1, 0, 1}, {1, 1, 1}, {0, 1, 0}}, {{1, 0, 0}, {0, 0, 1}, {0, 1, 1}, {1, 1, 0}}} {
				content.quads = append(content.quads, texturedQuad(pts, tex))
			}
		}
		if len(content.quads) > 0 {
			content.quads = append([]modelQuad{}, content.quads...)
		} else {
			for _, b := range content.boxes {
				for face, d := range directions {
					var pts [4][3]float32
					for i, p := range corners[face] {
						for a := range p {
							pts[i][a] = b.min[a]
							if p[a] == 1 {
								pts[i][a] = b.max[a]
							}
						}
					}
					tex := s.catalog.texture(name, face)
					p := newPart(content.mode)
					emitBoxQuad(p, pts, [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, [3]float32{}, tex, face)
					q := modelQuad{points: pts, normal: [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, texture: tex, mode: content.mode}
					for i := range q.uv {
						q.uv[i] = [2]float32{p.UVs[i*2], p.UVs[i*2+1]}
					}
					content.quads = append(content.quads, q)
				}
			}
		}

		matrix := mgl32.Translate3D(.5, .32, .5).Mul4(mgl32.Scale3D(.35, .35, .35)).Mul4(mgl32.HomogRotate3DX(-math.Pi / 3)).Mul4(mgl32.HomogRotate3DY(5 * math.Pi / 4)).Mul4(mgl32.Translate3D(-.5, -.5, -.5))
		transformQuads(&content, matrix)
	} else if len(content.quads) == 0 {
		tex := s.catalog.framedItemTexture(name, number(item["Damage"]))
		content.quads = []modelQuad{frameSprite(tex)}
	}
	transformQuads(&content, rotationMatrix([3]float32{.5, 0, .5}, [3]float32{0, -frameRotationDegrees(data), 0}))
	mountFrameModel(&content, mountedFace(props))
	return content.quads
}

func frameSprite(tex tile) modelQuad {
	q := texturedQuad([4][3]float32{{.25, .08, .75}, {.75, .08, .75}, {.75, .08, .25}, {.25, .08, .25}}, tex)
	for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
		q.uv[i] = [2]float32{(float32(tex.x) + uv[0]*float32(tex.w)) / atlasSize, (float32(tex.y) + uv[1]*float32(tex.h)) / atlasSize}
	}
	return q
}
func mountFrameModel(m *blockModel, face int) {
	n := [3]float32{0, 1, 0}
	switch face {
	case 0:
		n = [3]float32{0, -1, 0}
	case 2:
		n = [3]float32{0, 0, -1}
	case 3:
		n = [3]float32{0, 0, 1}
	case 4:
		n = [3]float32{-1, 0, 0}
	case 5:
		n = [3]float32{1, 0, 0}
	}
	v := mgl32.Vec3{0, 1, 0}
	if face == 0 {
		v = mgl32.Vec3{0, 0, 1}
	}
	if face == 1 {
		v = mgl32.Vec3{0, 0, -1}
	}
	u := v.Cross(mgl32.Vec3(n))
	rotation := mgl32.Mat4{u[0], u[1], u[2], 0, n[0], n[1], n[2], 0, -v[0], -v[1], -v[2], 0, 0, 0, 0, 1}
	transformQuads(m, mgl32.Translate3D(.5, .5, .5).Mul4(rotation).Mul4(mgl32.Translate3D(-.5, -.5, -.5)))
}

const frameRotationKey = "craftbox:frame_rotation_degrees"

func frameRotationDegrees(data map[string]any) float32 {
	value := data["ItemRotation"]
	_, degrees := value.(float32)
	if stored, ok := data[frameRotationKey]; ok {
		value = stored
		degrees = true
	}
	angle := numeric(value)
	if !degrees && math.Abs(angle) <= 7 && math.Trunc(angle) == angle {
		angle *= 45
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return 0
	}
	return float32(math.Mod(angle, 360))
}

func (s *Store) framedModelAt(base blockModel, props map[string]any, pos cube.Pos) blockModel {
	m := base
	m.quads = append([]modelQuad{}, base.quads...)
	m.quads = append(m.quads, s.frameContents(props, s.frames[pos])...)
	return m
}
