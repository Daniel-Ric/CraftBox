package assetlibrary

import "strings"

type packMaterial struct {
	parent     string
	definition map[string]any
}

func (x objectIndex) materialMode(name string) string {
	if _, exists := x.materials[strings.ToLower(name)]; !exists {
		return name
	}
	states, defines := map[string]bool{}, map[string]bool{}
	seen := map[string]bool{}
	base := ""
	var visit func(string) bool
	visit = func(name string) bool {
		key := strings.ToLower(name)
		if seen[key] || len(seen) >= 32 {
			return false
		}
		seen[key] = true
		m, custom := x.materials[key]
		if !custom {
			base = name
			states["Blending"] = strings.Contains(key, "alphablend") || strings.Contains(key, "slime_outer") || key == "blend"
			defines["USE_EMISSIVE"] = strings.Contains(key, "emissive")
			return true
		}
		if m.parent == "" || !visit(m.parent) {
			return false
		}
		for _, property := range []string{"states", "defines"} {
			target := states
			if property == "defines" {
				target = defines
			}
			if values, ok := m.definition[property].([]any); ok {
				clear(target)
				for _, value := range values {
					target[objectString(value)] = true
				}
			}
			for _, prefix := range []string{"+", "-"} {
				values, _ := m.definition[prefix+property].([]any)
				for _, value := range values {
					target[objectString(value)] = prefix == "+"
				}
			}
		}
		return true
	}
	if !visit(name) {
		return name
	}
	mode := "entity_alphatest"
	if strings.Contains(base, "one_sided") || strings.Contains(base, "single_sided") {
		mode += " one_sided"
	}
	if states["Blending"] {
		mode += " alphablend"
	}
	if defines["USE_EMISSIVE"] {
		mode += " emissive"
	}
	return mode
}
