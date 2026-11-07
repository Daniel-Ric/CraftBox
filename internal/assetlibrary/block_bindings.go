package assetlibrary

import (
	"fmt"
	"path"
	"strings"
)

func blockModelStem(file string) string {
	stem := strings.TrimSuffix(strings.ToLower(path.Base(file)), ".json")
	stem = strings.TrimSuffix(stem, ".geometry")
	return strings.TrimSuffix(stem, ".geo")
}
func (x objectIndex) namedBlockGeometry(name string) (string, bool) {
	namespace, local, ok := strings.Cut(strings.ToLower(name), ":")
	if !ok || namespace == "minecraft" {
		return "", false
	}
	matches := []string{}
	for _, id := range sortedKeys(x.geometry) {
		file := strings.ToLower(x.paths[id])
		key := strings.ToLower(id)
		suffix := strings.ReplaceAll(local, "_", ".")
		exactID := key == "geometry."+namespace+".block."+suffix || key == "geometry."+namespace+".blocks."+suffix || key == "geometry."+namespace+"."+suffix
		exactFile := path.Dir(file) == "models/blocks/"+namespace && blockModelStem(file) == local
		if exactID || exactFile {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], false
	}
	return "", len(matches) > 1
}
func (x objectIndex) namedBlockTextures(name string) []string {
	if refs := x.terrain[name]; len(refs) > 0 {
		return refs
	}
	namespace, local, ok := strings.Cut(name, ":")
	if !ok || namespace == "minecraft" {
		return nil
	}
	if file := x.texture("textures/blocks/" + namespace + "/" + local); file != "" {
		return []string{file}
	}
	return nil
}
func fullBlockGeometry(id string) bool {
	return id == "minecraft:geometry.full_block" || id == "geometry.full_block"
}
func (x objectIndex) resourceBlockMaterials(name string, value any) (map[string]any, map[string]string) {
	materials := map[string]any{}
	options := map[string]string{}
	if ref := objectString(value); ref != "" {
		materials["*"] = map[string]any{"texture": ref, "render_method": "alpha_test"}
	} else {
		for face, ref := range objectMap(value) {
			materials[face] = map[string]any{"texture": ref, "render_method": "alpha_test"}
		}
	}
	if value == nil {
		for _, ref := range x.namedBlockTextures(name) {
			if texture := x.texture(ref); texture != "" {
				if len(materials) == 0 {
					materials["*"] = map[string]any{"texture": ref, "render_method": "alpha_test"}
				}
				options[ref] = texture
			}
		}
	}
	return materials, options
}

func (x objectIndex) blockObject(name, file string, components map[string]any) Object {
	merged := map[string]any{}
	for key, value := range components {
		merged[key] = value
	}
	for _, key := range []string{"geometry", "material_instances"} {
		if merged["minecraft:"+key] == nil {
			merged["minecraft:"+key] = objectMap(merged["minecraft:embedded_visual"])[key]
		}
	}
	components = merged
	resourceOptions := map[string]string{}
	inferredTexture := false
	if components["minecraft:material_instances"] == nil {
		resourceName := strings.SplitN(name, " / variant ", 2)[0]
		value := objectMap(x.docs["blocks.json"][resourceName])["textures"]
		fallback, options := x.resourceBlockMaterials(resourceName, value)
		if len(fallback) > 0 {
			components["minecraft:material_instances"] = fallback
			resourceOptions = options
			inferredTexture = value == nil
		}
	}
	geo := objectString(components["minecraft:geometry"])
	geometryComponent := objectMap(components["minecraft:geometry"])
	if geo == "" {
		geo = objectString(geometryComponent["identifier"])
	}
	v := Object{Name: name, Kind: "block", Definition: file, GeometryID: geo, GeometryFile: x.paths[geo], Textures: map[string]string{}, Material: "alpha_test", MaterialModes: map[string]string{}, Transformation: objectMap(components["minecraft:transformation"])}
	v.TextureOptions = resourceOptions
	if inferredTexture {
		v.Warnings = append(v.Warnings, "Block texture inferred from an exact resource name; its material component is not supplied")
	}
	if len(v.Transformation) == 0 && components["minecraft:rotation"] != nil {
		v.Transformation = map[string]any{"rotation": components["minecraft:rotation"]}
	}
	if geo != "" && x.geometry[geo] == nil && !fullBlockGeometry(geo) {
		v.Warnings = append(v.Warnings, "Custom block geometry is not defined in this pack: "+geo)
	}
	for _, bone := range sortedKeys(objectMap(geometryComponent["bone_visibility"])) {
		if visible, ok := previewVisibility(objectMap(geometryComponent["bone_visibility"])[bone]); ok {
			v.Visibility = append(v.Visibility, BoneVisibility{bone, visible})
		} else {
			v.Warnings = append(v.Warnings, "Runtime block bone visibility is not available for "+bone)
		}
	}
	materials := objectMap(components["minecraft:material_instances"])
	if len(materials) == 0 {
		v.Warnings = append(v.Warnings, "Block materials are not defined in this pack")
	}
	mappings := objectMap(materials["mappings"])
	if len(objectMap(materials["materials"])) > 0 {
		materials = objectMap(materials["materials"])
	}
	combined := map[string]any{}
	for key, value := range materials {
		combined[key] = value
	}
	for key, value := range mappings {
		combined[key] = value
	}
	materials = combined
	for _, alias := range sortedKeys(materials) {
		var resolved any = materials[alias]
		visited := map[string]bool{alias: true}
		for step := 0; step < 20; step++ {
			ref, ok := resolved.(string)
			if !ok {
				break
			}
			if visited[ref] {
				resolved = nil
				break
			}
			visited[ref] = true
			resolved = materials[ref]
		}
		material := objectMap(resolved)
		ref := objectString(material["texture"])
		if refs := x.terrain[ref]; len(refs) > 0 {
			ref = refs[0]
		}
		v.Textures[alias] = x.texture(ref)
		mode := objectString(material["render_method"])
		if mode == "" {
			mode = "opaque"
		}
		v.MaterialModes[alias] = mode
		if alias == "*" || len(materials) == 1 {
			v.Material = mode
		}
	}
	if geo == "" || fullBlockGeometry(geo) {
		v.Geometry = unitBlockGeometry(v.Textures)
	}
	return v
}
func (x objectIndex) legacyBlockObject(name string, def map[string]any) (Object, bool) {
	value := def["textures"]
	materials, options := x.resourceBlockMaterials(name, value)
	geo, ambiguous := x.namedBlockGeometry(name)
	if len(materials) == 0 && geo == "" {
		return Object{}, false
	}
	components := map[string]any{"minecraft:material_instances": materials}
	if geo != "" {
		components["minecraft:geometry"] = geo
	}
	v := x.blockObject(name, "blocks.json", components)
	v.TextureOptions = options
	if geo != "" || value == nil {
		v.Warnings = append(v.Warnings, "Preview binding inferred from exact resource names in this pack; behavior-pack block components are not supplied")
	}
	if ambiguous {
		v.Warnings = append(v.Warnings, "Multiple exact geometry matches; no custom shape was inferred")
	}
	if geo == "" && strings.Contains(name, ":") && !strings.HasPrefix(name, "minecraft:") {
		v.Warnings = append(v.Warnings, "Block shape components are unavailable; a texture cube is shown")
	}
	return v, true
}
func (x objectIndex) unassignedGeometry(id string) Object {
	file := x.paths[id]
	v := Object{Name: id, Kind: "geometry", Definition: file, GeometryID: id, GeometryFile: file, Warnings: []string{"No entity or block binding in this pack. Select a texture from Files to inspect an unassigned model"}}
	if ref, known := x.verifiedGeometryTexture(id, file); known {
		if texture := x.texture(ref); texture != "" {
			v.Textures = map[string]string{"*": texture}
			v.Material = "alpha_test"
			v.Warnings = []string{"Preview uses a verified texture binding for this resource pack; no entity binding is supplied"}
		} else {
			message := "The texture for this geometry is not supplied in this resource pack"
			if ref != "" {
				message += ": " + ref
			}
			v.Warnings = []string{message}
		}
		return v
	}
	if strings.HasPrefix(file, "models/blocks/") {
		texture := x.texture(path.Join("textures", strings.TrimPrefix(path.Dir(file), "models/"), blockModelStem(file)))
		if texture != "" {
			v.Textures = map[string]string{"*": texture}
			v.Material = "alpha_test"
			v.Warnings = []string{fmt.Sprintf("Texture preview inferred from the exact model path (%s); behavior-pack block components are not supplied", texture)}
		}
		if len(v.Textures) == 0 {
			stem := blockModelStem(file)
			// A terrain atlas can use textures/block (singular) and an entirely
			// different directory layout from models/blocks. Use exact atlas names,
			// including named variants, only when the model stem is unambiguous.
			owners := 0
			for _, modelFile := range x.paths {
				if strings.HasPrefix(modelFile, "models/blocks/") && blockModelStem(modelFile) == stem {
					owners++
				}
			}
			if owners == 1 {
				// Segmented block models can share one atlas entry. Require all
				// three authored siblings in the same directory before using it.
				for _, suffix := range []string{"_bottom", "_middle", "_top"} {
					base := strings.TrimSuffix(stem, suffix)
					if base == stem || len(x.terrain[base]) == 0 {
						continue
					}
					siblings := map[string]bool{}
					for _, modelFile := range x.paths {
						if path.Dir(modelFile) == path.Dir(file) {
							siblings[blockModelStem(modelFile)] = true
						}
					}
					if siblings[base+"_bottom"] && siblings[base+"_middle"] && siblings[base+"_top"] {
						stem = base
					}
					break
				}
				options := map[string]string{}
				for _, key := range sortedKeys(x.terrain) {
					if key != stem && !strings.HasPrefix(key, stem+"_") {
						continue
					}
					// Do not let a shorter model name steal another model's textures.
					owned := true
					for _, modelFile := range x.paths {
						other := blockModelStem(modelFile)
						if strings.HasPrefix(modelFile, "models/blocks/") && len(other) > len(stem) && (key == other || strings.HasPrefix(key, other+"_")) {
							owned = false
							break
						}
					}
					if !owned {
						continue
					}
					for i, ref := range x.terrain[key] {
						if texture := x.texture(ref); texture != "" {
							name := key
							if i > 0 {
								name = fmt.Sprintf("%s / %d", key, i+1)
							}
							options[name] = texture
						}
					}
				}
				if len(options) > 0 {
					v.TextureOptions = options
					v.Textures = map[string]string{"*": options[sortedKeys(options)[0]]}
					v.Material = "alpha_test"
					v.Warnings = []string{"Texture preview inferred from matching terrain atlas names; behavior-pack block components are not supplied"}
				}
			}
		}
	}
	return v
}
