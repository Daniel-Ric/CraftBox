package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"os"
	"path/filepath"
	"testing"
)

func structureTestApp(t *testing.T, d *structure.Document) (*App, *structure.Session) {
	t.Helper()
	ctx := context.Background()
	s, e := structure.NewSession(ctx, t.TempDir(), d, nil)
	if e != nil {
		t.Fatal(e)
	}
	return &App{ctx: ctx, structureDocuments: map[string]*structure.Session{s.ID: s}}, s
}
func TestStructureWorldCopyPreservesOriginalAndWritesPlacement(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	source := filepath.Join(t.TempDir(), "source")
	reg := structure.Registry()
	db, e := (mcdb.Config{Blocks: reg}).Open(source)
	if e != nil {
		t.Fatal(e)
	}
	old, _ := structure.New("Old", structure.Vec3{1, 1, 1})
	h := structure.NewHistory(old)
	if e = h.Apply(ctx, structure.Edit{Kind: "fill", Block: structure.State{Name: "minecraft:stone"}}); e != nil {
		t.Fatal(e)
	}
	if e = structure.PlaceInDatabase(ctx, h.Doc, db, reg, structure.Vec3{0, 64, 0}, "replace", world.Overworld, false, nil); e != nil {
		t.Fatal(e)
	}
	if e = db.Close(); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(filepath.Join(source, "level.dat"))
	if e != nil {
		t.Fatal(e)
	}
	d, _ := structure.New("New", structure.Vec3{1, 1, 1})
	h = structure.NewHistory(d)
	if e = h.Apply(ctx, structure.Edit{Kind: "fill", Block: structure.State{Name: "minecraft:gold_block"}}); e != nil {
		t.Fatal(e)
	}
	app, s := structureTestApp(t, h.Doc)
	output := filepath.Join(t.TempDir(), "copy.mcworld")
	_, e = app.ExportStructureWorld(StructurePlacement{ID: s.ID, Source: source, Destination: output, Anchor: structure.Vec3{16, 64, 0}, Mode: "replace", BlockData: true, Inventories: true})
	if e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(filepath.Join(source, "level.dat"))
	if !bytes.Equal(before, after) {
		t.Fatal("source metadata changed")
	}
	root, temp, e := extractRenderArchive(ctx, output)
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(temp)
	copied, e := structure.FromWorld(ctx, root, "Check", structure.Bounds{Min: structure.Vec3{0, 64, 0}, Max: structure.Vec3{16, 64, 0}}, 0, nil)
	if e != nil {
		t.Fatal(e)
	}
	state, _ := copied.StateAt(structure.Vec3{})
	if state.Name != "minecraft:stone" {
		t.Fatal(state)
	}
	state, _ = copied.StateAt(structure.Vec3{16, 0, 0})
	if state.Name != "minecraft:gold_block" {
		t.Fatal(state)
	}
	original, e := structure.FromWorld(ctx, source, "Original", structure.Bounds{Min: structure.Vec3{16, 64, 0}, Max: structure.Vec3{16, 64, 0}}, 0, nil)
	if e != nil {
		t.Fatal(e)
	}
	if original.Cell(structure.Vec3{}).Known {
		t.Fatal("original received placement")
	}
	app.state.Worlds = []SavedWorld{{ID: "completed-capture", Name: "Completed", Path: source, State: "saved"}}
	imported, e := app.ImportStructureWorld(StructureImport{Name: "From completed capture", WorldID: "completed-capture", Bounds: structure.Bounds{Min: structure.Vec3{0, 64, 0}, Max: structure.Vec3{0, 64, 0}}})
	if e != nil {
		t.Fatal("saved capture did not resolve its path", e)
	}
	state, e = app.GetStructureBlock(imported.ID, structure.Vec3{})
	if e != nil || state.Name != "minecraft:stone" {
		t.Fatal(state, e)
	}
}
func TestFailedStructureWorldExportRetainsExistingOutput(t *testing.T) {
	d, _ := structure.New("Unknown", structure.Vec3{1, 1, 1})
	d.DefaultKnown = false
	app, s := structureTestApp(t, d)
	output := filepath.Join(t.TempDir(), "previous.mcworld")
	if e := os.WriteFile(output, []byte("previous output"), 0600); e != nil {
		t.Fatal(e)
	}
	_, e := app.ExportStructureWorld(StructurePlacement{ID: s.ID, Destination: output, Anchor: structure.Vec3{0, 64, 0}, Mode: "replace"})
	if e == nil {
		t.Fatal("unrecorded data was silently exported")
	}
	raw, _ := os.ReadFile(output)
	if string(raw) != "previous output" {
		t.Fatal("failed export replaced the existing output")
	}
}
func TestStructurePreviewCancellationIsScopedToViewer(t *testing.T) {
	a := NewApp()
	first := a.structurePreviewContext("blueprint", "main", "first")
	ghost := a.structurePreviewContext("blueprint", "ghost", "other")
	if first.Err() != nil || ghost.Err() != nil {
		t.Fatal("viewers cancelled each other")
	}
	a.CancelStructurePreview("blueprint", "main")
	if first.Err() == nil || ghost.Err() != nil {
		t.Fatal("cancellation escaped viewer scope")
	}
	next := a.structurePreviewContext("blueprint", "main", "second")
	if next.Err() != nil || ghost.Err() != nil {
		t.Fatal("restart cancelled another viewer")
	}
	a.cancelStructurePreviews("blueprint")
	if next.Err() == nil || ghost.Err() == nil {
		t.Fatal("document teardown did not release all previews")
	}
}
func TestStructurePackInstallPreservesStackAndInactivePacks(t *testing.T) {
	ctx := context.Background()
	target := t.TempDir()
	makePack := func(root, folder, id string) string {
		p := filepath.Join(root, "resource_packs", folder)
		if e := os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
		raw, _ := json.Marshal(map[string]any{"format_version": 2, "header": map[string]any{"uuid": id, "version": []int{1, 0, 0}}})
		if e := os.WriteFile(filepath.Join(p, "manifest.json"), raw, 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	makePack(target, "a", "inactive")
	makePack(target, "b", "base")
	makePack(target, "z", "override")
	if e := os.WriteFile(filepath.Join(target, "world_resource_packs.json"), []byte(`[{"pack_id":"override","version":[1,0,0],"custom":true},{"pack_id":"base","version":[1,0,0]}]`), 0600); e != nil {
		t.Fatal(e)
	}
	p := makePack(t.TempDir(), "new", "added")
	d, _ := structure.New("Packs", structure.Vec3{1, 1, 1})
	d.Packs = []structure.Dependency{{ID: "added", UUID: "added", Kind: "resource_packs", Source: p}}
	if e := installStructurePacks(ctx, d, target); e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(filepath.Join(target, "world_resource_packs.json"))
	var stack []map[string]any
	if e := json.Unmarshal(raw, &stack); e != nil {
		t.Fatal(e)
	}
	if len(stack) != 3 || stack[0]["pack_id"] != "override" || stack[1]["pack_id"] != "base" || stack[2]["pack_id"] != "added" || stack[0]["custom"] != true {
		t.Fatal(stack)
	}
}
