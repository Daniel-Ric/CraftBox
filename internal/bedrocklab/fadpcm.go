package bedrocklab

import (
	"encoding/binary"
	"fmt"
)

func decodeFADPCM(encoded []byte, frames uint64, channels int) ([]byte, error) {
	if channels < 1 || channels > 8 || frames > 8<<20 || frames*uint64(channels)*2 > 16<<20 {
		return nil, fmt.Errorf("audio preview exceeds supported duration or channel limits")
	}
	blocks := (frames + 255) / 256
	required := blocks * uint64(channels) * 140
	if required > uint64(len(encoded)) {
		return nil, fmt.Errorf("truncated FADPCM frames")
	}
	pcm := make([]byte, int(frames)*channels*2)
	coefficients := [7][2]int32{{0, 0}, {60, 0}, {122, 60}, {115, 52}, {98, 55}, {0, 0}, {0, 0}}
	for block := uint64(0); block < blocks; block++ {
		for channel := 0; channel < channels; channel++ {
			offset := (block*uint64(channels) + uint64(channel)) * 140
			frame := encoded[offset : offset+140]
			predictors := binary.LittleEndian.Uint32(frame)
			scales := binary.LittleEndian.Uint32(frame[4:])
			previous := int32(int16(binary.LittleEndian.Uint16(frame[8:])))
			older := int32(int16(binary.LittleEndian.Uint16(frame[10:])))
			for group := 0; group < 8; group++ {
				predictor := (predictors >> uint(group*4) & 15) % 7
				scale := uint(22 - (scales >> uint(group*4) & 15))
				for word := 0; word < 4; word++ {
					packed := binary.LittleEndian.Uint32(frame[12+group*16+word*4:])
					for nibble := 0; nibble < 8; nibble++ {
						signed := int32((packed>>uint(nibble*4)&15)<<28) >> scale
						value := (signed - older*coefficients[predictor][1] + previous*coefficients[predictor][0]) >> 6
						if value > 32767 {
							value = 32767
						}
						if value < -32768 {
							value = -32768
						}
						older, previous = previous, value
						sample := block*256 + uint64(group*32+word*8+nibble)
						if sample < frames {
							index := (int(sample)*channels + channel) * 2
							binary.LittleEndian.PutUint16(pcm[index:], uint16(int16(value)))
						}
					}
				}
			}
		}
	}
	return pcm, nil
}

func previewFSBAudio(b []byte, info *BinaryInfo) ([]byte, []float64, error) {
	if info == nil || info.Format != "FMOD FSB5" || info.Issue != "" || info.Limited || len(info.Samples) == 0 {
		return nil, nil, fmt.Errorf("audio bank metadata is incomplete")
	}
	sample := &info.Samples[0]
	codec := binary.LittleEndian.Uint32(b[24:])
	base := uint64(60)
	if binary.LittleEndian.Uint32(b[4:]) == 0 {
		base = 64
	}
	start := base + uint64(binary.LittleEndian.Uint32(b[12:])) + uint64(binary.LittleEndian.Uint32(b[16:])) + sample.DataOffset
	end := start + sample.DataBytes
	if end > uint64(len(b)) {
		return nil, nil, fmt.Errorf("encoded audio exceeds preview")
	}
	var pcm []byte
	var err error
	switch codec {
	case 16:
		pcm, err = decodeFADPCM(b[start:end], sample.Frames, sample.Channels)
	case 2:
		if binary.LittleEndian.Uint32(b[32:])&1 != 0 {
			return nil, nil, fmt.Errorf("big-endian PCM playback is unavailable")
		}
		size := sample.Frames * uint64(sample.Channels) * 2
		if size > 16<<20 || size > sample.DataBytes {
			return nil, nil, fmt.Errorf("invalid PCM stream size")
		}
		pcm = append([]byte(nil), b[start:start+size]...)
	default:
		return nil, nil, fmt.Errorf("playback is unavailable for codec %d", codec)
	}
	if err != nil {
		return nil, nil, err
	}
	sample.PCMHash = digest(pcm)
	wave := make([]byte, 44+len(pcm))
	copy(wave, "RIFF")
	binary.LittleEndian.PutUint32(wave[4:], uint32(36+len(pcm)))
	copy(wave[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wave[16:], 16)
	binary.LittleEndian.PutUint16(wave[20:], 1)
	binary.LittleEndian.PutUint16(wave[22:], uint16(sample.Channels))
	binary.LittleEndian.PutUint32(wave[24:], uint32(sample.Rate))
	binary.LittleEndian.PutUint32(wave[28:], uint32(sample.Rate*sample.Channels*2))
	binary.LittleEndian.PutUint16(wave[32:], uint16(sample.Channels*2))
	binary.LittleEndian.PutUint16(wave[34:], 16)
	copy(wave[36:], "data")
	binary.LittleEndian.PutUint32(wave[40:], uint32(len(pcm)))
	copy(wave[44:], pcm)
	peaks := make([]float64, 256)
	if sample.Frames > 0 {
		for frame := uint64(0); frame < sample.Frames; frame++ {
			bucket := min(uint64(255), frame*256/sample.Frames)
			for channel := 0; channel < sample.Channels; channel++ {
				value := int32(int16(binary.LittleEndian.Uint16(pcm[(int(frame)*sample.Channels+channel)*2:])))
				if value < 0 {
					value = -value
				}
				peak := float64(value) / 32768
				if peak > peaks[bucket] {
					peaks[bucket] = peak
				}
			}
		}
	}
	return wave, peaks, nil
}
