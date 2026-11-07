package assetlibrary

import (
	"strconv"
	"strings"
)

func controllerLayers(description map[string]any, controllers map[string]map[string]any) []string {
	layers := []string{}
	refs, _ := description["render_controllers"].([]any)
	for _, ref := range refs {
		name := objectString(ref)
		if name == "" {
			continue
		}
		controller := controllers[strings.ToLower(name)]
		geometry := strings.ToLower(objectString(controller["geometry"]))
		if !strings.HasPrefix(geometry, "geometry.") || strings.ContainsAny(geometry, " ?[]()") {
			continue
		}
		alias := strings.TrimPrefix(geometry, "geometry.")
		for key, value := range objectMap(description["geometry"]) {
			if strings.EqualFold(key, alias) {
				layers = append(layers, objectString(value))
				break
			}
		}
	}
	return layers
}

func controllerArray(expression string) (string, string, bool) {
	expression = strings.TrimSpace(expression)
	start := strings.Index(expression, "[")
	if start < 0 || !strings.HasSuffix(expression, "]") || !strings.HasPrefix(strings.ToLower(expression), "array.") {
		return "", "", false
	}
	return strings.ToLower(strings.TrimSpace(expression[:start])), strings.TrimSpace(expression[start+1 : len(expression)-1]), true
}
func controllerGeometryIndex(controller map[string]any, alias string) (int, string) {
	if condition, yes, no, ok := controllerConditional(objectString(controller["geometry"])); ok {
		if strings.EqualFold(yes, "geometry."+alias) && strings.EqualFold(no, "geometry."+alias) {
			return -1, ""
		}
		if strings.EqualFold(yes, "geometry."+alias) {
			return 1, condition
		}
		if strings.EqualFold(no, "geometry."+alias) {
			return 0, condition
		}
		return -2, condition
	}
	name, expr, ok := controllerArray(objectString(controller["geometry"]))
	if !ok {
		return -1, ""
	}
	values, _ := resourceValue(objectMap(objectMap(controller["arrays"])["geometries"]), name).([]any)
	for i, value := range values {
		if strings.EqualFold(objectString(value), "geometry."+alias) {
			return i, expr
		}
	}
	return -2, expr
}
func controllerResource(controller map[string]any, kind, expression string, index int, geometryExpression string) string {
	expression = strings.TrimSpace(expression)
	if condition, yes, no, ok := controllerConditional(expression); ok {
		if index >= 0 && sameControllerIndex(unwrapControllerExpression(condition), unwrapControllerExpression(geometryExpression)) {
			if index != 0 {
				return yes
			}
			return no
		}
		if geometryExpression != "" {
			return ""
		}
		return no
	}
	if name, expr, ok := controllerArray(expression); ok {
		values, _ := resourceValue(objectMap(objectMap(controller["arrays"])[kind]), name).([]any)
		chosen := 0
		if value, err := strconv.Atoi(expr); err == nil {
			chosen = value
		} else if sameControllerIndex(expr, geometryExpression) && index >= 0 {
			chosen = index
		} else if index >= 0 && geometryExpression != "" {
			return ""
		}
		if chosen >= 0 && chosen < len(values) {
			return objectString(values[chosen])
		}
		return ""
	}
	return expression
}
