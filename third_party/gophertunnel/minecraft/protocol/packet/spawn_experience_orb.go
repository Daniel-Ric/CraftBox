package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type SpawnExperienceOrb struct {
	Position mgl32.Vec3

	ExperienceAmount int32
}

func (*SpawnExperienceOrb) ID() uint32 {
	return IDSpawnExperienceOrb
}

func (pk *SpawnExperienceOrb) Marshal(io protocol.IO) {
	io.Vec3(&pk.Position)
	io.Varint32(&pk.ExperienceAmount)
}
