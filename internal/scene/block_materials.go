package scene

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dblezek/tga"
	"image"
	"image/color"
	"io/fs"
	"path"
	"sort"
	"strings"
)

func (c *Catalog) MissingTexturePaths() []string {
	paths := []string{}
	for key, t := range c.tiles {
		if key != "" && t.missing {
			paths = append(paths, key)
		}
	}
	sort.Strings(paths)
	return paths
}

type blockDefinition struct {
	Description struct {
		Identifier string         `json:"identifier"`
		States     map[string]any `json:"states"`
	} `json:"description"`
	Components   map[string]any `json:"components"`
	Permutations []struct {
		Condition  string         `json:"condition"`
		Components map[string]any `json:"components"`
	} `json:"permutations"`
}

func (c *Catalog) AddBlockDefinition(data []byte) error {
	data, err := standardizePackJSON(data)
	if err != nil {
		return err
	}
	var wrapper struct {
		Block blockDefinition `json:"minecraft:block"`
	}
	if err = json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	d := wrapper.Block
	if d.Description.Identifier == "" {
		if err = json.Unmarshal(data, &d); err != nil {
			return err
		}
	}
	if d.Description.Identifier == "" {
		return fmt.Errorf("block definition without identifier")
	}
	c.SetBlockComponents(d.Description.Identifier, d.Components)
	c.definitions[strings.TrimPrefix(d.Description.Identifier, "minecraft:")] = d
	return nil
}

func (c *Catalog) AddBlockDefinitions(f fs.FS) error {
	return fs.WalkDir(f, "blocks", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == "blocks" && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".json") {
			return nil
		}
		raw, err := fs.ReadFile(f, p)
		if err != nil {
			return err
		}
		return c.AddBlockDefinition(raw)
	})
}

func materialTable(components map[string]any) map[string]any {
	m, _ := components["minecraft:material_instances"].(map[string]any)
	if nested, ok := m["materials"].(map[string]any); ok {
		out := make(map[string]any, len(nested)+6)
		for k, v := range nested {
			out[k] = v
		}
		if mappings, ok := m["mappings"].(map[string]any); ok {
			for k, v := range mappings {
				out[k] = v
			}
		}
		return out
	}
	return m
}

func (c *Catalog) legacyMaterials(name string) map[string]any {
	faces := c.blocks[strings.TrimPrefix(name, "minecraft:")]
	materials := make(map[string]any, 6)
	for _, face := range faceNames {
		texture := faces[face]
		if texture == "" {
			texture = faces["side"]
		}
		if texture == "" {
			texture = faces["all"]
		}
		if texture == "" {
			texture = faces["up"]
		}
		if texture != "" {
			materials[face] = map[string]any{"texture": texture}
		}
	}
	return materials
}

func resolveMaterial(table map[string]any, name string) map[string]any {
	value, ok := table[name]
	if !ok {
		value = table["*"]
	}
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		if m, ok := value.(map[string]any); ok {
			return m
		}
		ref, ok := value.(string)
		if !ok || seen[ref] {
			return nil
		}
		seen[ref] = true
		value = table[ref]
	}
	return nil
}

func renderMode(material map[string]any) string {
	method, _ := material["render_method"].(string)
	if strings.HasPrefix(method, "blend") {
		return "blend"
	}
	if strings.HasPrefix(method, "alpha_test") || method == "double_sided" {
		return "mask"
	}
	return "opaque"
}

func materialTint(key string) (color.NRGBA, bool) {
	values := map[string]color.NRGBA{
		"default_foliage": {119, 171, 47, 255}, "birch_foliage": {128, 167, 85, 255}, "evergreen_foliage": {97, 153, 97, 255},
		"dry_foliage": {157, 146, 96, 255}, "grass": {145, 189, 89, 255}, "water": {63, 118, 228, 255}, "none": {255, 255, 255, 255},
	}
	for method, v := range values {
		if strings.Contains(key, "#tint:"+method+"#") {
			return v, true
		}
	}
	return color.NRGBA{}, false
}

func (c *Catalog) materialTexture(material map[string]any, tw, th float32) tile {
	key, _ := material["texture"].(string)
	if key == "" {
		return c.tiles[""]
	}
	p := c.terrain[key]
	if p == "" {
		p = key
		if !strings.HasPrefix(p, "textures/") {
			p = "textures/blocks/" + p
		}
	}
	cacheKey := p + fmt.Sprintf("#model:%g,%g", tw, th)
	if tint, ok := material["tint_method"].(string); ok {
		cacheKey += "#tint:" + tint + "#"
	} else {
		cacheKey += "#tint:none#"
	}
	if flag(material["alpha_masked_tint"]) {
		cacheKey += "#masked-tint"
	}
	if t, ok := c.tiles[cacheKey]; ok {
		return t
	}
	if fs.ValidPath(p) {
		for _, source := range c.sources {
			for _, ext := range []string{"", ".png", ".tga"} {
				raw, err := fs.ReadFile(source, p+ext)
				if err != nil {
					continue
				}
				var img image.Image
				if strings.EqualFold(path.Ext(p+ext), ".tga") {
					img, err = tga.Decode(bytes.NewReader(raw))
				} else {
					img, _, err = image.Decode(bytes.NewReader(raw))
				}
				if err != nil || img.Bounds().Dx() < 1 || img.Bounds().Dy() < 1 {
					continue
				}

				bounds := img.Bounds()
				w, h := bounds.Dx(), bounds.Dy()
				if tw > 0 && th > 0 {
					h = min(h, max(1, int(float32(w)*th/tw)))
				}
				crop := textureFrame{Image: img, bounds: image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+w, bounds.Min.Y+h)}
				scale := min(float32(1), 256/float32(max(w, h)))
				return c.addTileSized(cacheKey, crop, false, max(1, int(float32(w)*scale)), max(1, int(float32(h)*scale)))
			}
		}
	}
	t := c.tiles[""]
	t.missing = true
	c.tiles[cacheKey] = t
	return t
}

type textureFrame struct {
	image.Image
	bounds image.Rectangle
}

func (f textureFrame) Bounds() image.Rectangle { return f.bounds }
