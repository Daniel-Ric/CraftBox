package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ResourcePacksReadyForValidation struct{}

func (*ResourcePacksReadyForValidation) ID() uint32 {
	return IDResourcePacksReadyForValidation
}

func (pk *ResourcePacksReadyForValidation) Marshal(protocol.IO) {}
