package item

type WrittenBookGeneration struct {
	generation
}

type generation uint8

func OriginalGeneration() WrittenBookGeneration {
	return WrittenBookGeneration{0}
}

func CopyGeneration() WrittenBookGeneration {
	return WrittenBookGeneration{1}
}

func CopyOfCopyGeneration() WrittenBookGeneration {
	return WrittenBookGeneration{2}
}

func (g generation) Uint8() uint8 {
	return uint8(g)
}

func (g generation) String() string {
	switch g {
	case 0:
		return "original"
	case 1:
		return "copy of original"
	case 2:
		return "copy of copy"
	}
	panic("unknown written book generation")
}
