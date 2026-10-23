package block

type CopperType struct {
	copper
}

type copper uint8

func NormalCopper() CopperType {
	return CopperType{0}
}

func CutCopper() CopperType {
	return CopperType{1}
}

func ChiseledCopper() CopperType {
	return CopperType{2}
}

func (s copper) Uint8() uint8 {
	return uint8(s)
}

func (s copper) Name() string {
	switch s {
	case 0:
		return "Copper"
	case 1:
		return "Cut Copper"
	case 2:
		return "Chiseled Copper"
	}
	panic("unknown copper type")
}

func (s copper) String() string {
	switch s {
	case 0:
		return "default"
	case 1:
		return "cut"
	case 2:
		return "chiseled"
	}
	panic("unknown copper type")
}

func CopperTypes() []CopperType {
	return []CopperType{NormalCopper(), CutCopper(), ChiseledCopper()}
}
