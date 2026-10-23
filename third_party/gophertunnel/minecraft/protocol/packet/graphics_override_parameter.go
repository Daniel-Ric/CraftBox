package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type GraphicsOverrideParameter struct {
	Values []protocol.ParameterKeyframeValue

	FloatValue protocol.Optional[float32]

	Vec3Value protocol.Optional[mgl32.Vec3]

	BiomeIdentifier string

	PlayerIdentifier protocol.Optional[string]

	ParameterType uint8

	Reset bool
}

func (*GraphicsOverrideParameter) ID() uint32 {
	return IDGraphicsOverrideParameter
}

func (pk *GraphicsOverrideParameter) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Values)
	protocol.OptionalFunc(io, &pk.FloatValue, io.Float32)
	protocol.OptionalFunc(io, &pk.Vec3Value, io.Vec3)
	io.String(&pk.BiomeIdentifier)
	protocol.OptionalFunc(io, &pk.PlayerIdentifier, io.String)
	io.Uint8(&pk.ParameterType)
	io.Bool(&pk.Reset)
}
