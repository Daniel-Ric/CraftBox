package assetlibrary

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Object struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Kind                  string            `json:"kind"`
	Definition            string            `json:"definition"`
	GeometryID            string            `json:"geometryId"`
	GeometryFile          string            `json:"geometryFile"`
	Textures              map[string]string `json:"textures"`
	TextureOptions        map[string]string `json:"textureOptions"`
	TextureVariantObjects map[string]string `json:"textureVariantObjects,omitempty"`
	textureVariants       map[string]textureVariantBinding
	Material              string             `json:"material"`
	MaterialModes         map[string]string  `json:"materialModes,omitempty"`
	Transformation        map[string]any     `json:"transformation,omitempty"`
	Animations            map[string]string  `json:"animations"`
	Warnings              []string           `json:"warnings"`
	Geometry              any                `json:"-"`
	Layers                []string           `json:"layers"`
	BoneMaterials         []BoneMaterial     `json:"boneMaterials"`
	Visibility            []BoneVisibility   `json:"visibility"`
	DefaultAnimations     map[string]string  `json:"-"`
	PreviewQueries        map[string]float64 `json:"-"`
	Condition             string             `json:"condition,omitempty"`
	Permutation           int                `json:"permutation,omitempty"`
}
type ObjectPreview struct {
	Object           Object             `json:"object"`
	Geometry         any                `json:"geometry"`
	GeometryID       string             `json:"geometryId"`
	Textures         map[string]string  `json:"textures"`
	Image            string             `json:"image"`
	Warnings         []string           `json:"warnings"`
	InitialPose      []InitialAnimation `json:"initialPose"`
	Motion           *MotionRuntime     `json:"motion,omitempty"`
	TextureAnimation *TextureAnimation  `json:"textureAnimation,omitempty"`
	CapturedSlot     string             `json:"capturedSlot,omitempty"`
}
type InitialAnimation struct {
	Name      string `json:"name"`
	Animation any    `json:"animation"`
}
type objectIndex struct {
	blockWarnings []string
	readError     error
	resolved      []Object
	epoch         string
	sourceBytes   int
	geometry      map[string]any
	paths         map[string]string
	controllers   map[string]map[string]any
	terrain       map[string][]string
	itemTextures  map[string][]string
	animations    map[string]string
	docs          map[string]map[string]any
	names         map[string]bool
}

func objectMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
func objectString(v any) string { s, _ := v.(string); return s }
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func objectTextureRefs(v any) []string {
	if s, ok := v.(string); ok {
		return []string{s}
	}
	out := []string{}
	if all, ok := v.([]any); ok {
		for _, v := range all {
			out = append(out, objectTextureRefs(v)...)
		}
		return out
	}
	if m, ok := v.(map[string]any); ok {
		if p := objectString(m["path"]); p != "" {
			return []string{p}
		}
		if value, ok := m["textures"]; ok {
			return objectTextureRefs(value)
		}
	}
	return out
}
func (x objectIndex) texture(ref string) string {
	if ref == "" {
		return ""
	}
	for _, candidate := range []string{ref, ref + ".png", ref + ".tga", ref + ".jpg", ref + ".jpeg"} {
		if x.names[candidate] {
			return candidate
		}
	}
	return ""
}
func (x objectIndex) definitionTexture(file, ref string) string {
	parts := strings.Split(file, "/")
	if len(parts) > 2 && parts[0] == "subpacks" {
		if resolved := x.texture(strings.Join(parts[:2], "/") + "/" + ref); resolved != "" {
			return resolved
		}
	}
	return x.texture(ref)
}
func (l *Library) objectIndex(d Detail) objectIndex {
	x := objectIndex{geometry: map[string]any{}, paths: map[string]string{}, controllers: map[string]map[string]any{}, terrain: map[string][]string{}, itemTextures: map[string][]string{}, animations: map[string]string{}, docs: map[string]map[string]any{}, names: map[string]bool{}}
	root, _ := l.RootFor(d.Item.ID)
	for _, f := range d.Files {
		x.names[f.Path] = true
		if !strings.HasSuffix(strings.ToLower(f.Path), ".json") || f.Bytes > 4<<20 {
			continue
		}
		b, e := readLimit(filepath.Join(root, filepath.FromSlash(f.Path)), 4<<20)
		if e != nil {
			if x.readError == nil {
				x.readError = fmt.Errorf("Pack definition %s could not be read: %w", f.Path, e)
			}
			continue
		}
		if f.Path == "textures/flipbook_textures.json" {
			var entries []any
			if normalizedJSON(b, &entries) == nil {
				x.docs[f.Path] = map[string]any{"entries": entries}
				x.sourceBytes += len(b)
			}
			continue
		}
		var m map[string]any
		if normalizedJSON(b, &m) != nil {
			continue
		}
		x.sourceBytes += len(b)
		x.docs[f.Path] = m
		if geometries, ok := m["minecraft:geometry"].([]any); ok {
			for _, g := range geometries {
				gm := objectMap(g)
				id := objectString(objectMap(gm["description"])["identifier"])
				if id != "" {
					x.geometry[id] = map[string]any{"format_version": m["format_version"], "minecraft:geometry": []any{g}}
					x.paths[id] = f.Path
				}
			}
		}
		for key, value := range m {
			if strings.HasPrefix(key, "geometry.") {
				x.geometry[key] = map[string]any{"format_version": "1.8.0", key: value}
				x.paths[key] = f.Path
			}
		}
		for key, value := range objectMap(m["render_controllers"]) {
			x.controllers[strings.ToLower(key)] = objectMap(value)
		}
		for key := range objectMap(m["animations"]) {
			x.animations[key] = f.Path
		}
		if f.Path == "textures/terrain_texture.json" || f.Path == "textures/item_texture.json" {
			table := x.terrain
			if f.Path == "textures/item_texture.json" {
				table = x.itemTextures
			}
			for name, value := range objectMap(m["texture_data"]) {
				table[name] = objectTextureRefs(value)
			}
		}
	}
	return x
}
func (l *Library) Objects(id string) ([]Object, error) {
	x, e := l.catalog(id)
	if e != nil {
		return nil, e
	}
	return x.objects(), nil
}
func (x objectIndex) objects() []Object {
	if x.resolved != nil {
		return x.resolved
	}
	out := []Object{}
	used := map[string]bool{}
	add := func(v Object) {
		if v.Kind == "block" {
			v.Warnings = append(v.Warnings, x.blockWarnings...)
		}
		if len(out) >= 6000 {
			return
		}
		sum := sha256.Sum256([]byte(v.Kind + "\x00" + v.Name + "\x00" + v.Definition + "\x00" + v.GeometryID))
		v.ID = hex.EncodeToString(sum[:12])
		if v.Textures == nil {
			v.Textures = map[string]string{}
		}
		if v.TextureOptions == nil {
			v.TextureOptions = map[string]string{}
		}
		if v.Animations == nil {
			v.Animations = map[string]string{}
		}
		if v.Warnings == nil {
			v.Warnings = []string{}
		}
		out = append(out, v)
		if v.GeometryID != "" {
			used[v.GeometryID] = true
		}
	}
	for _, file := range sortedKeys(x.docs) {
		m := x.docs[file]
		for _, kind := range []string{"minecraft:client_entity", "minecraft:attachable"} {
			desc := objectMap(objectMap(m[kind])["description"])
			name := objectString(desc["identifier"])
			if name == "" {
				continue
			}
			geos := objectMap(desc["geometry"])
			tex := objectMap(desc["textures"])
			materials := objectMap(desc["materials"])
			options := map[string]string{}
			for _, alias := range sortedKeys(tex) {
				options[alias] = x.definitionTexture(file, objectString(tex[alias]))
			}
			anim := map[string]string{}
			for _, alias := range sortedKeys(objectMap(desc["animations"])) {
				ref := objectString(objectMap(desc["animations"])[alias])
				if file := x.animations[ref]; file != "" {
					anim[ref] = file
				}
			}
			for _, alias := range sortedKeys(geos) {
				geometryID := objectString(geos[alias])
				if geometryID == "" {
					continue
				}
				v := Object{Name: name, Kind: "entity", Definition: file, GeometryID: geometryID, GeometryFile: x.paths[geometryID], Textures: map[string]string{}, TextureOptions: options, Material: objectString(materials["default"]), Animations: anim}
				v.Layers = controllerLayers(desc, x.controllers)
				v.DefaultAnimations = map[string]string{}
				if list, ok := objectMap(desc["scripts"])["animate"].([]any); ok {
					for _, entry := range list {
						alias := objectString(entry)
						if alias == "idle" || strings.HasPrefix(alias, "idle_") || alias == "base_pose" || alias == "setup" {
							ref := objectString(objectMap(desc["animations"])[alias])
							if source := x.animations[ref]; source != "" {
								v.DefaultAnimations[ref] = source
							}
						}
					}
				}
				if len(geos) > 1 {
					v.Name += " / " + alias
				}
				if kind == "minecraft:attachable" {
					v.Kind = "item-model"
				}
				if x.geometry[geometryID] == nil {
					v.Warnings = append(v.Warnings, "Geometry is supplied by another pack or Minecraft: "+geometryID)
				}
				textureAlias := "default"
				if _, ok := tex[alias]; ok {
					textureAlias = alias
				}
				if _, ok := tex[textureAlias]; !ok && len(tex) > 0 {
					textureAlias = sortedKeys(tex)[0]
				}
				v.Textures["*"] = options[textureAlias]
				if controllers, ok := desc["render_controllers"].([]any); ok {
					for _, value := range controllers {
						controllerName := objectString(value)
						if controllerName == "" {
							for k := range objectMap(value) {
								controllerName = k
								break
							}
						}
						controller := x.controllers[strings.ToLower(controllerName)]
						if controller == nil && controllerName != "controller.render.default" {
							v.Warnings = append(v.Warnings, "Render controller is supplied by another pack: "+controllerName)
							continue
						}
						geometryRef := strings.ToLower(objectString(controller["geometry"]))
						if strings.HasPrefix(geometryRef, "geometry.") && strings.TrimPrefix(geometryRef, "geometry.") != strings.ToLower(alias) {
							continue
						}
						variantIndex, variantExpression := controllerGeometryIndex(controller, alias)
						if variantIndex == -2 {
							continue
						}
						v.PreviewQueries = mergePreviewQueries(v.PreviewQueries, controllerSelectionQueries(variantExpression, variantIndex))
						if arrays := objectMap(controller["arrays"]); len(arrays) > 0 {
							v.Warnings = append(v.Warnings, "This controller selects variants at runtime. Use the variant selector to inspect linked models and textures")
						}
						if refs, ok := controller["textures"].([]any); ok {
							if len(refs) > 1 {
								v.Warnings = append(v.Warnings, "Additional render-controller texture layers may depend on Minecraft material shaders")
							}
							for _, ref := range refs {
								r := controllerResource(controller, "textures", objectString(ref), variantIndex, variantExpression)
								if strings.HasPrefix(strings.ToLower(r), "texture.") {
									key := strings.TrimPrefix(strings.ToLower(r), "texture.")
									for textureKey, p := range options {
										if strings.EqualFold(textureKey, key) {
											v.Textures["*"] = p
											break
										}
									}
									break
								}
							}
						}
						if entries, ok := controller["materials"].([]any); ok {
							for _, entry := range entries {
								for bone, ref := range objectMap(entry) {
									materialAlias := strings.TrimPrefix(strings.ToLower(objectString(ref)), "material.")
									materialAlias = strings.TrimPrefix(strings.ToLower(controllerResource(controller, "materials", objectString(ref), variantIndex, variantExpression)), "material.")
									if material := objectString(materials[materialAlias]); material != "" {
										if bone == "*" {
											v.Material = material
										} else {
											v.BoneMaterials = append(v.BoneMaterials, BoneMaterial{bone, material})
										}
									}
								}
							}
						}
						if rules, ok := controller["part_visibility"].([]any); ok {
							for _, rule := range rules {
								for pattern, expression := range objectMap(rule) {
									if visible, known := previewVisibility(expression); known {
										v.Visibility = append(v.Visibility, BoneVisibility{pattern, visible})
									} else {
										v.Warnings = append(v.Warnings, "Runtime visibility is not available for bone "+pattern)
									}
								}
							}
						}
					}
				}
				add(v)
			}
		}
	}
	blockBindings := map[string]bool{}
	resolveBlock := func(name, file string, components map[string]any, condition ...string) {
		blockBindings[name] = true
		v := x.blockObject(name, file, components)
		if len(condition) > 0 {
			v.Condition = condition[0]
			if _, suffix, ok := strings.Cut(name, " / variant "); ok {
				v.Permutation, _ = strconv.Atoi(suffix)
			}
		}
		add(v)
	}
	for _, file := range sortedKeys(x.docs) {
		m := x.docs[file]
		if block := objectMap(m["minecraft:block"]); len(block) > 0 {
			name := objectString(objectMap(block["description"])["identifier"])
			components := objectMap(block["components"])
			resolveBlock(name, file, components)
			if permutations, ok := block["permutations"].([]any); ok {
				for i, p := range permutations {
					pm := objectMap(p)
					merged := map[string]any{}
					for k, v := range components {
						merged[k] = v
					}
					for k, v := range objectMap(pm["components"]) {
						merged[k] = v
					}
					resolveBlock(fmt.Sprintf("%s / variant %d", name, i+1), file, merged, objectString(pm["condition"]))
				}
			}
		}
	}
	if legacy := x.docs["blocks.json"]; legacy != nil {
		for _, name := range sortedKeys(legacy) {
			if blockBindings[name] {
				continue
			}
			if v, ok := x.legacyBlockObject(name, objectMap(legacy[name])); ok {
				add(v)
			}
		}
	}
	for _, id := range sortedKeys(x.geometry) {
		if used[id] {
			continue
		}
		add(x.unassignedGeometry(id))
	}
	for _, key := range sortedKeys(x.itemTextures) {
		for i, ref := range x.itemTextures[key] {
			name := key
			if i > 0 {
				name += fmt.Sprintf(" / %d", i+1)
			}
			add(Object{Name: name, Kind: "item", Definition: "textures/item_texture.json", Textures: map[string]string{"*": x.texture(ref)}})
		}
	}
	for _, file := range sortedKeys(x.names) {
		if fileKind(file) != "texture" || file == "pack_icon.png" {
			continue
		}
		kind := "texture"
		if strings.HasPrefix(file, "textures/ui/") || strings.HasPrefix(file, "textures/gui/") {
			kind = "ui"
		}
		add(Object{Name: file, Kind: kind, Definition: file, Textures: map[string]string{"*": file}})
	}
	sort.SliceStable(out, func(i, j int) bool {
		rank := map[string]int{"entity": 0, "item-model": 1, "block": 2, "geometry": 3, "item": 4, "texture": 5, "ui": 6}
		if rank[out[i].Kind] != rank[out[j].Kind] {
			return rank[out[i].Kind] < rank[out[j].Kind]
		}
		return out[i].Name < out[j].Name
	})
	x.bindObjectTextureVariants(out)
	return out
}
func unitBlockGeometry(textures map[string]string) any {
	faces := map[string]any{}
	for _, face := range []string{"east", "west", "up", "down", "south", "north"} {
		alias := face
		if textures[alias] == "" {
			if (face != "up" && face != "down") && textures["side"] != "" {
				alias = "side"
			} else if textures["all"] != "" {
				alias = "all"
			} else {
				alias = "*"
			}
		}
		faces[face] = map[string]any{"uv": []int{0, 0}, "uv_size": []int{16, 16}, "material_instance": alias}
	}
	return map[string]any{"minecraft:geometry": []any{map[string]any{"description": map[string]any{"identifier": "geometry.preview.block", "texture_width": 16, "texture_height": 16}, "bones": []any{map[string]any{"name": "block", "pivot": []int{0, 0, 0}, "cubes": []any{map[string]any{"origin": []int{-8, 0, -8}, "size": []int{16, 16, 16}, "uv": faces}}}}}}}
}
func (l *Library) ObjectPreview(id, objectID, variant string) (ObjectPreview, error) {
	return l.previewObject(id, objectID, variant, nil, nil)
}

func (l *Library) BlockStatePreview(id, objectID string, states map[string]any, matches func(string, map[string]any) (bool, bool)) (ObjectPreview, error) {
	return l.previewObject(id, objectID, "", states, matches)
}

func (l *Library) previewObject(id, objectID, variant string, states map[string]any, matches func(string, map[string]any) (bool, bool)) (ObjectPreview, error) {
	x, e := l.catalog(id)
	if e != nil {
		return ObjectPreview{}, e
	}
	cacheKey := id + "\x00" + x.epoch + "\x00" + objectID + "\x00" + variant
	if states != nil {
		raw, _ := json.Marshal(states)
		cacheKey += "\x00states:" + string(raw)
	}
	if l.previews != nil {
		if value, ok := l.previews.get(cacheKey); ok {
			return value, nil
		}
	}
	var selected *Object
	for _, v := range x.objects() {
		if v.ID == objectID {
			copy := v
			selected = &copy
			break
		}
	}
	if selected == nil {
		return ObjectPreview{}, errors.New("This object is no longer available")
	}
	v := *selected
	if states != nil && matches != nil && v.Kind == "block" {
		block := objectMap(x.docs[v.Definition]["minecraft:block"])
		if len(block) > 0 {
			components := map[string]any{}
			for key, value := range objectMap(block["components"]) {
				components[key] = value
			}
			if permutations, ok := block["permutations"].([]any); ok {
				for _, raw := range permutations {
					p := objectMap(raw)
					condition := objectString(p["condition"])
					enabled, supported := matches(condition, states)
					if !supported {
						return ObjectPreview{}, errors.New("Live block condition cannot be resolved: " + condition)
					}
					if enabled {
						for key, value := range objectMap(p["components"]) {
							components[key] = value
						}
					}
				}
			}
			resolved := x.blockObject(objectString(objectMap(block["description"])["identifier"]), v.Definition, components)
			resolved.ID = v.ID
			v = resolved
		}
	}
	if variant != "" {
		path, ok := v.TextureOptions[variant]
		if !ok {
			return ObjectPreview{}, errors.New("Texture variant is not defined for this object")
		}
		if binding, ok := v.textureVariants[variant]; ok {
			for _, target := range x.objects() {
				if target.ID == binding.ObjectID {
					v = target
					break
				}
			}
			v = x.variantMaterials(v, binding)
		}
		v.Textures = map[string]string{"*": path}
	}
	result := ObjectPreview{Object: v, Geometry: v.Geometry, GeometryID: v.GeometryID, Textures: map[string]string{}, Warnings: append([]string{}, v.Warnings...)}
	result.Motion, result.TextureAnimation = l.objectRuntime(id, x, v)
	// An explicit selection freezes the texture; runtime frame arrays must not
	// replace the user's selected variant on the next animation frame.
	if variant != "" {
		result.TextureAnimation = nil
	}
	for _, name := range sortedKeys(v.DefaultAnimations) {
		if len(result.InitialPose) >= 8 {
			break
		}
		if doc := x.docs[v.DefaultAnimations[name]]; doc != nil {
			result.InitialPose = append(result.InitialPose, InitialAnimation{name, doc})
		}
	}
	if v.GeometryID != "" {
		if geometry := x.geometry[v.GeometryID]; geometry != nil {
			result.Geometry = geometry
		}
	}
	var total int
	for _, alias := range sortedKeys(v.Textures) {
		file := v.Textures[alias]
		if file == "" {
			result.Warnings = append(result.Warnings, "Texture is not present for material "+alias)
			continue
		}
		root, _ := l.RootFor(id)
		preview, e := readAssetFile(root, file)
		if e != nil {
			result.Warnings = append(result.Warnings, e.Error())
			continue
		}
		if frames, sequence := blockTextureFrames(root, x, v, file, result.Geometry); len(frames) > 0 {
			preview.Data = frames[0]
			if alias == "*" && variant == "" && result.TextureAnimation == nil {
				result.TextureAnimation = sequence
			}
		}
		total += len(preview.Data)
		if total > 48<<20 {
			return result, errors.New("Combined model textures exceed the preview memory limit")
		}
		result.Textures[alias] = preview.Data
		if alias == "*" {
			result.Image = preview.Data
		}
	}
	if result.Image == "" {
		for _, alias := range []string{"default", "all", "side"} {
			if result.Textures[alias] != "" {
				result.Image = result.Textures[alias]
				break
			}
		}
		if result.Image == "" {
			unique := map[string]bool{}
			for alias, file := range v.Textures {
				if result.Textures[alias] != "" {
					unique[file] = true
				}
			}
			if len(unique) == 1 {
				for _, alias := range sortedKeys(result.Textures) {
					result.Image = result.Textures[alias]
					break
				}
			}
		}
	}
	if l.previews != nil {
		l.previews.put(cacheKey, result)
	}
	return result, nil
}
