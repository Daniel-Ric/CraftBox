package worlds

import (
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/internal/ingamefeedback"
	"regexp"
	"strings"
)

var menuFormatting = regexp.MustCompile(`(?i)§[0-9a-z]`)

func menuActionText(value string, index int, settings ingamefeedback.Settings) string {
	first, detail, _ := strings.Cut(value, "\n")
	colors := []string{"§d", "§9", "§5", "§e"}
	text := colors[index%len(colors)] + first + "§r"
	if detail != "" {
		text += "\n§8" + detail + "§r"
	}
	if strings.HasPrefix(first, "Back") || first == "Cancel" {
		text = "§8" + first + "§r"
	}
	if !settings.MenuColors {
		text = menuFormatting.ReplaceAllString(text, "")
	}
	return text
}
func presentMenuForm(form any, settings ingamefeedback.Settings) any {
	if settings.MenuColors {
		return form
	}
	raw, err := json.Marshal(form)
	if err != nil {
		return form
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return form
	}
	var walk func(any, string) any
	walk = func(value any, key string) any {
		switch v := value.(type) {
		case map[string]any:
			for name, item := range v {
				v[name] = walk(item, name)
			}
		case []any:
			for i, item := range v {
				v[i] = walk(item, key)
			}
		case string:
			switch key {
			case "title", "text", "content", "options", "placeholder", "button1", "button2":
				return menuFormatting.ReplaceAllString(v, "")
			}
		}
		return value
	}
	return walk(value, "")
}
