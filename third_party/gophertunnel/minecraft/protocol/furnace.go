package protocol

const (
	FurnaceLayoutNone = iota
	FurnaceLayoutInventoryOnly
	FurnaceLayoutDefault
)

const (
	FurnaceLeftTabNone = iota
	FurnaceLeftTabRecipeFood
	FurnaceLeftTabRecipeItems
	FurnaceLeftTabRecipeBlocks
	FurnaceLeftTabRecipeSearch
	FurnaceLeftTabInventory
)

type FurnaceOptions struct {
	LeftFurnaceTab int32

	Filtering bool

	Layout int32
}

func (x *FurnaceOptions) Marshal(r IO) {
	r.Varint32(&x.LeftFurnaceTab)
	r.Bool(&x.Filtering)
	r.Varint32(&x.Layout)
}
