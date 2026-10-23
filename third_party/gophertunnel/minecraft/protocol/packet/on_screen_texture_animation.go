package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type OnScreenTextureAnimation struct {
	AnimationType uint32
}

func (*OnScreenTextureAnimation) ID() uint32 {
	return IDOnScreenTextureAnimation
}

func (pk *OnScreenTextureAnimation) Marshal(io protocol.IO) {
	io.Uint32(&pk.AnimationType)
}
