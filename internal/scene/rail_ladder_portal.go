package scene

func (c *Catalog) railModel(name string, props map[string]any) blockModel {
	direction := number(props["rail_direction"])
	if direction < 0 || direction > 9 {
		direction = 0
	}
	normal := name == "minecraft:rail"
	if !normal && direction > 5 {
		direction = 0
	}
	face := 3
	if normal && direction >= 6 || !normal && flag(props["rail_data_bit"]) {
		face = 2
	}
	tex := c.texture(name, face)
	points := [4][3]float32{{0, 1.0 / 16, 1}, {1, 1.0 / 16, 1}, {1, 1.0 / 16, 0}, {0, 1.0 / 16, 0}}
	turns := 0
	switch direction {
	case 1, 2, 3, 9:
		turns = 1
	case 8:
		turns = 2
	case 7:
		turns = 3
	}
	for i, p := range points {
		for turn := 0; turn < turns; turn++ {
			p[0], p[2] = p[2], 1-p[0]
		}
		switch direction {
		case 2:
			p[1] += p[0]
		case 3:
			p[1] += 1 - p[0]
		case 4:
			p[1] += 1 - p[2]
		case 5:
			p[1] += p[2]
		}
		points[i] = p
	}
	return blockModel{name: name, mode: "mask", quads: []modelQuad{texturedQuad(points, tex)}}
}

func (c *Catalog) ladderModel(name string, props map[string]any) blockModel {

	face := 5
	switch number(props["facing_direction"]) {
	case 3:
		face = 4
	case 4:
		face = 1
	case 5:
		face = 0
	}
	for direction, index := range map[string]int{"north": 5, "south": 4, "west": 1, "east": 0} {
		if props["minecraft:cardinal_direction"] == direction {
			face = index
		}
	}
	var points [4][3]float32
	for i, p := range corners[face] {
		for axis := range p {
			points[i][axis] = float32(p[axis])
		}
		axis := face / 2
		points[i][axis] = 15.0 / 16
		if face%2 == 0 {
			points[i][axis] = 1.0 / 16
		}
	}
	return blockModel{name: name, mode: "mask", quads: []modelQuad{texturedQuad(points, c.texture(name, face))}}
}

func portalModel(name string, props map[string]any) blockModel {
	b := unit
	axis := 2
	if props["portal_axis"] == "z" {
		axis = 0
	}
	b.min[axis], b.max[axis] = 6.0/16, 10.0/16
	return blockModel{name: name, mode: "blend", boxes: []box{b}}
}
