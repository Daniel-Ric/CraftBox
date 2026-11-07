package scene

import (
	"strconv"
	"strings"
)

var vanillaEmission = map[string]int{"lava": 15, "flowing_lava": 15, "glowstone": 15, "sea_lantern": 15, "shroomlight": 15, "lit_redstone_lamp": 15, "lantern": 15, "soul_lantern": 10, "torch": 14, "redstone_torch": 7, "soul_torch": 10, "fire": 15, "soul_fire": 10, "lit_pumpkin": 15, "portal": 11, "end_portal": 15, "end_gateway": 15, "beacon": 15, "end_rod": 14, "magma": 3, "crying_obsidian": 10, "ochre_froglight": 15, "verdant_froglight": 15, "pearlescent_froglight": 15, "lit_furnace": 13, "lit_blast_furnace": 13, "lit_smoker": 13, "lit_redstone_ore": 9, "lit_deepslate_redstone_ore": 9, "amethyst_cluster": 5, "large_amethyst_bud": 4, "medium_amethyst_bud": 2, "small_amethyst_bud": 1, "brown_mushroom": 1, "brewing_stand": 1, "ender_chest": 7, "enchanting_table": 7, "conduit": 15, "campfire": 15, "soul_campfire": 10}

type SceneLight struct {
	Position [3]float32 `json:"position"`
	Level    uint8      `json:"level"`
	Color    [3]float32 `json:"color"`
}

func (s *Store) emissionLevel(rid uint32, name string) uint8 {
	stateName, states, found := s.registry.RuntimeIDToState(rid)
	if !found || stateName == "" {
		return 0
	}
	level := int(s.registry.LightBlock(rid))
	if strings.HasPrefix(name, "minecraft:") {
		n := strings.TrimPrefix(name, "minecraft:")
		if fallback, ok := vanillaEmission[n]; ok && level == 0 {
			level = fallback
		}
		if n == "unlit_redstone_torch" || n == "redstone_lamp" {
			level = 0
		}
		if n == "campfire" || n == "soul_campfire" {
			if flag(states["extinguished"]) {
				level = 0
			}
		}
		if strings.HasPrefix(n, "light_block_") {
			if v, err := strconv.Atoi(strings.TrimPrefix(n, "light_block_")); err == nil {
				level = v
			}
		}
		if n == "light_block" {
			level = number(states["block_light_level"])
		}
	}
	if d, ok := s.catalog.definitions[strings.TrimPrefix(name, "minecraft:")]; ok {
		value, has := d.Components["minecraft:light_emission"]
		for _, p := range d.Permutations {
			if matches, supported := blockCondition(p.Condition, states); supported && matches {
				if v, ok := p.Components["minecraft:light_emission"]; ok {
					value = v
					has = true
				}
			}
		}
		if has {
			if v, ok := value.(map[string]any); ok {
				value = v["emission"]
			}
			level = number(value)
		}
	}
	if level < 0 {
		return 0
	}
	if level > 15 {
		return 15
	}
	return uint8(level)
}

func (c *Catalog) emissionColor(name string) [3]float32 {
	t := c.texture(name, 2)
	if t.missing {
		if strings.Contains(name, "soul") {
			return [3]float32{.25, .8, 1}
		}
		if strings.Contains(name, "sea_lantern") {
			return [3]float32{.7, 1, .95}
		}
		return [3]float32{1, .63, .28}
	}
	var sum [3]float64
	var weight float64
	for y := 0; y < t.h; y++ {
		for x := 0; x < t.w; x++ {
			p := c.atlas.NRGBAAt(t.x+x, t.y+y)
			r, g, b := float64(p.R)/255, float64(p.G)/255, float64(p.B)/255
			lum := .2126*r + .7152*g + .0722*b
			w := float64(p.A) / 255 * lum * lum
			weight += w
			sum[0] += r * w
			sum[1] += g * w
			sum[2] += b * w
		}
	}
	if weight < .0001 {
		return [3]float32{1, .63, .28}
	}
	max := sum[0]
	if sum[1] > max {
		max = sum[1]
	}
	if sum[2] > max {
		max = sum[2]
	}
	return [3]float32{float32(sum[0] / max), float32(sum[1] / max), float32(sum[2] / max)}
}
