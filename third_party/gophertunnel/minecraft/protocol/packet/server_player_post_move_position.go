package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerPlayerPostMovePosition struct {
	Position mgl32.Vec3
}

func (*ServerPlayerPostMovePosition) ID() uint32 {
	return IDServerPlayerPostMovePosition
}

func (pk *ServerPlayerPostMovePosition) Marshal(io protocol.IO) {
	io.Vec3(&pk.Position)
}
