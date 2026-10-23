package desktop

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderImportRejectsArchiveTraversal(t *testing.T) {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	f, _ := w.Create("../escape.txt")
	f.Write([]byte("bad"))
	w.Close()
	path := filepath.Join(t.TempDir(), "bad.mcworld")
	os.WriteFile(path, b.Bytes(), 0600)
	if _, _, e := extractRenderArchive(context.Background(), path); e == nil {
		t.Fatal("unsafe archive accepted")
	}
}

func TestRenderPNGValidation(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 16, 9)))
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	if _, e := decodeRenderPNG(data); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{"data:image/jpeg;base64,AA==", "not an image", "data:image/png;base64,AAAA"} {
		if _, e := decodeRenderPNG(bad); e == nil {
			t.Fatal("invalid image accepted")
		}
	}
}

func TestRenderPackSelectionUsesOnlyActiveUUIDAndVersion(t *testing.T) {
	root := t.TempDir()
	for _, entry := range []struct {
		name    string
		version int
	}{{"active-old", 1}, {"active-new", 2}, {"inactive", 3}} {
		path := filepath.Join(root, "resource_packs", entry.name)
		os.MkdirAll(path, 0700)
		os.WriteFile(filepath.Join(path, "manifest.json"), []byte(fmt.Sprintf(`{"header":{"uuid":"id","version":[%d,0,0]}}`, entry.version)), 0600)
	}
	os.WriteFile(filepath.Join(root, "world_resource_packs.json"), []byte(`[{"pack_id":"id","version":[1,0,0]}]`), 0600)
	paths, e := renderPackPaths(root, "resource_packs", "world_resource_packs.json")
	if e != nil || len(paths) != 1 || filepath.Base(paths[0]) != "active-old" {
		t.Fatal("inactive or incorrect pack selected", paths, e)
	}
}

func TestRenderWorldImportAndEntityFreePages(t *testing.T) {
	root := t.TempDir()
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	db, e := mcdb.Config{Blocks: reg}.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	ch := chunk.New(reg, cube.Range{-64, 319})
	stone, _ := reg.StateToRuntimeID("minecraft:stone", nil)
	ch.SetBlock(1, 64, 1, 0, stone)
	db.StoreColumn(world.ChunkPos{}, world.Overworld, &chunk.Column{Chunk: ch})
	db.Close()

	writeBehavior := func(definition string) {
		var pack bytes.Buffer
		zw := zip.NewWriter(&pack)
		manifest, _ := zw.Create("manifest.json")
		manifest.Write([]byte(`{"format_version":2,"header":{"name":"Fixture","description":"Render fixture","uuid":"b85755e8-3b61-46a7-8eb3-5907736ab4b2","version":[1,0,0],"min_engine_version":[1,21,0]},"modules":[{"type":"data","uuid":"fc77df53-4a82-449c-8a59-ff0ee9097f97","version":[1,0,0]}]}`))
		block, _ := zw.Create("blocks/fixture.json")
		block.Write([]byte(definition))
		zw.Close()
		os.MkdirAll(filepath.Join(root, "behavior_packs"), 0700)
		os.WriteFile(filepath.Join(root, "behavior_packs", "fixture.mcpack"), pack.Bytes(), 0600)
		os.WriteFile(filepath.Join(root, "world_behavior_packs.json"), []byte(`[{"pack_id":"b85755e8-3b61-46a7-8eb3-5907736ab4b2","version":[1,0,0]}]`), 0600)
	}
	a := NewApp()
	writeBehavior(`{invalid block definition`)
	if _, e := a.importRenderWorld(context.Background(), root); e == nil {
		t.Fatal("archived behavior definitions were silently ignored")
	}
	writeBehavior(`{"format_version":"1.21.0","minecraft:block":{"description":{"identifier":"fixture:block"},"components":{}}}`)
	doc, e := a.importRenderWorld(context.Background(), root)
	if e != nil {
		t.Fatal(e)
	}
	defer a.CloseRenderWorld(doc.ID)
	if doc.Chunks != 1 {
		t.Fatal("chunks missing")
	}
	page, e := a.GetRenderScenePage(doc.ID, scene.PageRequest{})
	if e != nil || len(page.Chunks) != 1 {
		t.Fatal("imported scene unavailable", e)
	}
}
