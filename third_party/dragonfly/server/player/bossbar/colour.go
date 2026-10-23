package bossbar

type Colour struct{ colour }

func Pink() Colour {
	return Colour{colour(0)}
}

func Blue() Colour {
	return Colour{colour(1)}
}

func Red() Colour {
	return Colour{colour(2)}
}

func Green() Colour {
	return Colour{colour(3)}
}

func Yellow() Colour {
	return Colour{colour(4)}
}

func Purple() Colour {
	return Colour{colour(5)}
}

func RebeccaPurple() Colour {
	return Colour{colour(6)}
}

func White() Colour {
	return Colour{colour(7)}
}

type colour uint8

func (c colour) Uint8() uint8 {
	return uint8(c)
}
