package protocol

const (
	SoundDataUpdateStop = iota
	SoundDataUpdateSetVolume
	SoundDataUpdateSetPitch
	SoundDataUpdateFade
	SoundDataUpdateSeekTo
	SoundDataUpdatePause
	SoundDataUpdateResume
)

type SoundDataUpdate struct {
	Type uint8

	Volume float32

	Pitch float32

	Duration float32

	TargetVolume float32

	Seconds float32
}

func (x *SoundDataUpdate) Marshal(io IO) {
	index := uint32(x.Type)
	io.Varuint32(&index)
	x.Type = uint8(index)
	io.Uint8(&x.Type)
	switch x.Type {
	case SoundDataUpdateStop, SoundDataUpdatePause, SoundDataUpdateResume:
	case SoundDataUpdateSetVolume:
		io.Float32(&x.Volume)
	case SoundDataUpdateSetPitch:
		io.Float32(&x.Pitch)
	case SoundDataUpdateFade:
		io.Float32(&x.Duration)
		io.Float32(&x.TargetVolume)
	case SoundDataUpdateSeekTo:
		io.Float32(&x.Seconds)
	default:
		io.UnknownEnumOption(x.Type, "sound data update type")
	}
}
