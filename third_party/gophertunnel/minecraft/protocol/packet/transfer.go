package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Transfer struct {
	Address string

	Port uint16

	ReloadWorld bool

	GatheringJoinInfo protocol.Optional[protocol.GatheringJoinInfo]
}

func (*Transfer) ID() uint32 {
	return IDTransfer
}

func (pk *Transfer) Marshal(io protocol.IO) {
	io.String(&pk.Address)
	io.Uint16(&pk.Port)
	io.Bool(&pk.ReloadWorld)
	protocol.OptionalMarshaler(io, &pk.GatheringJoinInfo)
}
