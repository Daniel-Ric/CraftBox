package subcommands

import (
	"context"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"os"
	"path/filepath"
	"testing"
)

func TestRender3DExportsTexturedWorld(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	os.MkdirAll(assets, 0755)
	os.WriteFile(filepath.Join(assets, "provenance.json"), []byte(`{"source":"test fixture"}`), 0644)
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	db, err := (mcdb.Config{Blocks: reg}).Open(filepath.Join(root, "world"))
	if err != nil {
		t.Fatal(err)
	}
	c := chunk.New(reg, cube.Range{-64, 319})
	c.SetBlock(0, 64, 0, 0, stone)
	if err = db.StoreColumn(world.ChunkPos{0, 0}, world.Overworld, &chunk.Column{Chunk: c}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	out := filepath.Join(root, "world.glb")
	err = (Render3DCMD{}).Run(context.Background(), &Render3DSettings{WorldPath: filepath.Join(root, "world"), Assets: assets, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data[:4]) != "glTF" {
		t.Fatal("missing GLB", err)
	}
}
