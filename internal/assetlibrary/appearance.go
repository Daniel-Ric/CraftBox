package assetlibrary

import (
	"sort"
	"strings"
)

type ParticlePreview struct {
	Definition any    `json:"definition"`
	Image      string `json:"image"`
}
type MotionRuntime struct {
	Visibility     []RuntimeVisibility        `json:"visibility,omitempty"`
	PreviewQueries map[string]float64         `json:"previewQueries,omitempty"`
	Aliases        map[string]any             `json:"aliases"`
	Animations     map[string]any             `json:"animations"`
	Controllers    map[string]any             `json:"controllers"`
	Scripts        map[string]any             `json:"scripts"`
	Particles      map[string]ParticlePreview `json:"particles"`
	HideCape       bool                       `json:"hideCape"`
}
type TextureAnimation struct {
	Frames     []string `json:"frames"`
	Expression string   `json:"expression"`
	Scripts    any      `json:"scripts"`
}
type SkinAppearance struct {
	BodyID      string          `json:"bodyId"`
	Variants    []Item          `json:"variants"`
	Attachments []ObjectPreview `json:"attachments"`
	Warnings    []string        `json:"warnings"`
}

func (l *Library) objectRuntime(id string, x objectIndex, v Object) (*MotionRuntime, *TextureAnimation) {
	root, _ := l.RootFor(id)
	doc := x.docs[v.Definition]
	desc := objectMap(objectMap(doc["minecraft:attachable"])["description"])
	if len(desc) == 0 {
		desc = objectMap(objectMap(doc["minecraft:client_entity"])["description"])
	}
	if len(desc) == 0 {
		return nil, nil
	}
	motion := &MotionRuntime{Aliases: objectMap(desc["animations"]), Animations: map[string]any{}, Controllers: map[string]any{}, Scripts: objectMap(desc["scripts"]), Particles: map[string]ParticlePreview{}}
	motion.PreviewQueries = mergePreviewQueries(nil, v.PreviewQueries)
	motion.Visibility = v.RenderVisibility
	motion.HideCape = strings.Contains(objectString(motion.Scripts["parent_setup"]), "hide_cape")
	for _, doc := range x.docs {
		for name, animation := range objectMap(doc["animations"]) {
			for _, ref := range motion.Aliases {
				if ref == name {
					motion.Animations[name] = animation
				}
			}
		}
		for name, controller := range objectMap(doc["animation_controllers"]) {
			for _, ref := range motion.Aliases {
				if ref == name {
					motion.Controllers[name] = controller
				}
			}
		}
	}
	for alias, ref := range objectMap(desc["particle_effects"]) {
		for _, doc := range x.docs {
			particle := objectMap(doc["particle_effect"])
			description := objectMap(particle["description"])
			if description["identifier"] != ref {
				continue
			}
			path := x.texture(objectString(objectMap(description["basic_render_parameters"])["texture"]))
			if image, err := x.readTexture(root, path); err == nil {
				motion.Particles[alias] = ParticlePreview{particle, image.Data}
			}
		}
	}
	var texture *TextureAnimation
	controllers, _ := desc["render_controllers"].([]any)
	for _, entry := range controllers {
		controller := x.controllers[strings.ToLower(objectString(entry))]
		geoAlias := strings.TrimPrefix(strings.ToLower(objectString(controller["geometry"])), "geometry.")
		if resolveGeometryID(x.geometry, objectString(objectMap(desc["geometry"])[geoAlias])) != v.GeometryID {
			continue
		}
		refs, _ := controller["textures"].([]any)
		for _, entry := range refs {
			expression := objectString(entry)
			arrayName, indexExpression, ok := controllerArray(expression)
			if !ok {
				continue
			}
			for name, values := range objectMap(objectMap(controller["arrays"])["textures"]) {
				if strings.ToLower(name) != arrayName {
					continue
				}
				frames, _ := values.([]any)
				if len(frames) > 64 {
					continue
				}
				sequence := &TextureAnimation{Expression: indexExpression, Scripts: motion.Scripts["pre_animation"]}
				var bytes int
				for _, frame := range frames {
					alias := strings.TrimPrefix(strings.ToLower(objectString(frame)), "texture.")
					path := x.texture(objectString(objectMap(desc["textures"])[alias]))
					image, err := x.readTexture(root, path)
					if err != nil {
						continue
					}
					bytes += len(image.Data)
					if bytes > 16<<20 {
						break
					}
					sequence.Frames = append(sequence.Frames, image.Data)
				}
				if len(sequence.Frames) > 1 {
					texture = sequence
				}
			}
		}
	}
	return motion, texture
}

func (l *Library) SkinAppearance(id string) (SkinAppearance, error) {
	d, err := l.Detail(id)
	if err != nil {
		return SkinAppearance{}, err
	}
	out := SkinAppearance{BodyID: id, Variants: []Item{}, Attachments: []ObjectPreview{}, Warnings: []string{}}
	items, err := l.List()
	if err != nil {
		return out, err
	}
	identity := objectString(d.Metadata["playerUUID"])
	for _, item := range items {
		if item.Kind == "skins" && item.CaptureType != "display" && identity != "" && item.PlayerUUID == identity {
			out.Variants = append(out.Variants, item)
		}
	}
	sort.SliceStable(out.Variants, func(i, j int) bool {
		a, b := out.Variants[i], out.Variants[j]
		if a.Persona != b.Persona {
			return !a.Persona
		}
		return a.Created > b.Created
	})
	if len(out.Variants) > 0 {
		out.BodyID = out.Variants[0].ID
	}
	slots := l.capturedEquipment(d.Metadata)
	remaining := map[string]bool{}
	for _, value := range slots {
		if value != "" && value != "minecraft:air" {
			remaining[value] = true
		}
	}
	for _, pack := range items {
		if len(remaining) == 0 {
			break
		}
		if pack.Kind != "packs" || pack.Encrypted {
			continue
		}
		objects, err := l.Objects(pack.ID)
		if err != nil {
			continue
		}
		for _, identifier := range sortedKeys(remaining) {
			candidates := []Object{}
			playerCandidates := []Object{}
			for _, object := range objects {
				if object.Kind != "item-model" {
					continue
				}
				name, _, _ := strings.Cut(object.Name, " / ")
				switch name {
				case identifier:
					candidates = append(candidates, object)
				case identifier + ".player":
					playerCandidates = append(playerCandidates, object)
				}
			}
			for _, group := range [][]Object{playerCandidates, candidates} {
				if len(group) == 0 {
					continue
				}
				first := group[0]
				for _, v := range group {
					if strings.HasSuffix(v.GeometryID, ".base") {
						first = v
						break
					}
				}
				resolved := false
				for _, v := range group {
					include := v.ID == first.ID
					for _, layer := range first.Layers {
						include = include || layer == v.GeometryID
					}
					if !include {
						continue
					}
					if v.invisibleAttachable {
						resolved = true
						continue
					}
					if len(out.Attachments) >= 16 {
						continue
					}
					preview, err := l.ObjectPreview(pack.ID, v.ID, "")
					if err == nil && preview.Geometry != nil && preview.Image != "" {
						preview.CapturedSlot = equipmentSlot(slots, identifier)
						out.Attachments = append(out.Attachments, preview)
						resolved = true
					}
				}
				if resolved {
					delete(remaining, identifier)
					break
				}
			}
		}
	}
	for _, identifier := range sortedKeys(remaining) {
		if preview, ok := vanillaArmorPreview(identifier); ok && len(out.Attachments) < 16 {
			preview.CapturedSlot = equipmentSlot(slots, identifier)
			out.Attachments = append(out.Attachments, preview)
			continue
		}
		out.Warnings = append(out.Warnings, "Resource pack missing for "+identifier)
	}
	return out, nil
}

func equipmentSlot(slots map[string]string, identifier string) string {
	for _, slot := range sortedKeys(slots) {
		if slots[slot] == identifier {
			return slot
		}
	}
	return ""
}
