package scene

import "strings"

func (c *Catalog) orientDirectionalCube(m *blockModel, props map[string]any) {
	if len(m.quads) != 0 || m.cross || m.connect != "" || len(m.boxes) != 1 || m.boxes[0] != unit {
		return
	}
	name := strings.TrimPrefix(m.name, "minecraft:")
	switch name {
	case "barrel", "furnace", "lit_furnace", "smoker", "lit_smoker", "blast_furnace", "lit_blast_furnace":
	default:
		return
	}
	var textures [6]tile
	for face := range directions {
		textures[face] = c.texture(m.name, face)
	}
	if name == "barrel" {
		top := c.texture(m.name, 5)
		if flag(props["open_bit"]) {
			key := c.blocks[name]["north"]
			if paths := c.terrainVariants[key]; len(paths) > 1 && paths[1] != "" {
				top = c.materialTexture(map[string]any{"texture": paths[1]}, 16, 16)
			}
		}
		textures = [6]tile{textures[2], textures[2], top, textures[4], textures[2], textures[2]}
		facing := number(props["facing_direction"])
		texturedCubeSurface(m, textures, func(p [3]float32, point bool) [3]float32 {
			offset := float32(0)
			if point {
				offset = 1
			}
			switch facing {
			case 0:
				return [3]float32{p[0], offset - p[1], offset - p[2]}
			case 2:
				return [3]float32{p[0], p[2], offset - p[1]}
			case 3:
				return [3]float32{p[0], offset - p[2], p[1]}
			case 4:
				return [3]float32{offset - p[1], p[0], p[2]}
			case 5:
				return [3]float32{p[1], offset - p[0], p[2]}
			}
			return p
		})
		return
	}
	frontSlot := 4
	if strings.HasPrefix(name, "lit_") {
		frontSlot = 0
	}
	textures = [6]tile{textures[5], textures[5], textures[2], textures[3], textures[5], textures[frontSlot]}
	turns := 0
	switch props["minecraft:cardinal_direction"] {
	case "west":
		turns = 1
	case "south":
		turns = 2
	case "east":
		turns = 3
	}
	texturedCubeSurface(m, textures, func(p [3]float32, point bool) [3]float32 {
		offset := float32(0)
		if point {
			offset = 1
		}
		for i := 0; i < turns; i++ {
			p = [3]float32{p[2], p[1], offset - p[0]}
		}
		return p
	})
}
