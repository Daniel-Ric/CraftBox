package packet

import (
	"image/color"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundMapItemData struct {
	MapID int64

	Dimension byte

	LockedMap bool

	Origin protocol.BlockPos

	Scale protocol.Optional[byte]

	MapsIncludedIn protocol.Optional[[]int64]

	TrackedObjects protocol.Optional[[]protocol.MapTrackedObject]

	Decorations protocol.Optional[[]protocol.MapDecoration]

	Height protocol.Optional[int32]

	Width protocol.Optional[int32]

	XOffset protocol.Optional[int32]

	YOffset protocol.Optional[int32]

	Pixels protocol.Optional[[]color.RGBA]
}

func (*ClientBoundMapItemData) ID() uint32 {
	return IDClientBoundMapItemData
}

func (pk *ClientBoundMapItemData) Marshal(io protocol.IO) {
	io.Varint64(&pk.MapID)
	io.Uint8(&pk.Dimension)
	io.Bool(&pk.LockedMap)
	io.BlockPos(&pk.Origin)

	protocol.OptionalFunc(io, &pk.MapsIncludedIn, func(x *[]int64) {
		protocol.FuncSlice(io, x, io.Varint64)
	})
	protocol.OptionalFunc(io, &pk.Scale, io.Uint8)
	protocol.OptionalFunc(io, &pk.TrackedObjects, func(x *[]protocol.MapTrackedObject) {
		protocol.Slice(io, x)
	})
	protocol.OptionalFunc(io, &pk.Decorations, func(x *[]protocol.MapDecoration) {
		protocol.Slice(io, x)
	})
	protocol.OptionalFunc(io, &pk.Width, io.Varint32)
	protocol.OptionalFunc(io, &pk.Height, io.Varint32)
	protocol.OptionalFunc(io, &pk.XOffset, io.Varint32)
	protocol.OptionalFunc(io, &pk.YOffset, io.Varint32)
	protocol.OptionalFunc(io, &pk.Pixels, func(x *[]color.RGBA) {
		protocol.FuncSlice(io, x, io.BEARGB)
	})
}
