package assetlibrary

import (
	"strconv"
	"strings"
)

type textureVariantBinding struct {
	ObjectID   string
	Controller string
	Index      int
	Expression string
}

func resourceValue(values map[string]any, key string) any {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}
func resourceAlias(ref, kind string) string {
	prefix := kind + "."
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(ref)), prefix) {
		return strings.TrimSpace(ref)[len(prefix):]
	}
	return ""
}
func sameControllerIndex(a, b string) bool {
	return normalizeControllerIndex(a) == normalizeControllerIndex(b)
}
func objectDescription(x objectIndex, v Object) map[string]any {
	kind := "minecraft:client_entity"
	if v.Kind == "item-model" {
		kind = "minecraft:attachable"
	}
	return objectMap(objectMap(x.docs[v.Definition][kind])["description"])
}

// Bind through controller resource references, never filenames or display names.
// A texture used by several geometry slots stays on the current geometry when
// possible; an ambiguous cross-geometry association is not offered as a variant.
func (x objectIndex) bindObjectTextureVariants(objects []Object) {
	family := map[string]map[string][]Object{}
	for _, v := range objects {
		if v.Kind != "entity" && v.Kind != "item-model" {
			continue
		}
		key := v.Kind + "\x00" + v.Definition
		if family[key] == nil {
			family[key] = map[string][]Object{}
		}
		family[key][v.GeometryID] = append(family[key][v.GeometryID], v)
	}
	for n := range objects {
		v := &objects[n]
		if v.Kind != "entity" && v.Kind != "item-model" {
			continue
		}
		desc := objectDescription(x, *v)
		geometries := objectMap(desc["geometry"])
		candidates := map[string][]textureVariantBinding{}
		relevant := false
		geoID := func(ref string) string {
			return objectString(resourceValue(geometries, resourceAlias(ref, "geometry")))
		}
		refs, _ := desc["render_controllers"].([]any)
		for _, entry := range refs {
			names := []string{objectString(entry)}
			if names[0] == "" {
				names = sortedKeys(objectMap(entry))
			}
			for _, name := range names {
				controller := x.controllers[strings.ToLower(name)]
				if controller == nil {
					continue
				}
				geometryRef := objectString(controller["geometry"])
				addConditional := func(ref, targetGeo string, index int, expression string) {
					refs := []any{ref}
					if array, selected, ok := controllerArray(ref); ok {
						refs, _ = resourceValue(objectMap(objectMap(controller["arrays"])["textures"]), array).([]any)
						if position, err := strconv.Atoi(selected); err == nil {
							if position >= 0 && position < len(refs) {
								refs = refs[position : position+1]
							} else {
								refs = nil
							}
						}
					}
					for _, ref := range refs {
						alias := resourceAlias(objectString(ref), "texture")
						for key, path := range v.TextureOptions {
							if path == "" || !strings.EqualFold(key, alias) {
								continue
							}
							for _, target := range family[v.Kind+"\x00"+v.Definition][targetGeo] {
								candidates[key] = append(candidates[key], textureVariantBinding{target.ID, name, index, expression})
							}
						}
					}
				}
				if condition, yes, no, ok := controllerConditional(geometryRef); ok {
					if geoID(yes) != v.GeometryID && geoID(no) != v.GeometryID {
						continue
					}
					relevant = true
					textures, _ := controller["textures"].([]any)
					if len(textures) > 0 {
						for branch, ref := range []string{no, yes} {
							texture := controllerResource(controller, "textures", objectString(textures[0]), branch, condition)
							addConditional(texture, geoID(ref), branch, condition)
						}
					}
					continue
				}
				gArray, gExpr, gIsArray := controllerArray(geometryRef)
				geometryValues, _ := resourceValue(objectMap(objectMap(controller["arrays"])["geometries"]), gArray).([]any)
				currentIndex := -1
				if gIsArray {
					for i, g := range geometryValues {
						if geoID(objectString(g)) == v.GeometryID {
							currentIndex = i
							break
						}
					}
					if currentIndex < 0 {
						continue
					}
				} else if geoID(geometryRef) != v.GeometryID {
					continue
				}
				relevant = true
				textures, _ := controller["textures"].([]any)
				if len(textures) == 0 {
					continue
				}
				textureRef := objectString(textures[0])
				if condition, yes, no, ok := controllerConditional(textureRef); ok && !gIsArray {
					addConditional(no, v.GeometryID, 0, condition)
					addConditional(yes, v.GeometryID, 1, condition)
					continue
				}
				tArray, tExpr, tIsArray := controllerArray(textureRef)
				values := []any{textureRef}
				if tIsArray {
					values, _ = resourceValue(objectMap(objectMap(controller["arrays"])["textures"]), tArray).([]any)
				}
				// Independent runtime expressions cannot establish a geometry/texture pair.
				if gIsArray && tIsArray && !sameControllerIndex(gExpr, tExpr) {
					continue
				}
				for i, value := range values {
					key := ""
					alias := resourceAlias(objectString(value), "texture")
					for option := range v.TextureOptions {
						if strings.EqualFold(option, alias) {
							key = option
							break
						}
					}
					if key == "" || v.TextureOptions[key] == "" {
						continue
					}
					targetGeo := v.GeometryID
					index := i
					expr := tExpr
					if gIsArray {
						if tIsArray {
							if i >= len(geometryValues) {
								continue
							}
							targetGeo = geoID(objectString(geometryValues[i]))
						} else {
							index = currentIndex
							expr = gExpr
						}
					}
					for _, target := range family[v.Kind+"\x00"+v.Definition][targetGeo] {
						candidates[key] = append(candidates[key], textureVariantBinding{target.ID, name, index, expr})
					}
				}
			}
		}
		if !relevant && len(geometries) == 1 {
			continue
		} // ordinary single-model texture overrides
		options := map[string]string{}
		links := map[string]string{}
		bindings := map[string]textureVariantBinding{}
		for key, all := range candidates {
			chosen := all[0]
			ambiguous := false
			for _, b := range all {
				if b.ObjectID == v.ID {
					chosen = b
					ambiguous = false
					break
				}
				if b.ObjectID != chosen.ObjectID {
					ambiguous = true
				}
			}
			if ambiguous {
				continue
			}
			options[key] = v.TextureOptions[key]
			links[key] = chosen.ObjectID
			bindings[key] = chosen
		}
		if len(options) == 0 {
			for key, path := range v.TextureOptions {
				if path != "" && path == v.Textures["*"] {
					options[key] = path
				}
			}
		}
		v.TextureOptions = options
		v.TextureVariantObjects = links
		v.textureVariants = bindings
	}
}

func (x objectIndex) variantMaterials(v Object, binding textureVariantBinding) Object {
	v.PreviewQueries = mergePreviewQueries(v.PreviewQueries, controllerSelectionQueries(binding.Expression, binding.Index))
	controller := x.controllers[strings.ToLower(binding.Controller)]
	desc := objectDescription(x, v)
	materials := objectMap(desc["materials"])
	v.BoneMaterials = append([]BoneMaterial{}, v.BoneMaterials...)
	entries, _ := controller["materials"].([]any)
	for _, entry := range entries {
		for _, pattern := range sortedKeys(objectMap(entry)) {
			ref := objectString(objectMap(entry)[pattern])
			alias := resourceAlias(controllerResource(controller, "materials", ref, binding.Index, binding.Expression), "material")
			mode := x.materialMode(objectString(resourceValue(materials, alias)))
			if mode == "" {
				continue
			}
			if pattern == "*" {
				v.Material = mode
			} else {
				v.BoneMaterials = append(v.BoneMaterials, BoneMaterial{pattern, mode})
			}
		}
	}
	return v
}
