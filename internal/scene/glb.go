package scene

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

func EncodeGLB(s Snapshot) ([]byte, error) {
	if !s.Reset {
		return nil, fmt.Errorf("GLB requires a complete scene snapshot")
	}
	atlas, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s.Atlas, "data:image/png;base64,"))
	if err != nil || len(atlas) == 0 {
		return nil, fmt.Errorf("Texture atlas is missing or invalid")
	}
	w, err := NewGLBWriter(context.Background(), "")
	if err != nil {
		return nil, err
	}
	defer w.Close()
	for _, ch := range s.Chunks {
		if err := w.AddChunk(ch); err != nil {
			return nil, err
		}
	}
	var out bytes.Buffer
	if err := w.Finish(&out, atlas, s.Source); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
