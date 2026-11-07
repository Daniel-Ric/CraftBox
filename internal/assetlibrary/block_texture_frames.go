package assetlibrary

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"path/filepath"
	"slices"
	"strings"
)

// Geometry UV dimensions describe one atlas tile. Tall block textures may store
// several tiles vertically; stretching that entire strip corrupts every UV.
// Only split textures actually referenced by the terrain atlas, never arbitrary
// entity textures, UI images, or the original files in the library.
func blockTextureFrames(root string, x objectIndex, v Object, file string, geometry any) ([]string, *TextureAnimation) {
	if !strings.HasPrefix(v.GeometryFile, "models/blocks/") {
		return nil, nil
	}
	atlas := false
	for _, refs := range x.terrain {
		for _, ref := range refs {
			if x.texture(ref) == file {
				atlas = true
			}
		}
	}
	if !atlas {
		return nil, nil
	}
	var desc map[string]any
	if geos, ok := objectMap(geometry)["minecraft:geometry"].([]any); ok && len(geos) > 0 {
		desc = objectMap(objectMap(geos[0])["description"])
	}
	width, _ := desc["texture_width"].(float64)
	height, _ := desc["texture_height"].(float64)
	if width <= 0 || height <= 0 {
		return nil, nil
	}
	raw, err := readLimit(filepath.Join(root, filepath.FromSlash(file)), MaxPreviewBytes)
	if err != nil {
		return nil, nil
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width < 1 || int64(config.Width)*int64(config.Height) > 16<<20 {
		return nil, nil
	}
	frameHeight := float64(config.Width) * height / width
	h := int(frameHeight)
	if h < 1 || float64(h) != frameHeight || config.Height <= h || config.Height%h != 0 || config.Height/h > 64 {
		return nil, nil
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, nil
	}
	order, ticks := x.blockFrameOrder(file, config.Height/h)
	frames := make([]string, 0, len(order))
	total := 0
	for _, n := range order {
		tile := image.NewNRGBA(image.Rect(0, 0, config.Width, h))
		draw.Draw(tile, tile.Bounds(), source, image.Pt(0, n*h), draw.Src)
		var encoded bytes.Buffer
		if png.Encode(&encoded, tile) != nil {
			return nil, nil
		}
		data := imageURL(encoded.Bytes())
		total += len(data)
		if total > 16<<20 {
			return nil, nil
		}
		frames = append(frames, data)
	}
	if len(frames) > 1 {
		return frames, &TextureAnimation{Frames: frames, Expression: fmt.Sprintf("math.mod(math.floor(query.life_time * %g), %d)", 20/ticks, len(frames))}
	}
	return frames, nil
}

func (x objectIndex) blockFrameOrder(file string, count int) ([]int, float64) {
	var selected []int
	var speed float64
	entries, _ := x.docs["textures/flipbook_textures.json"]["entries"].([]any)
	for _, entry := range entries {
		m := objectMap(entry)
		if x.texture(objectString(m["flipbook_texture"])) != file {
			continue
		}
		order := []int{0}
		ticks, _ := m["ticks_per_frame"].(float64)
		if ticks <= 0 {
			ticks = 1
		}
		// Atlas variants and frame blending require runtime block state; use
		// a static tile rather than inventing which animation is active.
		_, indexed := m["atlas_index"]
		_, variant := m["atlas_tile_variant"]
		if indexed || variant || m["blend_frames"] == true {
			ticks = 0
			if start, ok := m["atlas_index"].(float64); ok && start >= 0 && start < float64(count) && float64(int(start)) == start {
				order[0] = int(start)
			}
		} else if values, ok := m["frames"].([]any); ok {
			order = nil
			if len(values) > 0 && len(values) <= 64 {
				for _, value := range values {
					n, ok := value.(float64)
					if !ok || n < 0 || n >= float64(count) || float64(int(n)) != n {
						order = nil
						break
					}
					order = append(order, int(n))
				}
			}
			if len(order) == 0 {
				order = []int{0}
				ticks = 0
			}
		} else {
			order = make([]int, count)
			for i := range order {
				order[i] = i
			}
		}
		if selected != nil && (!slices.Equal(selected, order) || speed != ticks) {
			return []int{0}, 0
		}
		selected = order
		speed = ticks
	}
	if selected == nil {
		return []int{0}, 0
	}
	return selected, speed
}
