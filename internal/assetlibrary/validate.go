package assetlibrary

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func validate(root string, d *Detail) {
	if d.Item.Encrypted {
		d.Issues = append(d.Issues, Issue{"info", "encrypted-pack", "contents.json", "This pack contains encrypted files. Capture the server again to add its readable session copy", ""})
		return
	}
	names := map[string]bool{}
	lower := map[string]string{}
	for _, f := range d.Files {
		names[f.Path] = true
		lower[strings.ToLower(f.Path)] = f.Path
	}
	issue := func(severity, code, file, message, reference string) {
		if len(d.Issues) < 2000 {
			d.Issues = append(d.Issues, Issue{severity, code, file, message, reference})
		}
	}
	checkTexture := func(file, ref string) {
		if strings.HasSuffix(file, ".texture_set.json") && !strings.HasPrefix(ref, "textures/") {
			ref = path.Join(path.Dir(file), ref)
		}
		if ref == "" || strings.HasPrefix(ref, "#") || !strings.Contains(ref, "/") {
			return
		}
		ref = strings.TrimPrefix(ref, "./")
		candidates := []string{ref}
		if path.Ext(ref) == "" {
			candidates = []string{ref + ".png", ref + ".tga", ref + ".jpg"}
		}
		for _, p := range candidates {
			if names[p] {
				return
			}
		}
		for _, p := range candidates {
			if found := lower[strings.ToLower(p)]; found != "" {
				issue("warning", "path-case", file, "Texture path has different capitalization", found)
				return
			}
		}
		issue("warning", "missing-texture", file, "Texture was not found in this pack; it may be supplied by another pack or Minecraft", ref)
	}
	geometryIDs := map[string]bool{}
	references := map[string][]string{}
	for _, f := range d.Files {
		if f.Kind == "texture" {
			source := previewFS{os.DirFS(root)}
			opened, e := source.Open(f.Path)
			if e != nil {
				issue("warning", "texture-preview", f.Path, e.Error(), "")
			} else {
				opened.Close()
			}
		}
		if !strings.EqualFold(path.Ext(f.Path), ".json") {
			continue
		}
		if f.Bytes > 4<<20 {
			issue("info", "large-json", f.Path, "JSON is too large for the interactive check", "")
			continue
		}
		b, e := readLimit(filepath.Join(root, filepath.FromSlash(f.Path)), 4<<20)
		if e != nil {
			issue("error", "read", f.Path, e.Error(), "")
			continue
		}
		var raw any
		if e = normalizedJSON(b, &raw); e != nil {
			issue("error", "invalid-json", f.Path, e.Error(), "")
			continue
		}
		if f.Path == "capture.json" && d.Item.Kind == "skins" {
			if m, ok := raw.(map[string]any); ok {
				d.Metadata = m
			}
		}
		if f.Path == "manifest.json" {
			if m, ok := raw.(map[string]any); ok {
				d.Metadata = m
				header, _ := m["header"].(map[string]any)
				if header == nil {
					issue("error", "manifest-header", f.Path, "Pack header is missing", "")
				} else {
					if _, e = uuid.Parse(fmt.Sprint(header["uuid"])); e != nil {
						issue("error", "manifest-uuid", f.Path, "Header UUID is invalid", "")
					}
					if textVersion(header["version"]) == "" {
						issue("error", "manifest-version", f.Path, "Pack version is missing", "")
					}
				}
				mods, _ := m["modules"].([]any)
				if len(mods) == 0 {
					issue("error", "manifest-modules", f.Path, "Pack modules are missing", "")
				}
				seen := map[string]bool{d.Item.UUID: true}
				for _, v := range mods {
					mod, _ := v.(map[string]any)
					id, _ := mod["uuid"].(string)
					if _, e = uuid.Parse(id); e != nil {
						issue("error", "module-uuid", f.Path, "Module UUID is invalid", id)
					}
					if seen[id] {
						issue("error", "duplicate-uuid", f.Path, "Header and modules must use different UUIDs", id)
					}
					seen[id] = true
				}
			}
		}
		var walk func(any, string)
		walk = func(v any, key string) {
			switch t := v.(type) {
			case map[string]any:
				for k, value := range t {
					if k == "identifier" {
						if s, ok := value.(string); ok && strings.HasPrefix(s, "geometry.") {
							geometryIDs[s] = true
						}
					}
					walk(value, k)
				}
			case []any:
				for _, v := range t {
					walk(v, key)
				}
			case string:
				if (key == "texture" || key == "textures" || key == "color" || key == "normal" || key == "heightmap" || key == "metalness_emissive_roughness" || key == "metalness_emissive_roughness_subsurface") && strings.HasPrefix(t, "textures/") {
					checkTexture(f.Path, t)
				}
				if strings.HasPrefix(t, "geometry.") && key != "identifier" {
					references[t] = append(references[t], f.Path)
				}
			}
		}
		walk(raw, "")
		if strings.Contains(f.Path, "terrain_texture") || strings.Contains(f.Path, "item_texture") {
			var walkTextures func(any)
			walkTextures = func(v any) {
				switch t := v.(type) {
				case map[string]any:
					for _, v := range t {
						walkTextures(v)
					}
				case []any:
					for _, v := range t {
						walkTextures(v)
					}
				case string:
					if strings.HasPrefix(t, "textures/") {
						checkTexture(f.Path, t)
					}
				}
			}
			walkTextures(raw)
		}
	}
	for id, files := range references {
		if !geometryIDs[id] && !strings.Contains(id, "humanoid") {
			for _, p := range files {
				issue("info", "geometry-reference", p, "Geometry is not defined in this pack; check the active pack stack", id)
			}
		}
	}
	if d.Item.Kind == "skins" {
		if !((d.Item.Width == 64 && d.Item.Height == 32) || (d.Item.Width == 64 && d.Item.Height == 64) || (d.Item.Width == 128 && d.Item.Height == 128)) {
			issue("warning", "skin-size", "skin.png", "This texture size may require custom geometry or may not import as a classic skin", "")
		}
		if d.Item.HasGeometry || d.Item.Persona {
			issue("info", "custom-skin", "geometry.json", "Custom geometry is available for previews and model export. Minecraft skin-pack export uses a classic model", "")
		}
	}
	if names["contents.json"] {
		issue("info", "content-index", "contents.json", "This pack contains a content index. Encrypted entries may not be readable", "")
	}
	if d.Item.Kind == "packs" && d.Item.MinimumVersion == "" {
		issue("warning", "minimum-version", "manifest.json", "Minimum Minecraft version is not specified", "")
	}
	for _, f := range d.Files {
		if f.Kind == "texture" && f.Bytes > 8<<20 {
			issue("info", "large-texture", f.Path, "Large texture; decoded memory use can be much higher than the file size", "")
		}
	}
}
