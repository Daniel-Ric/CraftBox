package item

type OptionalColour uint8

var colours = Colours()

func NewOptionalColour(c Colour) OptionalColour {
	return OptionalColour(c.Uint8() + 1)
}

func (oc OptionalColour) Colour() (Colour, bool) {
	if oc == 0 {
		return Colour{}, false
	}
	return colours[(oc - 1)], true
}

func (oc OptionalColour) Uint8() uint8 {
	return uint8(oc)
}

func (oc OptionalColour) Prepend(str string) string {
	if oc != 0 {
		return colours[(oc-1)].String() + "_" + str
	}
	return str
}

func OptionalColours() []OptionalColour {
	optionalColours := make([]OptionalColour, 17)
	for i, c := range colours {
		optionalColours[i+1] = NewOptionalColour(c)
	}
	return optionalColours
}
