package block

type GrindstoneAttachment struct {
	grindstoneAttachment
}

func StandingGrindstoneAttachment() GrindstoneAttachment {
	return GrindstoneAttachment{0}
}

func HangingGrindstoneAttachment() GrindstoneAttachment {
	return GrindstoneAttachment{1}
}

func WallGrindstoneAttachment() GrindstoneAttachment {
	return GrindstoneAttachment{2}
}

func GrindstoneAttachments() []GrindstoneAttachment {
	return []GrindstoneAttachment{StandingGrindstoneAttachment(), HangingGrindstoneAttachment(), WallGrindstoneAttachment()}
}

type grindstoneAttachment uint8

func (g grindstoneAttachment) Uint8() uint8 {
	return uint8(g)
}

func (g grindstoneAttachment) String() string {
	switch g {
	case 0:
		return "standing"
	case 1:
		return "hanging"
	case 2:
		return "side"
	}
	panic("should never happen")
}
