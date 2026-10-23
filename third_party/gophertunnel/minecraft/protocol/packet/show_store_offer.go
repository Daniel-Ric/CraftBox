package packet

import (
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	StoreOfferTypeMarketplace = iota
	StoreOfferTypeDressingRoom
	StoreOfferTypeServerPage
)

type ShowStoreOffer struct {
	OfferID uuid.UUID

	Type byte
}

func (*ShowStoreOffer) ID() uint32 {
	return IDShowStoreOffer
}

func (pk *ShowStoreOffer) Marshal(io protocol.IO) {
	io.UUID(&pk.OfferID)
	io.Uint8(&pk.Type)
}
