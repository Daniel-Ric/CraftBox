package item

type FireworkShape struct {
	fireworkShape
}

func FireworkShapeSmallSphere() FireworkShape {
	return FireworkShape{0}
}

func FireworkShapeHugeSphere() FireworkShape {
	return FireworkShape{1}
}

func FireworkShapeStar() FireworkShape {
	return FireworkShape{2}
}

func FireworkShapeCreeperHead() FireworkShape {
	return FireworkShape{3}
}

func FireworkShapeBurst() FireworkShape {
	return FireworkShape{4}
}

type fireworkShape uint8

func (f fireworkShape) Uint8() uint8 {
	return uint8(f)
}

func (f fireworkShape) Name() string {
	switch f {
	case 0:
		return "Small Sphere"
	case 1:
		return "Huge Sphere"
	case 2:
		return "Star"
	case 3:
		return "Creeper Head"
	case 4:
		return "Burst"
	}
	panic("unknown firework type")
}

func (f fireworkShape) String() string {
	switch f {
	case 0:
		return "small_sphere"
	case 1:
		return "huge_sphere"
	case 2:
		return "star"
	case 3:
		return "creeper_head"
	case 4:
		return "burst"
	}
	panic("unknown firework type")
}

func FireworkShapes() []FireworkShape {
	return []FireworkShape{FireworkShapeSmallSphere(), FireworkShapeHugeSphere(), FireworkShapeStar(), FireworkShapeCreeperHead(), FireworkShapeBurst()}
}
