package creative

type Category struct {
	category
}

type category uint8

func ConstructionCategory() Category {
	return Category{1}
}

func NatureCategory() Category {
	return Category{2}
}

func EquipmentCategory() Category {
	return Category{3}
}

func ItemsCategory() Category {
	return Category{4}
}

func (s category) Uint8() uint8 {
	return uint8(s)
}
