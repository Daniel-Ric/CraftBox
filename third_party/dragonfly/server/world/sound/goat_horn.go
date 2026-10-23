package sound

type Horn struct {
	goatHornType
}

func Ponder() Horn {
	return Horn{0}
}

func Sing() Horn {
	return Horn{1}
}

func Seek() Horn {
	return Horn{2}
}

func Feel() Horn {
	return Horn{3}
}

func Admire() Horn {
	return Horn{4}
}

func Call() Horn {
	return Horn{5}
}

func Yearn() Horn {
	return Horn{6}
}

func Dream() Horn {
	return Horn{7}
}

type goatHornType uint8

func (g goatHornType) Uint8() uint8 {
	return uint8(g)
}

func (g goatHornType) Name() string {
	switch g {
	case 0:
		return "Ponder"
	case 1:
		return "Sing"
	case 2:
		return "Seek"
	case 3:
		return "Feel"
	case 4:
		return "Admire"
	case 5:
		return "Call"
	case 6:
		return "Yearn"
	case 7:
		return "Dream"
	}
	panic("should never happen")
}

func GoatHorns() []Horn {
	return []Horn{Ponder(), Sing(), Seek(), Feel(), Admire(), Call(), Yearn(), Dream()}
}
