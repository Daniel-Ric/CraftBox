package packet

import (
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Unknown struct {
	PacketID uint32

	Payload []byte
}

func (pk *Unknown) ID() uint32 {
	return pk.PacketID
}

func (pk *Unknown) Marshal(io protocol.IO) {
	io.Bytes(&pk.Payload)
}

func (pk *Unknown) String() string {
	return fmt.Sprintf("{ID:0x%x Payload:0x%x}", pk.PacketID, pk.Payload)
}
