package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SpawnParticleEffect struct {
	Dimension byte

	EntityUniqueID int64

	Position mgl32.Vec3

	ParticleName string

	MoLangVariables protocol.Optional[[]byte]
}

func (*SpawnParticleEffect) ID() uint32 {
	return IDSpawnParticleEffect
}

func (pk *SpawnParticleEffect) Marshal(io protocol.IO) {
	io.Uint8(&pk.Dimension)
	io.ActorUniqueID(&pk.EntityUniqueID)
	io.Vec3(&pk.Position)
	io.String(&pk.ParticleName)
	protocol.OptionalFunc(io, &pk.MoLangVariables, io.ByteSlice)
}
