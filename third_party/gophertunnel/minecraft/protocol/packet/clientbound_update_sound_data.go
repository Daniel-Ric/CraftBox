package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type ClientboundUpdateSoundData struct {
	ServerSoundHandle uint64

	Stop      protocol.SoundDataUpdate
	SetVolume protocol.SoundDataUpdate
	SetPitch  protocol.SoundDataUpdate
	Fade      protocol.SoundDataUpdate
	SeekTo    protocol.SoundDataUpdate
	Pause     protocol.SoundDataUpdate
	Resume    protocol.SoundDataUpdate
}

func (*ClientboundUpdateSoundData) ID() uint32 {
	return IDClientboundUpdateSoundData
}

func (pk *ClientboundUpdateSoundData) Marshal(io protocol.IO) {
	io.Uint64(&pk.ServerSoundHandle)
	protocol.Single(io, &pk.Stop)
	protocol.Single(io, &pk.SetVolume)
	protocol.Single(io, &pk.SetPitch)
	protocol.Single(io, &pk.Fade)
	protocol.Single(io, &pk.SeekTo)
	protocol.Single(io, &pk.Pause)
	protocol.Single(io, &pk.Resume)
}
