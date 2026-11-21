package resourcepack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"path"
	"strings"

	"github.com/Daniel-Ric/CraftBox/utils/skinconverter"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sirupsen/logrus"
)

func (p *Pack) addPlayerSkinLayers(id uuid.UUID, skin *skinconverter.Skin, entity *ClientEntityDescription) error {
	if len(skin.Animations) == 0 {
		return nil
	}
	controllers := map[string]any{}
	attempted := map[string]struct{}{}
	playerName := strings.ReplaceAll(id.String(), "-", "_")
	for _, animation := range skin.Animations {
		var alias string
		switch animation.AnimationType {
		case protocol.SkinAnimationHead:
			alias = "animated_face"
		case protocol.SkinAnimationBody32x32:
			alias = "animated_32x32"
		case protocol.SkinAnimationBody128x128:
			alias = "animated_128x128"
		default:
			continue
		}
		if _, exists := attempted[alias]; exists {
			continue
		}
		width, height, count := animation.ImageWidth, animation.ImageHeight, animation.FrameCount
		if width == 0 || width > 4096 || height == 0 || height > 8192 ||
			!(count >= 1 && count <= 1024) || count != float32(uint32(count)) ||
			height%uint32(count) != 0 || uint64(len(animation.ImageData)) != uint64(width)*uint64(height)*4 {
			continue
		}

		attempted[alias] = struct{}{}
		version, geometry, err := skin.ParseGeometryLayer(alias)
		if err != nil {
			logrus.Warnf("player %s: could not read skin layer %s: %v", id, alias, err)
			continue
		}
		if geometry == nil {
			continue
		}

		name := playerName + "." + alias
		geometryID := "geometry.craftbox_player." + name
		geometry.Description.Identifier = geometryID
		if version == "" || version == "1.8.0" {
			version = "1.12.0"
		}
		model, err := json.Marshal(skinconverter.SkinGeometryFile{FormatVersion: version, Geometry: []skinconverter.SkinGeometry{*geometry}})
		if err != nil {
			return err
		}

		texture := &image.NRGBA{Pix: animation.ImageData, Stride: int(width) * 4, Rect: image.Rect(0, 0, int(width), int(height))}
		var data bytes.Buffer
		if err := png.Encode(&data, texture); err != nil {
			return err
		}
		texturePath := path.Join("textures", "player", name)
		p.Files[texturePath+".png"] = data.Bytes()
		p.Files[path.Join("models", "entity", "craftbox_player", geometryID+".geo.json")] = model

		controllerID := "controller.render.craftbox_player." + name
		var offset any = 0
		if count > 1 {
			frames := int(count)
			if animation.AnimationType == protocol.SkinAnimationHead && animation.ExpressionType == protocol.ExpressionTypeBlinking {
				offset = fmt.Sprintf("math.mod(query.life_time, 4.3) > 4.15 ? 1.0 / %d : 0.0", frames)
			} else {
				offset = fmt.Sprintf("math.mod(math.floor(query.life_time * 7), %d) / %d", frames, frames)
			}
		}
		controllers[controllerID] = map[string]any{
			"rebuild_animation_matrices": true,
			"geometry":                   "Geometry." + alias,
			"textures":                   []string{"Texture." + alias},
			"materials":                  []map[string]string{{"*": "Material." + alias}},
			"uv_anim":                    map[string]any{"offset": []any{0, offset}, "scale": []int{1, 1}},
		}
		entity.Geometry[alias] = geometryID
		entity.Textures[alias] = texturePath
		entity.Materials[alias] = "player_animated"
		entity.RenderControllers = append(entity.RenderControllers, controllerID)
	}
	if len(controllers) == 0 {
		return nil
	}
	data, err := json.Marshal(map[string]any{"format_version": "1.8.0", "render_controllers": controllers})
	if err != nil {
		return err
	}
	p.Files[path.Join("render_controllers", playerName+".render_controllers.json")] = data
	return nil
}
