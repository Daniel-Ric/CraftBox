package scene

import (
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectCapturedActors(t *testing.T) {
	root := os.Getenv("CRAFT_BOX_CAPTURE_FIXTURE")
	if root == "" {
		t.Skip("local capture")
	}
	reg := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]merge.Block{}}
	db, err := (mcdb.Config{Blocks: reg, LDBOptions: &opt.Options{ReadOnly: true}}).Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	it := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer it.Release()
	actors := []any{}
	signs := []any{}
	counts := map[string]int{}
	catalog := NewCatalog()
	if assets := os.Getenv("CRAFT_BOX_CAPTURE_ASSETS"); assets != "" {
		var e error
		catalog, e = LoadCatalog("../../assets/vanilla-3d")
		if e != nil {
			t.Fatal(e)
		}
		packs, e := os.ReadDir(filepath.Join(assets, "resource_packs"))
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range packs {
			if p.IsDir() {
				if e = catalog.AddPack(os.DirFS(filepath.Join(assets, "resource_packs", p.Name()))); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
	faces := map[string]int{}
	for it.Next() {
		col := it.Column()
		for _, e := range col.Entities {
			actors = append(actors, e)
			id, _ := e.Data["identifier"].(string)
			counts[id]++
			if os.Getenv("CRAFT_BOX_CAPTURE_ASSETS") != "" {
				m := catalog.actorModel(ActorFromNBT(e.ID, e.Data))
				faces[id] += len(m.quads)
			}
		}
		for _, e := range col.BlockEntities {
			if e.Data["id"] == "Sign" || e.Data["id"] == "HangingSign" {
				signs = append(signs, e)
			}
		}
	}
	if err = it.Error(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.MarshalIndent(map[string]any{"counts": counts, "entities": actors, "signs": signs}, "", "  ")
	if err = os.WriteFile(filepath.Join(root, "actors.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("entities=%d signs=%d counts=%v", len(actors), len(signs), counts)
	t.Logf("actor faces=%v; unresolved=%v", faces, catalog.MissingTexturePaths())
}
