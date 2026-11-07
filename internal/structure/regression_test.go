package structure

import (
	"context"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"os"
	"path/filepath"
	"testing"
)

func TestCutEntitiesIsAtomicAndUndoable(t *testing.T) {
	ctx := context.Background()
	d, _ := New("Cut", Vec3{4, 2, 1})
	d.Entities = []map[string]any{{"identifier": "minecraft:armor_stand", "Pos": []float32{.5, .5, .5}}}
	h := NewHistory(d)
	if err := h.Apply(ctx, Edit{Kind: "cut", Bounds: []Bounds{{Min: Vec3{}, Max: Vec3{0, 1, 0}}}}); err != nil {
		t.Fatal(err)
	}
	if len(h.Doc.Entities) != 0 || len(h.Clipboard.Entities) != 1 {
		t.Fatal("cut duplicated entity")
	}
	if err := h.Apply(ctx, Edit{Kind: "paste", Destination: Vec3{2, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	if len(h.Doc.Entities) != 1 {
		t.Fatal("paste count")
	}
	p, _ := entityPosition(h.Doc.Entities[0])
	if p[0] != 2.5 {
		t.Fatal(p)
	}
	h.Undo()
	h.Undo()
	if len(h.Doc.Entities) != 1 {
		t.Fatal("undo did not restore source entity")
	}
}
func TestWallShulkerAndChestReferencesTransform(t *testing.T) {
	ctx := context.Background()
	d, _ := New("Directional", Vec3{3, 1, 2})
	d.Origin = Vec3{100, 60, 200}
	idx, _ := d.AddState(State{Name: "minecraft:cobblestone_wall", States: map[string]any{"wall_connection_type_north": "tall", "wall_connection_type_east": "short", "wall_connection_type_south": "none", "wall_connection_type_west": "none"}})
	d.Set(Vec3{}, Cell{[2]int32{idx, -1}, true})
	d.BlockData[Key(Vec3{})] = map[string]any{"block_entity_data": map[string]any{"id": "ShulkerBox", "facing": uint8(5), "pairx": int32(101), "pairz": int32(200)}}
	out, w, err := Transform(ctx, d, 1, "")
	if err != nil || len(w) != 0 {
		t.Fatal(err, w)
	}
	s, _ := out.StateAt(Vec3{1, 0, 0})
	if s.States["wall_connection_type_east"] != "tall" || s.States["wall_connection_type_south"] != "short" {
		t.Fatal(s)
	}
	be := out.BlockData[Key(Vec3{1, 0, 0})]["block_entity_data"].(map[string]any)
	if be["facing"] != uint8(3) || be["pairx"] != int32(101) || be["pairz"] != int32(201) {
		t.Fatal(be)
	}
	target, _ := New("Target", Vec3{8, 2, 8})
	placed, _, err := Place(ctx, d, target, Vec3{2, 0, 3}, "replace", false)
	if err != nil {
		t.Fatal(err)
	}
	be = placed.BlockData[Key(Vec3{2, 0, 3})]["block_entity_data"].(map[string]any)
	if be["pairx"] != int32(3) || be["pairz"] != int32(3) {
		t.Fatal("pair reference was not moved", be)
	}
}
func TestRepeatedWorldPlacementDoesNotOverwriteEntities(t *testing.T) {
	ctx := context.Background()
	d, _ := New("Entities", Vec3{1, 1, 1})
	idx, _ := d.AddState(State{Name: "minecraft:stone"})
	d.Set(Vec3{}, Cell{[2]int32{idx, -1}, true})
	d.Entities = []map[string]any{{"identifier": "minecraft:armor_stand", "Pos": []float32{.5, .5, .5}, "Rotation": []float32{0, 0}}}
	reg := Registry()
	db, err := (mcdb.Config{Blocks: reg}).Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.SaveLocalPlayerData(map[string]any{"UniqueID": int64(-4000)}); err != nil {
		t.Fatal(err)
	}
	for _, anchor := range []Vec3{{0, 64, 0}, {32, 64, 0}} {
		if err = PlaceInDatabase(ctx, d, db, reg, anchor, "replace", world.Overworld, true, nil); err != nil {
			t.Fatal(err)
		}
	}
	ids := map[int64]bool{}
	for _, x := range []int32{0, 2} {
		col, e := db.LoadColumn(world.ChunkPos{x, 0}, world.Overworld)
		if e != nil {
			t.Fatal(e)
		}
		if len(col.Entities) != 1 {
			t.Fatalf("chunk %d: %d entities", x, len(col.Entities))
		}
		id := col.Entities[0].ID
		if id >= -4000 {
			t.Fatal("new entity overlapped reserved player identity range", id)
		}
		if ids[id] {
			t.Fatal("entity ID reused", id)
		}
		ids[id] = true
	}
}
func TestRecoveryOwnsPackAndRetainsViewAndDirty(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	d, _ := New("Recovery", Vec3{2, 1, 1})
	p := t.TempDir()
	if err := os.WriteFile(filepath.Join(p, "manifest.json"), []byte(`{"format_version":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	d.Packs = []Dependency{{ID: "one", Kind: "resource_packs", Source: p}}
	s, err := NewSession(ctx, root, d, map[string]any{"layer": float64(0)})
	if err != nil {
		t.Fatal(err)
	}
	s.Path = "somewhere.craftbox"
	if err = s.Autosave(ctx); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(p, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	restored, err := RecoverSession(ctx, root, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.Summary().Dirty || restored.Path != s.Path || restored.View["layer"] != float64(0) {
		t.Fatal(restored.Summary())
	}
	if restored.History.Doc.Packs[0].Missing {
		t.Fatal("owned pack missing after recovery")
	}
	if _, err = os.Stat(filepath.Join(restored.History.Doc.Packs[0].Source, "manifest.json")); err != nil {
		t.Fatal(err)
	}
}
