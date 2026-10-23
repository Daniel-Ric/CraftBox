package protocol

import "github.com/sandertv/gophertunnel/minecraft/nbt"

const (
	ItemEntryVersionLegacy = iota
	ItemEntryVersionDataDriven
	ItemEntryVersionNone
)

type ItemInstance struct {
	StackNetworkID int32

	Stack ItemStack
}

type ItemStack struct {
	ItemType

	BlockRuntimeID int32

	Count uint16

	NBTData map[string]any

	CanBePlacedOn []string

	CanBreak []string

	BlockingTick int64
}

type StackRequestItem struct {
	Identifier string

	MetadataValue uint32

	BlockRuntimeID int32

	Count uint16

	NBTData map[string]any

	CanBePlacedOn []string

	CanBreak []string

	BlockingTick int64
}

type itemUserData struct {
	nbtData       map[string]any
	canBePlacedOn []string
	canBreak      []string
	blockingTick  int64
}

func itemStackUserData(x *ItemStack) itemUserData {
	return itemUserData{
		nbtData:       x.NBTData,
		canBePlacedOn: x.CanBePlacedOn,
		canBreak:      x.CanBreak,
		blockingTick:  x.BlockingTick,
	}
}

func stackRequestItemUserData(x *StackRequestItem) itemUserData {
	return itemUserData{
		nbtData:       x.NBTData,
		canBePlacedOn: x.CanBePlacedOn,
		canBreak:      x.CanBreak,
		blockingTick:  x.BlockingTick,
	}
}

type ItemType struct {
	NetworkID int32

	MetadataValue uint32
}

type ItemEntry struct {
	Name string

	RuntimeID int16

	ComponentBased bool

	Version int32

	Data map[string]any
}

func (x *ItemEntry) Marshal(r IO) {
	r.String(&x.Name)
	r.Int16(&x.RuntimeID)
	r.Bool(&x.ComponentBased)
	r.Varint32(&x.Version)
	r.NBT(&x.Data, nbt.NetworkLittleEndian)
}

type MaterialReducerOutput struct {
	NetworkID int32

	Count int32
}

func (x *MaterialReducerOutput) Marshal(r IO) {
	r.Varint32(&x.NetworkID)
	r.Varint32(&x.Count)
}

type MaterialReducer struct {
	InputItem ItemType

	Outputs []MaterialReducerOutput
}
