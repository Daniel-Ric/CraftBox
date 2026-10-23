package category

type Category struct {
	group    string
	category uint8
}

func Construction() Category {
	return Category{category: 1}
}

func Nature() Category {
	return Category{category: 2}
}

func Equipment() Category {
	return Category{category: 3}
}

func Items() Category {
	return Category{category: 4}
}

func (c Category) Uint8() uint8 {
	return c.category
}

func (c Category) WithGroup(group string) Category {
	c.group = group
	return c
}

func (c Category) String() string {
	switch c.category {
	case 1:
		return "construction"
	case 2:
		return "nature"
	case 3:
		return "equipment"
	case 4:
		return "items"
	}
	panic("should never happen")
}

func (c Category) Group() string {
	if len(c.group) > 0 {
		return "itemGroup.name." + c.group
	}
	return ""
}
