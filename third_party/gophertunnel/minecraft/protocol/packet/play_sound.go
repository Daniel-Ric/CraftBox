package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type PlaySound struct {
	SoundName string

	Position mgl32.Vec3

	Volume float32

	Pitch float32

	LoopCount int32

	BypassListenerRangeCheck bool

	Handle protocol.Optional[uint64]

	PlaybackPositionSeconds protocol.Optional[float32]
}

func (*PlaySound) ID() uint32 {
	return IDPlaySound
}

func (pk *PlaySound) Marshal(io protocol.IO) {
	io.String(&pk.SoundName)
	io.SoundPos(&pk.Position)
	io.Float32(&pk.Volume)
	io.Float32(&pk.Pitch)
	io.Varint32(&pk.LoopCount)
	io.Bool(&pk.BypassListenerRangeCheck)
	protocol.OptionalFunc(io, &pk.Handle, io.Uint64)
	protocol.OptionalFunc(io, &pk.PlaybackPositionSeconds, io.Float32)
}
