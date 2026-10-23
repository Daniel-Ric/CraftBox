package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type StopSound struct {
	SoundName string

	StopAll bool

	StopMusicLegacy bool
}

func (*StopSound) ID() uint32 {
	return IDStopSound
}

func (pk *StopSound) Marshal(io protocol.IO) {
	io.String(&pk.SoundName)
	io.Bool(&pk.StopAll)
	io.Bool(&pk.StopMusicLegacy)
}
