package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type ClientboundUpdateSoundData struct {
	ServerSoundHandle uint64

	Update protocol.SoundDataUpdate
}

func (*ClientboundUpdateSoundData) ID() uint32 {
	return IDClientboundUpdateSoundData
}

func (pk *ClientboundUpdateSoundData) Marshal(io protocol.IO) {
	io.Uint64(&pk.ServerSoundHandle)
	protocol.Single(io, &pk.Update)
}
