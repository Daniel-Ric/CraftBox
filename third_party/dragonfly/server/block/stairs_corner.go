package block

type StairsCorner struct {
	stairsCorner
}

func NoStairsCorner() StairsCorner {
	return StairsCorner{0}
}

func InnerLeftStairsCorner() StairsCorner {
	return StairsCorner{1}
}

func InnerRightStairsCorner() StairsCorner {
	return StairsCorner{2}
}

func OuterLeftStairsCorner() StairsCorner {
	return StairsCorner{3}
}

func OuterRightStairsCorner() StairsCorner {
	return StairsCorner{4}
}

func StairsCorners() []StairsCorner {
	return []StairsCorner{NoStairsCorner(), InnerLeftStairsCorner(), InnerRightStairsCorner(), OuterLeftStairsCorner(), OuterRightStairsCorner()}
}

type stairsCorner uint8

func (s stairsCorner) Uint8() uint8 {
	return uint8(s)
}

func (s stairsCorner) Inner() bool {
	return s == 1 || s == 2
}

func (s stairsCorner) Left() bool {
	return s == 1 || s == 3
}

func (s stairsCorner) String() string {
	switch s {
	case 0:
		return "none"
	case 1:
		return "inner_left"
	case 2:
		return "inner_right"
	case 3:
		return "outer_left"
	case 4:
		return "outer_right"
	}
	panic("should never happen")
}
