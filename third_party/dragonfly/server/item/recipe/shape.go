package recipe

type Shape [2]int

func (s Shape) Width() int {
	return s[0]
}

func (s Shape) Height() int {
	return s[1]
}

func NewShape(width, height int) Shape {
	return Shape{width, height}
}
