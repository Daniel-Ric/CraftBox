package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	RespawnStateSearchingForSpawn = iota
	RespawnStateReadyToSpawn
	RespawnStateClientReadyToSpawn
)

type Respawn struct {
	Position mgl32.Vec3

	State byte

	EntityRuntimeID uint64
}

func (*Respawn) ID() uint32 {
	return IDRespawn
}

func (pk *Respawn) Marshal(io protocol.IO) {
	io.Vec3(&pk.Position)
	io.Uint8(&pk.State)
	io.ActorRuntimeID(&pk.EntityRuntimeID)
}
