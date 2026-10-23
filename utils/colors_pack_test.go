package utils

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

func TestResolveColorsSkipsUnreadableEncryptedPacks(t *testing.T) {
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	for name, data := range map[string]string{
		"manifest.json":                   `{"format_version":2,"header":{"name":"Encrypted fixture","description":"test","uuid":"85149a83-050c-4113-8f6e-739c11a64b42","version":[1,0,0]},"modules":[{"type":"resources","uuid":"804c5aa2-37f0-4c3d-ab3e-77ceaceab17b","version":[1,0,0]}]}`,
		"blocks.json":                     "ciphertext, not JSON",
		"textures/flipbook_textures.json": "ciphertext, not JSON",
	} {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.WriteString(f, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	pack, err := resource.Read(bytes.NewReader(archive.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	logger := logrus.StandardLogger()
	var logs bytes.Buffer
	old := logger.Out
	logger.SetOutput(&logs)
	defer logger.SetOutput(old)
	colors := ResolveColors(nil, []*resource.Pack{pack.WithContentKey("fixture-key")})
	if len(colors) != 0 {
		t.Fatalf("colors from unreadable pack: %v", colors)
	}
	if bytes.Contains(logs.Bytes(), []byte("invalid literal")) || bytes.Contains(logs.Bytes(), []byte("level=error")) {
		t.Fatalf("encrypted bytes were parsed as JSON: %s", logs.String())
	}
}
