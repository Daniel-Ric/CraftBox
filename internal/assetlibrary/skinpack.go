package assetlibrary

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type preparedSkin struct{ directory, name string }

func splitSkinPack(ctx context.Context, root string) ([]preparedSkin, error) {
	b, e := readLimit(filepath.Join(root, "skins.json"), 4<<20)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var pack struct {
		LocalizationName string `json:"localization_name"`
		Skins            []struct {
			LocalizationName string `json:"localization_name"`
			Texture          string `json:"texture"`
			Geometry         string `json:"geometry"`
			Cape             string `json:"cape"`
		} `json:"skins"`
	}
	if e = normalizedJSON(b, &pack); e != nil {
		return nil, e
	}
	if len(pack.Skins) == 0 || len(pack.Skins) > 200 {
		return nil, fmt.Errorf("Skin packs must contain between 1 and 200 skins")
	}
	lang := map[string]string{}
	if raw, e := readLimit(filepath.Join(root, "texts", "en_US.lang"), 1<<20); e == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if ok {
				lang[k] = v
			}
		}
	}
	result := []preparedSkin{}
	ok := false
	defer func() {
		if !ok {
			for _, s := range result {
				os.RemoveAll(s.directory)
			}
		}
	}()
	for _, skin := range pack.Skins {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		if _, e = safeName(skin.Texture); e != nil {
			return nil, e
		}
		data, e := readLimit(filepath.Join(root, filepath.FromSlash(skin.Texture)), MaxPreviewBytes)
		if e != nil {
			return nil, e
		}
		if _, e = checkedPNG(data); e != nil {
			return nil, e
		}
		dir, e := os.MkdirTemp(filepath.Dir(root), "skin-entry-")
		if e != nil {
			return nil, e
		}
		name := skin.LocalizationName
		if value := lang["skin."+pack.LocalizationName+"."+name]; value != "" {
			name = value
		}
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(skin.Texture), ".png")
		}
		result = append(result, preparedSkin{dir, name})
		if e = os.WriteFile(filepath.Join(dir, "skin.png"), data, 0600); e != nil {
			return nil, e
		}
		arm := "wide"
		if strings.Contains(strings.ToLower(skin.Geometry), "slim") || strings.Contains(strings.ToLower(skin.Geometry), "alex") {
			arm = "slim"
		}
		meta, _ := json.Marshal(map[string]any{"armModel": arm, "skinId": skin.LocalizationName, "geometryId": skin.Geometry})
		if e = os.WriteFile(filepath.Join(dir, "capture.json"), meta, 0600); e != nil {
			return nil, e
		}
		if skin.Cape != "" {
			if _, e = safeName(skin.Cape); e != nil {
				return nil, e
			}
			cape, e := readLimit(filepath.Join(root, skin.Cape), MaxPreviewBytes)
			if e != nil {
				return nil, e
			}
			if _, e = checkedPNG(cape); e != nil {
				return nil, e
			}
			if e = os.WriteFile(filepath.Join(dir, "cape.png"), cape, 0600); e != nil {
				return nil, e
			}
		}
		if raw, e := readLimit(filepath.Join(root, "geometry.json"), 4<<20); e == nil {
			var geometry map[string]any
			if normalizedJSON(raw, &geometry) == nil {
				if all, ok := geometry["minecraft:geometry"].([]any); ok {
					for _, entry := range all {
						g, _ := entry.(map[string]any)
						d, _ := g["description"].(map[string]any)
						if d["identifier"] == skin.Geometry {
							encoded, _ := json.Marshal(map[string]any{"format_version": "1.12.0", "minecraft:geometry": []any{g}})
							if e = os.WriteFile(filepath.Join(dir, "geometry.json"), encoded, 0600); e != nil {
								return nil, e
							}
							break
						}
					}
				} else if selected, ok := geometry[skin.Geometry]; ok {
					encoded, _ := json.Marshal(map[string]any{"format_version": "1.8.0", skin.Geometry: selected})
					if e = os.WriteFile(filepath.Join(dir, "geometry.json"), encoded, 0600); e != nil {
						return nil, e
					}
				}
			}
		}
	}
	ok = true
	return result, nil
}
