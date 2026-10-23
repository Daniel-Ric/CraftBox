package protocol

const (
	CreativeCategoryAll = iota
	CreativeCategoryConstruction
	CreativeCategoryNature
	CreativeCategoryEquipment
	CreativeCategoryItems
	CreativeCategoryItemCommandOnly
	CreativeCategoryUndefined
)

type CreativeGroup struct {
	Category byte

	Name string

	Icon ItemStack
}

func (x *CreativeGroup) Marshal(r IO) {
	r.Uint8(&x.Category)
	r.String(&x.Name)
	r.Item(&x.Icon)
}

type CreativeItem struct {
	CreativeItemNetworkID uint32

	Item ItemStack

	GroupIndex uint32
}

func (x *CreativeItem) Marshal(r IO) {
	r.Varuint32(&x.CreativeItemNetworkID)
	r.Item(&x.Item)
	r.Varuint32(&x.GroupIndex)
}
