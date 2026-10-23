package scene

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapturedBlockDetails(t *testing.T) {
	root := os.Getenv("CRAFT_BOX_CAPTURE_FIXTURE")
	if root == "" {
		t.Skip("local fixture")
	}
	reg := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]merge.Block{}}
	db, err := (mcdb.Config{Blocks: reg, LDBOptions: &opt.Options{ReadOnly: true}}).Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	it := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer it.Release()
	seen := map[string]bool{}
	portals := map[cube.Pos]string{}
	details := map[cube.Pos]string{}
	states := map[cube.Pos]map[string]any{}
	var catalog *Catalog
	if assets := os.Getenv("CRAFT_BOX_CAPTURE_ASSETS"); assets != "" {
		catalog = NewCatalog()
		for _, folder := range []string{"resource_packs", "behavior_packs"} {
			packs, e := os.ReadDir(filepath.Join(assets, folder))
			if e != nil {
				t.Fatal(e)
			}
			for _, p := range packs {
				if !p.IsDir() {
					continue
				}
				fs := os.DirFS(filepath.Join(assets, folder, p.Name()))
				if folder == "resource_packs" {
					e = catalog.AddPack(fs)
				} else {
					e = catalog.AddBlockDefinitions(fs)
				}
				if e != nil {
					t.Fatal(e)
				}
			}
		}
	}
	pots := 0
	for it.Next() {
		col := it.Column()
		cp := it.Position()
		for _, e := range col.BlockEntities {
			if e.Data["id"] == "FlowerPot" && pots < 6 {
				t.Logf("POT %v %#v", e.Pos, e.Data)
				pots++
			}
		}
		for sy, sub := range col.Chunk.Sub() {
			scan := false
			for _, layer := range sub.Layers() {
				p := layer.Palette()
				for i := 0; i < p.Len(); i++ {
					name, props, _ := reg.RuntimeIDToState(p.Value(uint16(i)))
					if strings.Contains(name, "portal_") {
						scan = true
					}
					if strings.Contains(name, "sapling") || strings.Contains(name, "moss") || strings.Contains(name, "pot") || strings.Contains(name, "anvil") || strings.Contains(name, "amethyst") {
						key := name
						if !seen[key] {
							t.Logf("FORM %s %v", name, props)
							seen[key] = true
						}
					}
				}
			}
			if scan {
				for x := 0; x < 16; x++ {
					for z := 0; z < 16; z++ {
						for dy := 0; dy < 16; dy++ {
							y := col.Chunk.Range()[0] + sy*16 + dy
							name, props, _ := reg.RuntimeIDToState(col.Chunk.Block(uint8(x), int16(y), uint8(z), 0))
							if strings.Contains(name, "portal_") {
								p := cube.Pos{int(cp[0])*16 + x, y, int(cp[1])*16 + z}
								portals[p] = name
								states[p] = props
								details[p] = fmt.Sprintf("%s %v", name, props)
							}
						}
					}
				}
			}
		}
	}
	counts := map[string]int{}
	for p, name := range portals {
		if name == "cubecraft:portal_side" {
			for _, d := range []cube.Pos{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
				n := cube.Pos{p[0] + d[0], p[1] + d[1], p[2] + d[2]}
				if portals[n] == "cubecraft:portal_inside" {
					if counts[name] < 6 {
						t.Logf("SIDE %v %s inside=%v", p, details[p], d)
					}
					if catalog != nil {
						m, ok := catalog.customModel(name, states[p])
						if !ok || len(m.quads) < 7 {
							t.Fatal("missing portal side model")
						}
						center := func(quads []modelQuad) [3]float32 {
							var out [3]float32
							for _, q := range quads {
								for _, v := range q.points {
									for a := range out {
										out[a] += v[a] / float32(len(quads)*4)
									}
								}
							}
							return out
						}
						white, blue := center(m.quads[:6]), center(m.quads[6:])
						dot := float32(0)
						for a := 0; a < 3; a++ {
							dot += (blue[a] - white[a]) * float32(d[a])
						}
						if dot <= .01 {
							t.Errorf("portal edge faces outward at %v %v white=%v blue=%v inside=%v", p, states[p], white, blue, d)
						}
					}
					counts[name]++
				}
			}
		}
	}
	if catalog != nil && counts["cubecraft:portal_side"] == 0 {
		t.Fatal("fixture did not exercise portal edges")
	}
	t.Logf("Verified %d actual portal-edge/inside pairs", counts["cubecraft:portal_side"])
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
}
