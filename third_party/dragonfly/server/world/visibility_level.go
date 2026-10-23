package world

type VisibilityLevel struct {
	visibility
}

func PublicVisibility() VisibilityLevel {
	return VisibilityLevel{0}
}

func EnforceInvisible() VisibilityLevel {
	return VisibilityLevel{1}
}

func EnforceVisible() VisibilityLevel {
	return VisibilityLevel{2}
}

type visibility uint8

func (v visibility) EnforceVisibility() bool {
	return v > 0
}
