package protocol

const (
	SoundDataUpdateStop uint32 = iota
	SoundDataUpdateSetVolume
	SoundDataUpdateSetPitch
	SoundDataUpdateFade
	SoundDataUpdateSeekTo
	SoundDataUpdatePause
	SoundDataUpdateResume
)

type SoundDataUpdate struct {
	Type uint32

	Volume float32

	Pitch float32

	Duration float32

	TargetVolume float32

	Seconds float32
}

func (x *SoundDataUpdate) Marshal(io IO) {
	io.Varuint32(&x.Type)
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
