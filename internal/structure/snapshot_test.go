package structure

import (
	"context"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"os"
	"path/filepath"
	"testing"
)

func TestLiveSnapshotRemainsDetachedAndMissingChunksStayUnknown(t *testing.T) {
	ctx := context.Background()
	reg := Registry()
	s := scene.NewStore(reg, nil)
	defer s.Close()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", nil)
	water, _ := reg.StateToRuntimeID("minecraft:water", map[string]any{"liquid_depth": int32(0)})
	c := chunk.New(reg, cube.Range{0, 15})
	c.SetBlock(0, 0, 0, 0, stone)
	c.SetBlock(0, 0, 0, 1, water)
	s.Put(world.ChunkPos{0, 0}, c, map[cube.Pos]map[string]any{{0, 0, 0}: {"id": "Sign", "Text": "Keep this"}})
	d, err := FromScene(ctx, s, "Snapshot", Bounds{Min: Vec3{}, Max: Vec3{31, 0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Cell(Vec3{15, 0, 0}).Known || d.Cell(Vec3{16, 0, 0}).Known {
		t.Fatal("missing chunk was treated as air")
	}
	cell := d.Cell(Vec3{})
	if d.Palette[cell.Layers[1]].Name != "minecraft:water" {
		t.Fatal("second block layer lost")
	}
	c.SetBlock(0, 0, 0, 0, reg.AirRuntimeID())
	s.Put(world.ChunkPos{0, 0}, c)
	state, _ := d.StateAt(Vec3{})
	if state.Name != "minecraft:stone" {
		t.Fatal("live capture modified detached blueprint")
	}
	err = s.VisitColumns(ctx, nil, func(p world.ChunkPos, col *chunk.Column, r world.BlockRegistry) error {
		s.Put(p, col.Chunk)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestCapturedSessionOwnsOriginalResourcePack(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	ctx := context.Background()
	reg := Registry()
	s := scene.NewStore(reg, nil)
	defer s.Close()
	s.Put(world.ChunkPos{}, chunk.New(reg, cube.Range{0, 15}))
	source := t.TempDir()
	manifest := `{// A valid captured JSONC manifest.
"format_version":2,"header":{"name":"Original resource pack","description":"test","uuid":"00000000-0000-4000-8000-000000000001","version":[1,0,0],"min_engine_version":[1,21,0]},"modules":[{"type":"resources","uuid":"00000000-0000-4000-8000-000000000002","version":[1,0,0]}]}`
	if e := os.WriteFile(filepath.Join(source, "manifest.json"), []byte(manifest), 0600); e != nil {
		t.Fatal(e)
	}
	pack, e := resource.ReadPath(source)
	if e != nil {
		t.Fatal(e)
	}
	saved, e := NewCapturedSession(ctx, t.TempDir(), s, "Live", Bounds{Min: Vec3{}, Max: Vec3{}}, []*resource.Pack{pack}, nil)
	if e != nil {
		t.Fatal(e)
	}
	if len(saved.History.Doc.Packs) != 1 {
		t.Fatal("original live pack missing")
	}
	dep := saved.History.Doc.Packs[0]
	if dep.Source == source || dep.Missing {
		t.Fatal("pack was not adopted", dep)
	}
	if _, e = os.Stat(filepath.Join(dep.Source, "manifest.json")); e != nil {
		t.Fatal(e)
	}
}
func TestSessionTracksUnsavedCameraChanges(t *testing.T) {
	d, _ := New("Camera", Vec3{1, 1, 1})
	s, err := NewSession(context.Background(), t.TempDir(), d, map[string]any{"layer": 0})
	if err != nil {
		t.Fatal(err)
	}
	s.MarkSaved()
	if s.Dirty() {
		t.Fatal("saved session marked dirty")
	}
	s.View = map[string]any{"layer": 1}
	if !s.Dirty() {
		t.Fatal("camera/layer changes must be saved into project")
	}
}
