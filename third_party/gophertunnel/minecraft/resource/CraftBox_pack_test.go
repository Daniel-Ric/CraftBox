package resource

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"testing"
)

func TestCapturePackArchiveAndFilesystem(t *testing.T) {
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	for name, data := range map[string]string{
		"pack/manifest.json":     `{"format_version":2,"header":{"name":"Capture test","description":"test","uuid":"85149a83-050c-4113-8f6e-739c11a64b42","version":[1,0,0],"min_engine_version":[1,26,50]},"modules":[{"type":"resources","uuid":"804c5aa2-37f0-4c3d-ab3e-77ceaceab17b","version":[1,0,0]}]}`,
		"pack/textures/test.txt": "unchanged texture bytes",
	} {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(f, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	pack, err := FromReaderAt(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(pack, "textures/test.txt")
	if err != nil || string(data) != "unchanged texture bytes" {
		t.Fatalf("texture = %q, %v", data, err)
	}
	if _, err := fs.ReadFile(pack, "blocks.json"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing optional file must report fs.ErrNotExist, got %v", err)
	}
	var forwarded bytes.Buffer
	if _, err := pack.WriteTo(&forwarded); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(forwarded.Bytes(), archive.Bytes()) {
		t.Fatal("forwarding changed pack archive")
	}
	encrypted := pack.WithContentKey("key")
	if encrypted.CanRead() || !pack.CanRead() {
		t.Fatal("content key mutated original pack")
	}
}

func TestCapturePackLookupsDoNotReparseTheArchive(t *testing.T) {
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	manifest := `{"format_version":2,"header":{"name":"Lookup test","description":"test","uuid":"85149a83-050c-4113-8f6e-739c11a64b42","version":[1,0,0],"min_engine_version":[1,26,50]},"modules":[{"type":"resources","uuid":"804c5aa2-37f0-4c3d-ab3e-77ceaceab17b","version":[1,0,0]}]}`
	for i := 0; i < 102; i++ {
		name, data := fmt.Sprintf("pack/textures/filler-%d.txt", i), "unchanged texture"
		if i == 0 {
			name, data = "pack/manifest.json", manifest
		}
		if i == 101 {
			name = "pack/textures/LAST.txt"
		}
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
	pack, err := Read(bytes.NewReader(archive.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(1, func() {
		if _, err := fs.ReadFile(pack, "textures/not-in-this-overlay.png"); !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
	})
	if allocs > 50 {
		t.Fatalf("missing texture lookup reparsed archive: %.0f allocations", allocs)
	}
	keyed := pack.WithContentKey("test-copy")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			for _, p := range []*Pack{pack, keyed} {
				data, err := fs.ReadFile(p, "textures/last.TXT")
				if err != nil || string(data) != "unchanged texture" {
					t.Errorf("case-insensitive relative lookup: %q %v", data, err)
				}
				data, err = p.ReadFile("PACK/TEXTURES/LAST.txt")
				if err != nil || string(data) != "unchanged texture" {
					t.Errorf("full lookup: %q %v", data, err)
				}
				if p.BaseDir() != "pack" {
					t.Error("lost base directory")
				}
				f, err := p.Open("textures/LAST.txt")
				if err != nil {
					t.Error(err)
				} else {
					f.Close()
				}
			}
		})
	}
	wg.Wait()
	var original bytes.Buffer
	if _, err := pack.WriteTo(&original); err != nil || !bytes.Equal(original.Bytes(), archive.Bytes()) {
		t.Fatal("pack forwarding changed", err)
	}
}
