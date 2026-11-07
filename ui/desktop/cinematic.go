package desktop

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func decodeCinematicFile(kind, encoded string) ([]byte, error) {
	if len(encoded) == 0 || len(encoded) > 256<<20 {
		return nil, fmt.Errorf("Export is empty or exceeds 256 MiB")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	valid := false
	switch kind {
	case "gif":
		valid = len(raw) >= 10 && (bytes.HasPrefix(raw, []byte("GIF89a")) || bytes.HasPrefix(raw, []byte("GIF87a")))
	case "mp4":
		valid = len(raw) >= 12 && string(raw[4:8]) == "ftyp"
	case "mov":
		valid = len(raw) >= 12 && string(raw[4:8]) == "ftyp" && string(raw[8:12]) == "qt  "
	case "mkv":
		valid = bytes.HasPrefix(raw, []byte{0x1a, 0x45, 0xdf, 0xa3}) && bytes.Contains(raw[:min(len(raw), 256)], []byte("matroska"))
	case "webm":
		valid = bytes.HasPrefix(raw, []byte{0x1a, 0x45, 0xdf, 0xa3})
	case "json":
		valid = json.Valid(raw)
	}
	if !valid {
		return nil, fmt.Errorf("Invalid cinematic export (%s)", kind)
	}
	return raw, nil
}
