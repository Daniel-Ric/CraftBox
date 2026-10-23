package block

type FroglightType struct {
	froglight
}

type froglight uint8

func Pearlescent() FroglightType {
	return FroglightType{0}
}

func Verdant() FroglightType {
	return FroglightType{1}
}

func Ochre() FroglightType {
	return FroglightType{2}
}

func (f froglight) Uint8() uint8 {
	return uint8(f)
}

func (f froglight) Name() string {
	switch f {
	case 0:
		return "Pearlescent Froglight"
	case 1:
		return "Verdant Froglight"
	case 2:
		return "Ochre Froglight"
	}
	panic("unknown froglight type")
}

func (f froglight) String() string {
	switch f {
	case 0:
		return "pearlescent"
	case 1:
		return "verdant"
	case 2:
		return "ochre"
	}
	panic("unknown froglight type")
}

func FroglightTypes() []FroglightType {
	return []FroglightType{Pearlescent(), Verdant(), Ochre()}
}
