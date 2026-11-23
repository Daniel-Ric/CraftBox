package assetlibrary

import (
	"io/fs"
	"maps"
	"strings"
	"sync"

	"github.com/Daniel-Ric/CraftBox/assets"
)

const playerArmorModel = "models/entity/player_armor.json"

func vanillaEquipmentDocument(name string) map[string]any {
	source, err := assets.Vanilla()
	if err != nil {
		return nil
	}
	raw, err := fs.ReadFile(source, name)
	if err != nil {
		return nil
	}
	var document map[string]any
	if normalizedJSON(raw, &document) != nil {
		return nil
	}
	return document
}

var vanillaArmorModels = sync.OnceValue(func() map[string]any {
	return vanillaEquipmentDocument(playerArmorModel)
})

func armorGeometry(id string) any {
	definitions := vanillaArmorModels()
	bones := map[string]map[string]any{}
	names := []string{}
	seen := map[string]bool{}
	var width, height any
	var merge func(string) bool
	merge = func(id string) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		for key, raw := range definitions {
			name, parent, _ := strings.Cut(key, ":")
			if name != id {
				continue
			}
			if parent != "" && !merge(parent) {
				return false
			}
			definition := objectMap(raw)
			if value := definition["texturewidth"]; value != nil {
				width = value
			}
			if value := definition["textureheight"]; value != nil {
				height = value
			}
			local, _ := definition["bones"].([]any)
			for _, rawBone := range local {
				bone := objectMap(rawBone)
				name := objectString(bone["name"])
				if bones[name] == nil {
					names = append(names, name)
					bones[name] = map[string]any{}
				}
				if bone["reset"] == true {
					delete(bones[name], "cubes")
				}
				maps.Copy(bones[name], bone)
			}
			return true
		}
		return false
	}
	if !merge(id) || width == nil || height == nil || len(names) == 0 {
		return nil
	}
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, bones[name])
	}
	return map[string]any{"minecraft:geometry": []any{map[string]any{
		"description": map[string]any{"identifier": id, "texture_width": width, "texture_height": height},
		"bones":       out,
	}}}
}

func vanillaArmorPreview(identifier string) (ObjectPreview, bool) {
	item, ok := strings.CutPrefix(identifier, "minecraft:")
	if !ok || item == "" || strings.ContainsAny(item, `/\`) {
		return ObjectPreview{}, false
	}
	path := "attachables/" + item + ".player.json"
	document := vanillaEquipmentDocument(path)
	desc := objectMap(objectMap(document["minecraft:attachable"])["description"])
	if objectString(desc["identifier"]) != identifier+".player" {
		return ObjectPreview{}, false
	}
	id := objectString(objectMap(desc["geometry"])["default"])
	if !strings.HasPrefix(id, "geometry.player.armor.") {
		return ObjectPreview{}, false
	}
	geometry := armorGeometry(id)
	texture := vanillaTexture(objectString(objectMap(desc["textures"])["default"]))
	if geometry == nil || texture == "" {
		return ObjectPreview{}, false
	}
	preview, err := (objectIndex{}).readTexture("", texture)
	if err != nil {
		return ObjectPreview{}, false
	}
	v := Object{
		ID: "vanilla:" + identifier, Name: identifier, Kind: "item-model",
		Definition: path, GeometryID: id, GeometryFile: playerArmorModel,
		Material: objectString(objectMap(desc["materials"])["default"]),
		Textures: map[string]string{"*": texture},
	}
	return ObjectPreview{
		Object: v, Geometry: geometry, GeometryID: id, Image: preview.Data,
		Textures: map[string]string{"*": preview.Data},
	}, true
}
