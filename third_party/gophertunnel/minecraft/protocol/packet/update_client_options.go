package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	GraphicsModeSimple = iota
	GraphicsModeFancy
	GraphicsModeAdvanced
	GraphicsModeRayTraced
)

type UpdateClientOptions struct {
	GraphicsMode protocol.Optional[byte]

	FilterProfanity protocol.Optional[bool]
}

func (*UpdateClientOptions) ID() uint32 {
	return IDUpdateClientOptions
}

func (pk *UpdateClientOptions) Marshal(io protocol.IO) {
	protocol.OptionalFunc(io, &pk.GraphicsMode, io.Uint8)
	protocol.OptionalFunc(io, &pk.FilterProfanity, io.Bool)
}
