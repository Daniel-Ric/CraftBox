package utils

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"encoding/json"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"io/fs"
	"testing"
)

func encryptFixtureCFB8(data []byte, key string) []byte {
	block, _ := aes.NewCipher([]byte(key))
	iv := append([]byte{}, []byte(key)[:16]...)
	out := make([]byte, len(data))
	var stream [16]byte
	for i, b := range data {
		block.Encrypt(stream[:], iv)
		out[i] = b ^ stream[0]
		copy(iv, iv[1:])
		iv[15] = out[i]
	}
	return out
}
func TestReadablePackUsesServerKeyAndPreservesOriginal(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	fileKey := "abcdef0123456789abcdef0123456789"
	contents, _ := json.Marshal(map[string]any{"content": []map[string]string{{"path": "textures/test.png", "key": fileKey}, {"path": "manifest.json"}}})
	header := make([]byte, 256)
	copy(header[4:8], []byte{0xfc, 0xb9, 0xcf, 0x9b})
	files := map[string][]byte{"pack/manifest.json": []byte(`{"format_version":2,"header":{"name":"RP","description":"test","uuid":"85149a83-050c-4113-8f6e-739c11a64b42","version":[1,0,0]},"modules":[{"type":"resources","uuid":"804c5aa2-37f0-4c3d-ab3e-77ceaceab17b","version":[1,0,0]}]}`), "pack/contents.json": append(header, encryptFixtureCFB8(contents, key)...), "pack/textures/test.png": encryptFixtureCFB8([]byte("test pixels"), fileKey)}
	var raw bytes.Buffer
	writer := zip.NewWriter(&raw)
	for name, data := range files {
		f, _ := writer.Create(name)
		f.Write(data)
	}
	writer.Close()
	original, err := resource.Read(bytes.NewReader(raw.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	original = original.WithContentKey(key)
	decoded, err := ReadablePack(original)
	if err != nil {
		t.Fatal(err)
	}
	pixels, err := fs.ReadFile(decoded, "textures/test.png")
	if err != nil || string(pixels) != "test pixels" || !decoded.CanRead() {
		t.Fatalf("%q %v", pixels, err)
	}
	var unchanged bytes.Buffer
	original.WriteTo(&unchanged)
	if !bytes.Equal(unchanged.Bytes(), raw.Bytes()) {
		t.Fatal("forwarded original changed")
	}
	if _, err := ReadablePack(original.WithContentKey("wrong-key")); err == nil {
		t.Fatal("invalid key accepted")
	}
	if _, err := ReadablePack(original.WithContentKey("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")); err == nil {
		t.Fatal("wrong key accepted")
	}
}
