package cube

type Range [2]int

func (r Range) Min() int {
	return r[0]
}

func (r Range) Max() int {
	return r[1]
}

func (r Range) Height() int {
	return r[1] - r[0]
}
