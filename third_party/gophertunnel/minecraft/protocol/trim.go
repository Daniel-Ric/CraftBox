package protocol

type TrimPattern struct {
	ItemName string

	PatternID string
}

func (x *TrimPattern) Marshal(r IO) {
	r.String(&x.ItemName)
	r.String(&x.PatternID)
}

type TrimMaterial struct {
	MaterialID string

	Colour string

	ItemName string
}

func (x *TrimMaterial) Marshal(r IO) {
	r.String(&x.MaterialID)
	r.String(&x.Colour)
	r.String(&x.ItemName)
}
