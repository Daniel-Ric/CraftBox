package bedrocklab

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type BinaryField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type SoundSample struct {
	PCMHash    string  `json:"pcmHash,omitempty"`
	Name       string  `json:"name"`
	Rate       int     `json:"rate"`
	Channels   int     `json:"channels"`
	Frames     uint64  `json:"frames"`
	Duration   float64 `json:"duration"`
	DataOffset uint64  `json:"dataOffset"`
	DataBytes  uint64  `json:"dataBytes"`
	Codebook   string  `json:"codebook,omitempty"`
}
type BinaryInfo struct {
	Format      string        `json:"format"`
	Fields      []BinaryField `json:"fields"`
	Samples     []SoundSample `json:"samples"`
	PayloadHash string        `json:"payloadHash,omitempty"`
	Issue       string        `json:"issue,omitempty"`
	Limited     bool          `json:"limited"`
}
type MCBHeader struct {
	PayloadKey    string `json:"payloadKey"`
	FormatVersion string `json:"formatVersion"`
	PayloadOffset int    `json:"payloadOffset"`
}

func ReadMCBHeader(b []byte) (h MCBHeader, err error) {
	defer func() {
		if v := recover(); v != nil {
			if f, ok := v.(decodeFailure); ok {
				err = f.err
			} else {
				panic(v)
			}
		}
	}()
	m := &mcbReader{b: b, location: "MCB header"}
	if m.u32() != 0x42434d7f {
		return h, fmt.Errorf("invalid MCB magic")
	}
	major, minor, patch := m.u16(), m.u16(), m.u16()
	pre, build := m.str(), m.str()
	h.FormatVersion = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	if major == 9999 && minor == 9999 && patch == 9999 && pre == "beta" && build == "" {
		h.FormatVersion = "beta"
	} else {
		if pre != "" {
			h.FormatVersion += "-" + pre
		}
		if build != "" {
			h.FormatVersion += "+" + build
		}
	}
	h.PayloadKey = m.str()
	h.PayloadOffset = m.pos
	return h, nil
}

func InspectBinary(name string, b []byte, total int64) *BinaryInfo {
	info := &BinaryInfo{Fields: []BinaryField{}, Samples: []SoundSample{}}
	field := func(name string, v any) { info.Fields = append(info.Fields, BinaryField{name, fmt.Sprint(v)}) }
	if len(b) >= 4 && bytes.Equal(b[:4], []byte{0x7f, 'M', 'C', 'B'}) {
		info.Format = "Compiled Bedrock MCB"
		h, err := ReadMCBHeader(b)
		if err != nil {
			info.Issue = err.Error()
			return info
		}
		field("Document type", h.PayloadKey)
		field("Format version", h.FormatVersion)
		field("Header bytes", h.PayloadOffset)
		field("Compiled payload bytes", total-int64(h.PayloadOffset))
		return info
	}
	if len(b) >= 4 && string(b[:4]) == "FSB5" {
		info.Format = "FMOD FSB5"
		if len(b) < 60 {
			info.Issue = "Truncated FSB5 header"
			return info
		}
		u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(b[off : off+4]) }
		version, count, headers, names, data, codec := u32(4), u32(8), uint64(u32(12)), uint64(u32(16)), uint64(u32(20)), u32(24)
		base := uint64(60)
		if version == 0 {
			base = 64
		}
		if version > 1 {
			info.Issue = "Unsupported FSB5 header version"
			return info
		}
		namesAt := base + headers
		dataAt := namesAt + names
		codecs := map[uint32]string{1: "PCM 8-bit", 2: "PCM 16-bit", 3: "PCM 24-bit", 4: "PCM 32-bit", 5: "PCM float", 6: "GC ADPCM", 7: "IMA ADPCM", 8: "VAG", 9: "HEVAG", 10: "XMA", 11: "MPEG", 12: "CELT", 13: "ATRAC9", 14: "XWMA", 15: "Vorbis", 16: "FADPCM", 17: "Opus"}
		codecName := codecs[codec]
		if codecName == "" {
			codecName = fmt.Sprintf("Unknown codec %d", codec)
		}
		field("Codec", codecName)
		field("Streams", count)
		field("Container version", version)
		field("Sample header bytes", headers)
		field("Name table bytes", names)
		field("Encoded payload bytes", data)
		if dataAt+data != uint64(total) || headers < uint64(count)*8 || count > 10000 || names > 0 && names < uint64(count)*4 {
			info.Issue = "Invalid FSB5 section sizes or stream count"
			return info
		}
		if dataAt > uint64(len(b)) {
			info.Issue = "FSB5 metadata extends beyond the available preview"
			info.Limited = true
			return info
		}
		rates := []int{4000, 8000, 11000, 11025, 16000, 22050, 24000, 32000, 44100, 48000, 96000}
		channels := []int{1, 2, 6, 8}
		pos := base
		for i := uint32(0); i < count && i < 256; i++ {
			if pos+8 > namesAt {
				info.Issue = "Truncated FSB5 stream header"
				return info
			}
			mode := binary.LittleEndian.Uint64(b[pos : pos+8])
			pos += 8
			rateIndex := int(mode >> 1 & 15)
			if rateIndex >= len(rates) {
				info.Issue = "Unsupported FSB5 sample-rate code"
				return info
			}
			sample := SoundSample{Rate: rates[rateIndex], Channels: channels[mode>>5&3], Frames: mode >> 34 & 0x3fffffff, DataOffset: ((mode >> 7) & 0x07ffffff) << 5}
			if sample.DataOffset > data {
				info.Issue = "FSB5 stream offset exceeds data section"
				return info
			}
			if mode&1 != 0 {
				for chunks := 0; ; chunks++ {
					if chunks > 1024 || pos+4 > namesAt {
						info.Issue = "Invalid FSB5 metadata chunks"
						return info
					}
					chunk := u32(int(pos))
					pos += 4
					size := uint64(chunk >> 1 & 0xffffff)
					kind := chunk >> 25 & 0x7f
					if pos+size > namesAt {
						info.Issue = "FSB5 chunk exceeds sample header section"
						return info
					}
					switch {
					case kind == 1 && size >= 1:
						sample.Channels = int(b[pos])
					case kind == 2 && size >= 4:
						sample.Rate = int(u32(int(pos)))
					case kind == 11 && size >= 4:
						sample.Codebook = fmt.Sprintf("%08x", u32(int(pos)))
					}
					pos += size
					if chunk&1 == 0 {
						break
					}
				}
			}
			if sample.Rate < 1 || sample.Rate > 768000 || sample.Channels < 1 || sample.Channels > 64 {
				info.Issue = "Invalid FSB5 sample rate or channel count"
				return info
			}
			sample.Duration = float64(sample.Frames) / float64(sample.Rate)
			if names > 0 {
				off := uint64(u32(int(namesAt + uint64(i)*4)))
				if off < uint64(count)*4 || off >= names {
					info.Issue = "Invalid FSB5 stream name offset"
					return info
				}
				tail := b[namesAt+off : dataAt]
				end := bytes.IndexByte(tail, 0)
				if end < 0 {
					info.Issue = "Unterminated FSB5 stream name"
					return info
				}
				if end > 1024 {
					end = 1024
				}
				sample.Name = strings.ToValidUTF8(string(tail[:end]), "�")
			}
			if sample.Name == "" {
				sample.Name = fmt.Sprintf("Stream %d", i+1)
			}
			info.Samples = append(info.Samples, sample)
		}
		for i := range info.Samples {
			end := data
			if i+1 < len(info.Samples) {
				end = info.Samples[i+1].DataOffset
			} else if uint32(len(info.Samples)) < count {
				info.Limited = true
				break
			}
			if end < info.Samples[i].DataOffset {
				info.Issue = "FSB5 streams are not ordered by data offset"
				return info
			}
			info.Samples[i].DataBytes = end - info.Samples[i].DataOffset
		}
		if dataAt+data <= uint64(len(b)) {
			info.PayloadHash = digest(b[dataAt : dataAt+data])
		}
		return info
	}
	if len(b) >= 16 && binary.LittleEndian.Uint64(b) == 0x267052a0b125277d {
		info.Format = "Bedrock BRArchive"
		field("Format version", binary.LittleEndian.Uint32(b[12:]))
		field("Packed files", binary.LittleEndian.Uint32(b[8:]))
		return info
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".ogg" || ext == ".wav" || ext == ".mp3" {
		info.Format = "Audio file"
		field("Encoding", strings.TrimPrefix(ext, "."))
		return info
	}
	if !utf8.Valid(b) || bytes.ContainsRune(b, 0) {
		info.Format = "Binary data"
		field("Bytes", total)
		return info
	}
	return nil
}
