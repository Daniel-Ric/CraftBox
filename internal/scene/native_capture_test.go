package scene

import (
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"os"
	"strings"
	"testing"
)

func TestCapturedFenceAndDecorativeStates(t *testing.T) {
	out := os.Getenv("CRAFT_BOX_NATIVE_DETAILS")
	if out == "" {
		t.Skip("optional actual-state diagnostics")
	}
	reg := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]merge.Block{}}
	db, err := (mcdb.Config{Blocks: reg, LDBOptions: &opt.Options{ReadOnly: true}}).Open(os.Getenv("CRAFT_BOX_CAPTURE_FIXTURE"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	it := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer it.Release()
	type entry struct {
		X, Y, Z int
		Name    string
		Props   map[string]any
	}
	records := []entry{}
	counts := map[string]int{}
	relevant := func(n string) bool {
		return strings.Contains(n, "fence") || strings.Contains(n, "anvil") || strings.Contains(n, "sapling") || strings.Contains(n, "leaves") || n == "minecraft:vine"
	}
	for it.Next() {
		c := it.Column().Chunk
		cp := it.Position()
		for sy, sub := range c.Sub() {
			scan := false
			for _, l := range sub.Layers() {
				p := l.Palette()
				for i := 0; i < p.Len(); i++ {
					n, _, _ := reg.RuntimeIDToState(p.Value(uint16(i)))
					if relevant(n) {
						scan = true
					}
				}
			}
			if !scan {
				continue
			}
			for x := 0; x < 16; x++ {
				for z := 0; z < 16; z++ {
					for dy := 0; dy < 16; dy++ {
						y := c.Range()[0] + sy*16 + dy
						n, props, _ := reg.RuntimeIDToState(c.Block(uint8(x), int16(y), uint8(z), 0))
						if relevant(n) {
							records = append(records, entry{int(cp[0])*16 + x, y, int(cp[1])*16 + z, n, props})
							counts[n]++
						}
					}
				}
			}
		}
	}
	if err = it.Error(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(out, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log(counts)
}
