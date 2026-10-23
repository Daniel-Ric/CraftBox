package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type UpdateTrade struct {
	WindowID byte

	WindowType byte

	Size int32

	TradeTier int32

	VillagerUniqueID int64

	EntityUniqueID int64

	DisplayName string

	NewTradeUI bool

	DemandBasedPrices bool

	SerialisedOffers []byte
}

func (*UpdateTrade) ID() uint32 {
	return IDUpdateTrade
}

func (pk *UpdateTrade) Marshal(io protocol.IO) {
	io.Uint8(&pk.WindowID)
	io.Uint8(&pk.WindowType)
	io.Varint32(&pk.Size)
	io.Varint32(&pk.TradeTier)
	io.ActorUniqueID(&pk.VillagerUniqueID)
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.String(&pk.DisplayName)
	io.Bool(&pk.NewTradeUI)
	io.Bool(&pk.DemandBasedPrices)
	io.Bytes(&pk.SerialisedOffers)
}
