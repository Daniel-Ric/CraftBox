package assetlibrary

import (
	"fmt"
	"strings"
)

type ModelSources struct {
	Documents   map[string]any `json:"documents"`
	Animations  map[string]any `json:"animations"`
	Controllers map[string]any `json:"controllers"`
	Aliases     map[string]any `json:"aliases"`
	Scripts     map[string]any `json:"scripts"`
	Warnings    []string       `json:"warnings"`
}

func (x objectIndex) modelSources(v Object) ModelSources {
	desc := objectDescription(x, v)
	out := ModelSources{Documents: map[string]any{}, Animations: map[string]any{}, Controllers: map[string]any{}, Aliases: objectMap(desc["animations"]), Scripts: objectMap(desc["scripts"]), Warnings: []string{}}
	addDocument := func(path string) {
		if doc := x.docs[path]; doc != nil {
			out.Documents[path] = doc
		}
	}
	addDocument(v.Definition)
	addDocument(v.GeometryFile)
	animations, controllers, owners := map[string]any{}, map[string]any{}, map[string]string{}
	for _, file := range sortedKeys(x.docs) {
		doc := x.docs[file]
		for name, animation := range objectMap(doc["animations"]) {
			animations[name] = animation
			owners[name] = file
		}
		for name, controller := range objectMap(doc["animation_controllers"]) {
			controllers[name] = controller
			owners[name] = file
		}
		for _, ref := range objectMap(desc["sound_effects"]) {
			if objectMap(doc["sound_definitions"])[objectString(ref)] != nil {
				addDocument(file)
			}
		}
		for name := range objectMap(doc["render_controllers"]) {
			owners[name] = file
		}
		particle := objectMap(doc["particle_effect"])
		if name := objectString(objectMap(particle["description"])["identifier"]); name != "" {
			owners[name] = file
		}
	}
	pending := []string{}
	for _, alias := range sortedKeys(out.Aliases) {
		ref := out.Aliases[alias]
		if name := objectString(ref); name != "" {
			pending = append(pending, name)
		}
	}
	if refs, ok := out.Scripts["animate"].([]any); ok {
		for _, ref := range refs {
			if name := objectString(ref); name != "" {
				pending = append(pending, name)
			} else {
				pending = append(pending, sortedKeys(objectMap(ref))...)
			}
		}
	}
	for name, file := range v.Animations {
		if animation := objectMap(x.docs[file]["animations"])[name]; animation != nil {
			animations[name] = animation
			owners[name] = file
		}
		pending = append(pending, name)
		addDocument(file)
	}
	for _, ref := range objectMap(desc["particle_effects"]) {
		addDocument(owners[objectString(ref)])
	}
	for _, field := range []string{"render_controllers", "animation_controllers"} {
		entries, _ := desc[field].([]any)
		for _, entry := range entries {
			if name := objectString(entry); name != "" {
				addDocument(owners[name])
				if field == "animation_controllers" {
					pending = append(pending, name)
				}
			} else {
				for key, value := range objectMap(entry) {
					name := key
					if field == "animation_controllers" {
						if ref := objectString(value); ref != "" {
							name = ref
						}
						pending = append(pending, name)
					}
					addDocument(owners[name])
				}
			}
		}
	}
	visited := map[string]bool{}
	canonical := map[string]string{}
	for _, name := range sortedKeys(animations) {
		canonical[strings.ToLower(name)] = name
	}
	for _, name := range sortedKeys(controllers) {
		canonical[strings.ToLower(name)] = name
	}
	aliasIndex := map[string]string{}
	for _, name := range sortedKeys(out.Aliases) {
		aliasIndex[strings.ToLower(name)] = objectString(out.Aliases[name])
	}
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		aliases := map[string]bool{}
		for {
			key := strings.ToLower(name)
			if aliases[key] {
				out.Warnings = append(out.Warnings, fmt.Sprintf("Cyclic animation alias: %s", name))
				break
			}
			aliases[key] = true
			alias := objectString(out.Aliases[name])
			if alias == "" {
				alias = aliasIndex[key]
			}
			if alias == "" || strings.EqualFold(alias, name) {
				break
			}
			name = alias
		}
		if animations[name] == nil && controllers[name] == nil {
			if exact := canonical[strings.ToLower(name)]; exact != "" {
				name = exact
			}
		}
		if visited[strings.ToLower(name)] {
			continue
		}
		visited[strings.ToLower(name)] = true
		if animations[name] == nil && controllers[name] == nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("Animation source is not present in the captured pack: %s", name))
		}
		if animation := animations[name]; animation != nil {
			out.Animations[name] = animation
			addDocument(owners[name])
		}
		if controller := controllers[name]; controller != nil {
			out.Controllers[name] = controller
			addDocument(owners[name])
			for _, state := range objectMap(objectMap(controller)["states"]) {
				entries, _ := objectMap(state)["animations"].([]any)
				for _, entry := range entries {
					if ref := objectString(entry); ref != "" {
						pending = append(pending, ref)
					} else {
						for ref := range objectMap(entry) {
							pending = append(pending, ref)
						}
					}
				}
			}
		}
	}
	warnings := map[string]bool{}
	for _, warning := range out.Warnings {
		warnings[warning] = true
	}
	out.Warnings = sortedKeys(warnings)
	return out
}

func (l *Library) ModelSources(id, objectID string) (ModelSources, error) {
	x, err := l.catalog(id)
	if err != nil {
		return ModelSources{}, err
	}
	if objectID == "" {
		return x.modelSources(Object{Definition: "capture.json", GeometryFile: "geometry.json"}), nil
	}
	for _, v := range x.objects() {
		if v.ID == objectID {
			return x.modelSources(v), nil
		}
	}
	return ModelSources{}, fmt.Errorf("Selected model is no longer available")
}
