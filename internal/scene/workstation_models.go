package scene

func appendFaceTexturedBox(m *blockModel, b box, textures [6]tile) {
	for face, tex := range textures {
		part := blockModel{mode: m.mode}
		appendModelBox(&part, b, tex)
		m.quads = append(m.quads, part.quads[face])
	}
}
func (c *Catalog) openBookModel() blockModel {
	m := blockModel{mode: "mask"}
	tex := c.materialTexture(map[string]any{"texture": "textures/entity/enchanting_table_book"}, 64, 32)
	for side := 0; side < 2; side++ {
		half := blockModel{mode: "mask"}
		lo, hi := float32(-.375), float32(0)
		coverU, pageU := float32(0), float32(0)
		angle := float32(-20)
		if side == 1 {
			lo, hi = 0, .375
			coverU = 16
			pageU = 12
			angle = 20
		}
		cover := texturedQuad([4][3]float32{{lo, 0, .3125}, {hi, 0, .3125}, {hi, 0, -.3125}, {lo, 0, -.3125}}, tex)
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			cover.uv[i] = [2]float32{(float32(tex.x) + (coverU+uv[0]*6)/64*float32(tex.w)) / atlasSize, (float32(tex.y) + uv[1]*10/32*float32(tex.h)) / atlasSize}
		}
		half.quads = append(half.quads, cover)
		pageLo, pageHi := lo+.03125, hi-.03125
		appendSkinBox(&half, box{[3]float32{pageLo, .01, -.25}, [3]float32{pageHi, .0725, .25}}, tex, [2]float32{pageU, 10}, 64, 32)
		top := &half.quads[3]
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			top.uv[i] = [2]float32{(float32(tex.x) + (pageU+uv[0]*5)/64*float32(tex.w)) / atlasSize, (float32(tex.y) + (10+uv[1]*8)/32*float32(tex.h)) / atlasSize}
		}
		transformQuads(&half, rotationMatrix([3]float32{}, [3]float32{0, 0, angle}))
		m.quads = append(m.quads, half.quads...)
	}
	return m
}
func placeModel(m *blockModel, offset [3]float32) {
	for qi := range m.quads {
		for i := range m.quads[qi].points {
			for a := range offset {
				m.quads[qi].points[i][a] += offset[a]
			}
		}
	}
}
func orientHorizontalModel(m *blockModel, props map[string]any) {
	copy := map[string]any{}
	for k, v := range props {
		copy[k] = v
	}
	if _, ok := copy["minecraft:cardinal_direction"]; !ok {
		switch number(props["direction"]) {
		case 0:
			copy["minecraft:cardinal_direction"] = "south"
		case 1:
			copy["minecraft:cardinal_direction"] = "west"
		case 2:
			copy["minecraft:cardinal_direction"] = "north"
		case 3:
			copy["minecraft:cardinal_direction"] = "east"
		}
	}
	rotateFacing(m, copy)
}
func lecternHasBook(data map[string]any) bool {
	if value, explicit := data["hasBook"]; explicit {
		return flag(value)
	}
	item, ok := data["book"].(map[string]any)
	if !ok {
		return false
	}
	name, _ := item["Name"].(string)
	present := name != "" && name != "minecraft:air"
	if count, ok := item["Count"]; ok {
		present = present && number(count) > 0
	}
	return present
}
func (c *Catalog) lecternModel(props, data map[string]any) blockModel {
	m := blockModel{name: "minecraft:lectern", mode: "opaque", entityVisual: "lectern"}
	wood := c.materialTexture(map[string]any{"texture": "lectern_bottom"}, 16, 16)
	appendModelBox(&m, box{[3]float32{}, [3]float32{1, .125, 1}}, wood)
	front, side := c.texture(m.name, 5), c.texture(m.name, 4)
	appendFaceTexturedBox(&m, box{[3]float32{.25, .125, .25}, [3]float32{.75, .75, .75}}, [6]tile{side, side, wood, wood, side, front})
	desk := blockModel{mode: "opaque"}
	top := c.texture(m.name, 2)
	appendFaceTexturedBox(&desk, box{[3]float32{0, .625, 0}, [3]float32{1, .875, 1}}, [6]tile{side, side, top, wood, side, front})
	transformQuads(&desk, rotationMatrix([3]float32{.5, .75, .5}, [3]float32{-22.5, 0, 0}))
	m.quads = append(m.quads, desk.quads...)
	if lecternHasBook(data) {
		book := c.openBookModel()
		placeModel(&book, [3]float32{.5, .9, .5})
		transformQuads(&book, rotationMatrix([3]float32{.5, .75, .5}, [3]float32{-22.5, 0, 0}))
		m.quads = append(m.quads, book.quads...)
	}
	orientHorizontalModel(&m, props)
	return m
}
func (c *Catalog) enchantingTableModel() blockModel {
	m := blockModel{name: "minecraft:enchanting_table", mode: "opaque"}
	var textures [6]tile
	for i := range textures {
		textures[i] = c.texture(m.name, i)
	}
	appendFaceTexturedBox(&m, box{[3]float32{}, [3]float32{1, .75, 1}}, textures)
	book := c.openBookModel()
	transformQuads(&book, rotationMatrix([3]float32{}, [3]float32{-12, 25, 0}))
	placeModel(&book, [3]float32{.5, 1, .5})
	m.quads = append(m.quads, book.quads...)
	return m
}
func (c *Catalog) stonecutterModel(props map[string]any) blockModel {
	m := blockModel{name: "minecraft:stonecutter_block", mode: "opaque"}
	top := c.texture(m.name, 2)
	bottom := c.texture(m.name, 3)
	side := c.texture(m.name, 5)
	appendFaceTexturedBox(&m, box{[3]float32{}, [3]float32{1, 9.0 / 16, 1}}, [6]tile{side, side, top, bottom, side, side})
	// Side sprites occupy their bottom nine rows; the saw is a separate cutout.
	saw := texturedQuad([4][3]float32{{.5, 9.0 / 16, 1}, {.5, 9.0 / 16, 0}, {.5, 1, 0}, {.5, 1, 1}}, c.texture(m.name, 0))
	for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
		saw.uv[i] = [2]float32{(float32(saw.texture.x) + uv[0]*16) / atlasSize, (float32(saw.texture.y) + 9 + uv[1]*7) / atlasSize}
	}
	m.quads = append(m.quads, saw)
	orientHorizontalModel(&m, props)
	return m
}
func (c *Catalog) dripstoneModel(props map[string]any) blockModel {
	thickness, _ := props["dripstone_thickness"].(string)
	if thickness == "" {
		thickness = "tip"
	}
	if thickness == "tip_merge" {
		thickness = "merge"
	}
	switch thickness {
	case "tip", "frustum", "middle", "base", "merge":
	default:
		thickness = "tip"
	}
	index := 1
	direction := "up"
	if flag(props["hanging"]) || props["vertical_direction"] == "down" {
		index = 0
		direction = "down"
	}
	key := "pointed_dripstone_" + thickness
	path := "textures/blocks/pointed_dripstone_" + direction + "_" + thickness
	if choices := c.terrainVariants[key]; len(choices) > index {
		path = choices[index]
	}
	tex := c.materialTexture(map[string]any{"texture": path}, 16, 16)
	m := blockModel{name: "minecraft:pointed_dripstone", mode: "mask"}
	for _, p := range [][4][3]float32{{{0, 0, 0}, {1, 0, 1}, {1, 1, 1}, {0, 1, 0}}, {{1, 0, 0}, {0, 0, 1}, {0, 1, 1}, {1, 1, 0}}} {
		m.quads = append(m.quads, texturedQuad(p, tex))
	}
	return m
}
func (c *Catalog) grindstoneModel(props map[string]any) blockModel {
	m := blockModel{name: "minecraft:grindstone", mode: "opaque"}
	wood := c.materialTexture(map[string]any{"texture": "grindstone_leg"}, 16, 16)
	pivot := c.materialTexture(map[string]any{"texture": "grindstone_pivot"}, 16, 16)
	side := c.materialTexture(map[string]any{"texture": "grindstone_side"}, 16, 16)
	round := c.materialTexture(map[string]any{"texture": "grindstone_round"}, 16, 16)
	for _, x := range []float32{2.0 / 16, 12.0 / 16} {
		appendModelBox(&m, box{[3]float32{x, 0, 6.0 / 16}, [3]float32{x + 2.0/16, 7.0 / 16, 10.0 / 16}}, wood)
	}
	appendModelBox(&m, box{[3]float32{2.0 / 16, 6.0 / 16, 5.0 / 16}, [3]float32{14.0 / 16, 12.0 / 16, 11.0 / 16}}, pivot)
	appendFaceTexturedBox(&m, box{[3]float32{4.0 / 16, 4.0 / 16, 2.0 / 16}, [3]float32{12.0 / 16, 1, 14.0 / 16}}, [6]tile{side, side, round, round, round, round})
	switch props["attachment"] {
	case "hanging":
		mountModel(&m, 0)
	case "side":
		mountModel(&m, 2)
	}
	orientHorizontalModel(&m, props)
	return m
}
