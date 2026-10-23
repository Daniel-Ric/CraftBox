package scene

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"os"
	"path/filepath"
	"testing"
)

func TestOfficialBDSFixture(t *testing.T) {
	fixture := os.Getenv("CRAFT_BOX_BDS_FIXTURE")
	if fixture == "" {
		t.Skip("set CRAFT_BOX_BDS_FIXTURE to the stopped probe directory")
	}
	assets := os.Getenv("CRAFT_BOX_3D_ASSETS")
	c, err := LoadCatalog(assets)
	if err != nil {
		t.Fatal(err)
	}
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	db, err := (mcdb.Config{Blocks: reg, LDBOptions: &opt.Options{ReadOnly: true}}).Open(filepath.Join(fixture, "worlds", "3d-fixture"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(reg, c)
	defer store.Close()
	it := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer it.Release()
	names := map[string]int{}
	for it.Next() {
		ch := it.Column().Chunk
		store.Put(it.Position(), ch)
		for y := ch.Range()[0]; y <= ch.Range()[1]; y++ {
			for x := uint8(0); x < 16; x++ {
				for z := uint8(0); z < 16; z++ {
					name, _, _ := reg.RuntimeIDToState(ch.Block(x, int16(y), z, 0))
					if name != "minecraft:air" {
						names[name]++
					}
				}
			}
		}
	}
	if err = it.Error(); err != nil {
		t.Fatal(err)
	}
	t.Log("Captured block counts:", names)
	for name := range names {
		for face := 0; face < 6; face++ {
			if tile := c.texture(name, face); tile.missing {
				t.Errorf("missing texture %s face %d", name, face)
			}
		}
	}
	snapshot, err := store.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Chunks) != 2 {
		t.Fatalf("expected two BDS columns, got %d", len(snapshot.Chunks))
	}
	data, err := EncodeGLB(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(fixture, "verified.glb"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
