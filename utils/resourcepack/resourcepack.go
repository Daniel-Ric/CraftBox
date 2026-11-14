package resourcepack

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/skinconverter"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

type Pack struct {
	Manifest resource.Manifest
	Files    map[string][]byte
}

func parseVersion(ver string) [3]int {
	frag := strings.Split(ver, ".")
	if len(frag) != 3 {
		panic("invalid version number " + ver)
	}
	a, _ := strconv.ParseInt(frag[0], 10, 64)
	b, _ := strconv.ParseInt(frag[1], 10, 64)
	c, _ := strconv.ParseInt(frag[2], 10, 64)
	return [3]int{int(a), int(b), int(c)}
}

func buildManifest(headerUUID, moduleUUID uuid.UUID) resource.Manifest {
	return resource.Manifest{
		FormatVersion: 2,
		Header: resource.Header{
			Name:               "auto-generated resource pack",
			Description:        "This resource pack contains auto-generated content from dragonfly",
			UUID:               headerUUID,
			Version:            [3]int{0, 0, 1},
			MinimumGameVersion: parseVersion(protocol.CurrentVersion),
		},
		Modules: []resource.Module{
			{
				UUID:        moduleUUID.String(),
				Description: "This resource pack contains auto-generated content from dragonfly",
				Type:        "resources",
				Version:     [3]int{0, 0, 1},
			},
		},
	}
}

type ClientEntityFile struct {
	FormatVersion string        `json:"format_version"`
	ClientEntity  *ClientEntity `json:"minecraft:client_entity"`
}

type ClientEntity struct {
	Description *ClientEntityDescription `json:"description"`
}

type ClientEntityDescription struct {
	Identifier        string            `json:"identifier"`
	Materials         map[string]string `json:"materials"`
	Geometry          map[string]string `json:"geometry"`
	Textures          map[string]string `json:"textures"`
	Scripts           map[string]any    `json:"scripts,omitempty"`
	Animations        map[string]string `json:"animations,omitempty"`
	RenderControllers []string          `json:"render_controllers"`
	EnableAttachables bool              `json:"enable_attachables,omitempty"`
}

func New() *Pack {
	p := Pack{
		Manifest: buildManifest(uuid.New(), uuid.New()),
		Files:    make(map[string][]byte),
	}
	return &p
}

type EntityPlayer struct {
	Identifier string
	UUID       uuid.UUID
}

func ValidPlayerSkin(skin *protocol.Skin) bool {
	return skin != nil && skin.SkinImageWidth > 0 && skin.SkinImageHeight > 0 && skin.SkinImageWidth <= 4096 && skin.SkinImageHeight <= 4096 && uint64(len(skin.SkinData)) >= uint64(skin.SkinImageWidth)*uint64(skin.SkinImageHeight)*4
}

func (p *Pack) MakePlayers(players []EntityPlayer, playerSkins map[uuid.UUID]*protocol.Skin) error {
	for _, player := range players {
		if !ValidPlayerSkin(playerSkins[player.UUID]) {
			logrus.Warnf("no skin for player %s", player.UUID)
			continue
		}
		skin := skinconverter.Skin{Skin: playerSkins[player.UUID]}

		skinTexture := image.NewNRGBA(image.Rect(0, 0, int(skin.SkinImageWidth), int(skin.SkinImageHeight)))
		copy(skinTexture.Pix, skin.SkinData)

		var skinTextureName = path.Join("textures", "player", player.UUID.String())

		textureData := bytes.NewBuffer(nil)
		png.Encode(textureData, skinTexture)
		p.Files[skinTextureName+".png"] = textureData.Bytes()

		var capeTextureName string
		if w, h := int(skin.CapeImageWidth), int(skin.CapeImageHeight); skin.CapeID != "" && w > 0 && h > 0 && w <= 4096 && h <= 4096 && len(skin.CapeData) >= w*h*4 {
			capeTexture := image.NewNRGBA(image.Rect(0, 0, w, h))
			copy(capeTexture.Pix, skin.CapeData)
			capeTextureName = path.Join("textures", "player", "cape_"+player.UUID.String())
			capeData := bytes.NewBuffer(nil)
			png.Encode(capeData, capeTexture)
			p.Files[capeTextureName+".png"] = capeData.Bytes()
		}

		geometryIdentifier, _, geometry, err := skin.ParseGeometry()
		if err != nil {
			logrus.Warnf("player %s: %s, drawing it with the default model", player.UUID, err)
			geometry = nil
		}

		if geometry != nil {

			geometryIdentifier = "geometry.craftbox_player." + strings.ReplaceAll(player.UUID.String(), "-", "_")
		} else {

			if !strings.Contains(strings.ToLower(geometryIdentifier), "slim") {
				geometryIdentifier = "geometry.humanoid.custom"
			} else {
				geometryIdentifier = "geometry.humanoid.customSlim"
			}
		}

		if geometry != nil {
			geometry.Description.Identifier = geometryIdentifier
			p.Files[path.Join("models", "entity", "craftbox_player", geometryIdentifier+".geo.json")], _ = json.Marshal(skinconverter.SkinGeometryFile{

				FormatVersion: "1.12.0",
				Geometry:      []skinconverter.SkinGeometry{*geometry},
			})
		}

		var clientEntity = ClientEntityFile{
			FormatVersion: "1.10.0",
			ClientEntity: &ClientEntity{
				Description: &ClientEntityDescription{
					Identifier:        player.Identifier,
					Materials:         map[string]string{"default": "entity_alphatest"},
					Textures:          map[string]string{"default": skinTextureName},
					Geometry:          map[string]string{"default": geometryIdentifier},
					RenderControllers: []string{"controller.render.default"},

					EnableAttachables: true,
					Scripts:           playerScripts,
					Animations:        playerAnimations,
				},
			},
		}
		if capeTextureName != "" {
			d := clientEntity.ClientEntity.Description
			d.Materials["cape"] = "entity_alphatest"
			d.Textures["cape"] = capeTextureName
			d.Geometry["cape"] = capeGeometryID
			d.RenderControllers = append(d.RenderControllers, capeRenderController)
			d.Scripts, d.Animations = withCape(d.Scripts, d.Animations)
			p.addCapeFiles()
		}
		clientEntityData, err := json.Marshal(&clientEntity)
		if err != nil {
			return err
		}
		p.Files[path.Join("entity", "players", player.UUID.String()+".json")] = clientEntityData
	}

	return nil
}

func (p *Pack) WriteToFS(fs utils.WriterFS) error {
	f, err := fs.Create("manifest.json")
	if err != nil {
		return err
	}
	_ = json.NewEncoder(f).Encode(&p.Manifest)
	f.Close()

	for name, content := range p.Files {
		f, err := fs.Create(name)
		if err != nil {
			return err
		}
		f.Write(content)
		f.Close()
	}
	return nil
}

var playerAnimations = map[string]string{
	"humanoid_base_pose":     "animation.humanoid.base_pose",
	"look_at_target_default": "animation.humanoid.look_at_target.default",
	"move.arms":              "animation.player.move.arms",
	"move.legs":              "animation.player.move.legs",
	"riding.arms":            "animation.player.riding.arms",
	"riding.legs":            "animation.player.riding.legs",
	"riding.root":            "animation.player.riding.root",
	"holding":                "animation.player.holding",
	"sneaking":               "animation.player.sneaking",
	"bob":                    "animation.player.bob",
	"swimming":               "animation.player.swim",
	"swimming.legs":          "animation.player.swim.legs",
}

var playerScripts = map[string]any{

	"scale": "0.9375",
	"initialize": []string{
		"variable.swim_amount = 0.0;",
		"variable.riding_y_offset = 0.0;",
		"variable.is_first_person = 0.0;",
		"variable.is_paperdoll = 0.0;",
		"variable.map_face_icon = 0.0;",
		"variable.bob_animation = 0.0;",
		"variable.is_holding_right = 0.0;",
		"variable.is_holding_left = 0.0;",
		"variable.is_brandishing_spear = 0.0;",
		"variable.is_holding_spyglass = 0.0;",
		"variable.is_using_vr = 0.0;",
		"variable.is_blinking = 0.0;",
		"variable.last_blink_time = 0.0;",
		"variable.hand_bob = 0.0;",
		"variable.attack_time = 0.0;",
		"variable.attack_body_rot_y = 0.0;",
		"variable.player_x_rotation = 0.0;",
		"variable.gliding_speed_value = 1.0;",
		"variable.short_arm_offset_right = 0.0;",
		"variable.short_arm_offset_left = 0.0;",
		"variable.helmet_layer_visible = 1.0;",
		"variable.leg_layer_visible = 1.0;",
		"variable.boot_layer_visible = 1.0;",
		"variable.chest_layer_visible = 1.0;",
		"variable.tcos0 = 0.0;",
	},
	"pre_animation": []string{
		"variable.tcos0 = (math.cos(query.modified_distance_moved * 38.17) * query.modified_move_speed / variable.gliding_speed_value) * 57.3;",

		"variable.riding_y_offset = query.is_riding_any_entity_of_type('minecraft:minecart', 'minecraft:boat', 'minecraft:chest_boat', 'minecraft:strider') ? -3.0 : 0.0;",
	},
	"animate": []any{
		"humanoid_base_pose",
		"look_at_target_default",
		map[string]string{"move.arms": "!query.is_riding && !query.is_swimming"},
		map[string]string{"move.legs": "!query.is_riding && !query.is_swimming"},
		map[string]string{"riding.arms": "query.is_riding"},
		map[string]string{"riding.legs": "query.is_riding"},
		map[string]string{"riding.root": "query.is_riding"},
		map[string]string{"bob": "!query.is_riding"},
		"holding",
		map[string]string{"sneaking": "query.is_sneaking && !query.is_swimming"},
		map[string]string{"swimming": "query.is_swimming"},
		map[string]string{"swimming.legs": "query.is_swimming"},
	},
}

const (
	capeGeometryID       = "geometry.craftbox_player.cape"
	capeRenderController = "controller.render.craftbox_player.cape"
)

var capeGeometry = map[string]any{
	"format_version": "1.12.0",
	"minecraft:geometry": []any{map[string]any{
		"description": map[string]any{
			"identifier":     capeGeometryID,
			"texture_width":  64,
			"texture_height": 32,
		},

		"bones": []any{
			map[string]any{"name": "root", "pivot": []float32{0, 0, 0}},
			map[string]any{"name": "waist", "parent": "root", "pivot": []float32{0, 12, 0}},
			map[string]any{"name": "body", "parent": "waist", "pivot": []float32{0, 24, 0}},
			map[string]any{
				"name": "cape", "parent": "body",
				"pivot":              []float32{0, 24, 3},
				"bind_pose_rotation": []float32{0, 180, 0},
				"rotation":           []float32{0, 180, 0},
				"cubes": []any{map[string]any{
					"origin": []float32{-5, 8, 3},
					"size":   []float32{10, 16, 1},
					"uv":     []float32{0, 0},
				}},
			},
		},
	}},
}

var capeRenderControllers = map[string]any{
	"format_version": "1.8.0",
	"render_controllers": map[string]any{
		capeRenderController: map[string]any{
			"geometry":  "Geometry.cape",
			"materials": []any{map[string]string{"*": "Material.cape"}},
			"textures":  []string{"Texture.cape"},
		},
	},
}

const capeAnimation = "animation.craftbox_player.cape"

var capeAnimations = map[string]any{
	"format_version": "1.8.0",
	"animations": map[string]any{
		capeAnimation: map[string]any{
			"loop":  true,
			"bones": map[string]any{"cape": map[string]any{"rotation": []float32{-6, 0, 0}}},
		},
	},
}

func (p *Pack) addCapeFiles() {
	const model = "models/entity/craftbox_player/cape.geo.json"
	if _, ok := p.Files[model]; ok {
		return
	}
	p.Files[model], _ = json.Marshal(capeGeometry)
	p.Files["render_controllers/craftbox_player.render_controllers.json"], _ = json.Marshal(capeRenderControllers)
	p.Files["animations/craftbox_player.animation.json"], _ = json.Marshal(capeAnimations)
}

func withCape(scripts map[string]any, animations map[string]string) (map[string]any, map[string]string) {
	s := maps.Clone(scripts)
	s["animate"] = append(slices.Clone(scripts["animate"].([]any)), "cape")
	a := maps.Clone(animations)
	a["cape"] = capeAnimation
	return s, a
}
