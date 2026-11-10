package assetlibrary

import "fmt"

type ModelSources struct {
	Documents   map[string]any `json:"documents"`
	Animations  map[string]any `json:"animations"`
	Controllers map[string]any `json:"controllers"`
	Aliases     map[string]any `json:"aliases"`
	Scripts     map[string]any `json:"scripts"`
}

func (x objectIndex) modelSources(v Object) ModelSources {
	desc := objectDescription(x, v)
	out := ModelSources{Documents: map[string]any{}, Animations: map[string]any{}, Controllers: map[string]any{}, Aliases: objectMap(desc["animations"]), Scripts: objectMap(desc["scripts"])}
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
		for name := range objectMap(doc["render_controllers"]) {
			owners[name] = file
		}
		particle := objectMap(doc["particle_effect"])
		if name := objectString(objectMap(particle["description"])["identifier"]); name != "" {
			owners[name] = file
		}
	}
	pending := []string{}
	for _, ref := range out.Aliases {
		if name := objectString(ref); name != "" {
			pending = append(pending, name)
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
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if alias := objectString(out.Aliases[name]); alias != "" {
			name = alias
		}
		if visited[name] {
			continue
		}
		visited[name] = true
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
