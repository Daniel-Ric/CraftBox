package scene

func fenceGateModel(name string, props map[string]any) blockModel {
	b := func(x0, y0, z0, x1, y1, z1 float32) box {
		return box{[3]float32{x0 / 16, y0 / 16, z0 / 16}, [3]float32{x1 / 16, y1 / 16, z1 / 16}}
	}
	boxes := []box{b(0, 5, 7, 2, 16, 9), b(14, 5, 7, 16, 16, 9)}
	if flag(props["open_bit"]) {
		for _, x := range []float32{1, 13} {
			boxes = append(boxes, b(x, 6, 2, x+2, 9, 8), b(x, 12, 2, x+2, 15, 8), b(x, 9, 2, x+2, 12, 4))
		}
	} else {
		boxes = append(boxes, b(2, 6, 7, 14, 9, 9), b(2, 12, 7, 14, 15, 9), b(6, 9, 7, 8, 12, 9), b(8, 9, 7, 10, 12, 9))
	}
	turns := 0
	switch props["minecraft:cardinal_direction"] {
	case "west":
		turns = 1
	case "south":
		turns = 2
	case "east":
		turns = 3
	}
	if _, ok := props["minecraft:cardinal_direction"]; !ok {
		turns = (number(props["direction"]) + 2) & 3
	}
	for i, box := range boxes {
		for n := 0; n < turns; n++ {
			box = rotateBoxY(box)
		}
		if flag(props["in_wall_bit"]) {
			box.min[1] -= 3.0 / 16
			box.max[1] -= 3.0 / 16
		}
		boxes[i] = box
	}
	axis := 2
	if turns%2 == 1 {
		axis = 0
	}
	return blockModel{name: name, mode: "opaque", boxes: boxes, gateAxis: axis}
}
func rotateBoxY(b box) box {
	return box{[3]float32{b.min[2], b.min[1], 1 - b.max[0]}, [3]float32{b.max[2], b.max[1], 1 - b.min[0]}}
}
