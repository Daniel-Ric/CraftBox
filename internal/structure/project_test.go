package structure

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPortableProjectPreservesUnknownAndPack(t *testing.T) {
	d, _ := New("Portable", Vec3{2, 1, 1})
	d.DefaultKnown = false
	d.Set(Vec3{}, Cell{Layers: [2]int32{0, -1}, Known: true})
	d.Anchor = Vec3{1, 0, 0}
	pack := t.TempDir()
	if err := os.WriteFile(filepath.Join(pack, "manifest.json"), []byte(`{"format_version":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	d.Packs = []Dependency{{ID: "pack1", Name: "Original", Kind: "resource", Source: pack}}
	var archive bytes.Buffer
	if err := WriteProject(context.Background(), &archive, d, map[string]any{"layer": 3}, true); err != nil {
		t.Fatal(err)
	}
	restored, view, err := ReadProject(context.Background(), bytes.NewReader(archive.Bytes()), int64(archive.Len()), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if restored.Cell(Vec3{1, 0, 0}).Known || !restored.Cell(Vec3{}).Known || restored.Anchor != d.Anchor {
		t.Fatal("portable roundtrip changed missing coverage or anchor")
	}
	if len(restored.Packs) != 1 || restored.Packs[0].Missing {
		t.Fatal("pack lost")
	}
	if _, err := os.Stat(filepath.Join(restored.Packs[0].Source, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if view["layer"] != float64(3) {
		t.Fatal("lost view settings")
	}
}

func TestProjectRejectsArchiveTraversal(t *testing.T) {
	var raw bytes.Buffer
	z := zip.NewWriter(&raw)
	f, _ := z.Create("../escape")
	f.Write([]byte("bad"))
	z.Close()
	if _, _, err := ReadProject(context.Background(), bytes.NewReader(raw.Bytes()), int64(raw.Len()), t.TempDir()); err == nil {
		t.Fatal("archive path traversal accepted")
	}
}
