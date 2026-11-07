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
			if image, err := readAssetFile(root, path); err == nil {
				motion.Particles[alias] = ParticlePreview{particle, image.Data}
			}
		}
	}
	var texture *TextureAnimation
	controllers, _ := desc["render_controllers"].([]any)
	for _, entry := range controllers {
		controller := x.controllers[strings.ToLower(objectString(entry))]
		geoAlias := strings.TrimPrefix(strings.ToLower(objectString(controller["geometry"])), "geometry.")
		if objectString(objectMap(desc["geometry"])[geoAlias]) != v.GeometryID {
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
					image, err := readAssetFile(root, path)
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
		if value != "" {
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
		for identifier := range remaining {
			candidates := []Object{}
			for _, object := range objects {
				if object.Kind == "item-model" && strings.Split(object.Name, " / ")[0] == identifier {
					candidates = append(candidates, object)
				}
			}
			if len(candidates) == 0 {
				continue
			}
			first := candidates[0]
			for _, v := range candidates {
				if strings.HasSuffix(v.GeometryID, ".base") {
					first = v
					break
				}
			}
			for _, v := range candidates {
				include := v.ID == first.ID
				for _, layer := range first.Layers {
					include = include || layer == v.GeometryID
				}
				if !include || len(out.Attachments) >= 16 {
					continue
				}
				preview, err := l.ObjectPreview(pack.ID, v.ID, "")
				if err == nil && preview.Geometry != nil && preview.Image != "" {
					for slot, value := range slots {
						if value == identifier {
							preview.CapturedSlot = slot
							break
						}
					}
					out.Attachments = append(out.Attachments, preview)
				}
			}
			delete(remaining, identifier)
		}
	}
	for identifier := range remaining {
		out.Warnings = append(out.Warnings, "Resource pack missing for "+identifier)
	}
	return out, nil
}
