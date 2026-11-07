package assetlibrary

import "strings"

type BoneMaterial struct {
	Pattern  string `json:"pattern"`
	Material string `json:"material"`
}
type BoneVisibility struct {
	Pattern string `json:"pattern"`
	Visible bool   `json:"visible"`
}

func previewVisibility(value any) (bool, bool) {
	if result, ok := value.(bool); ok {
		return result, true
	}
	if result, ok := value.(float64); ok {
		return result != 0, true
	}
	expression := strings.ToLower(strings.TrimSpace(objectString(value)))
	if parts := strings.Split(expression, "||"); len(parts) > 1 {
		result := false
		for _, part := range parts {
			v, ok := previewVisibility(part)
			if !ok {
				return false, false
			}
			result = result || v
		}
		return result, true
	}
	if parts := strings.Split(expression, "&&"); len(parts) > 1 {
		result := true
		for _, part := range parts {
			v, ok := previewVisibility(part)
			if !ok {
				return false, false
			}
			result = result && v
		}
		return result, true
	}
	if strings.HasPrefix(expression, "!") {
		v, ok := previewVisibility(strings.TrimSpace(expression[1:]))
		return !v, ok
	}
	switch expression {
	case "true", "1", "1.0":
		return true, true
	case "false", "0", "0.0", "variable.is_first_person", "v.is_first_person", "query.is_invisible", "q.is_invisible", "query.is_sneaking", "q.is_sneaking", "variable.is_sneaking", "v.is_sneaking":
		return false, true
	}
	return false, false
}
