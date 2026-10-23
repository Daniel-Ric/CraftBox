package desktop

import (
	"encoding/base64"
	"testing"
)

func TestCinematicContainerValidation(t *testing.T) {
	for kind, data := range map[string]string{"gif": "GIF89a123456", "mp4": "\x00\x00\x00\x18ftypisom", "webm": "\x1a\x45\xdf\xa3webm", "json": "{\"version\":1}"} {
		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		if _, err := decodeCinematicFile(kind, encoded); err != nil {
			t.Fatal(kind, err)
		}
		if _, err := decodeCinematicFile(kind, base64.StdEncoding.EncodeToString([]byte("invalid"))); err == nil {
			t.Fatal("invalid container accepted", kind)
		}
	}
	if _, err := decodeCinematicFile("exe", "AAAA"); err == nil {
		t.Fatal("unsupported extension")
	}
}
