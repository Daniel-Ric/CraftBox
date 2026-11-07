package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

type entry struct {
	Count    int            `json:"count"`
	Level    any            `json:"packEmission,omitempty"`
	IDs      map[uint32]int `json:"runtimeIDs"`
	Examples [][3]int       `json:"examples"`
	Rejected int            `json:"rejectedByRendererIDGuard"`
}

func main() {
	root := os.Args[1]
	definitions := map[string]any{}
	err := filepath.WalkDir(filepath.Join(root, "behavior_packs"), func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		var v struct {
			Block struct {
				Description struct {
					ID string `json:"identifier"`
				} `json:"description"`
				Components map[string]any `json:"components"`
			} `json:"minecraft:block"`
		}
		if json.Unmarshal(data, &v) == nil && v.Block.Description.ID != "" {
			if l, ok := v.Block.Components["minecraft:light_emission"]; ok {
				definitions[v.Block.Description.ID] = l
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	reg := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: make(map[uint32]merge.Block)}
	db, err := (mcdb.Config{Blocks: reg, LDBOptions: &opt.Options{ReadOnly: true}}).Open(root)
	if err != nil {
		panic(err)
	}

	var store *scene.Store
	if len(os.Args) > 2 {
		catalog, e := scene.LoadCatalog("assets/vanilla-3d")
		if e != nil {
			panic(e)
		}
		packs, e := os.ReadDir(filepath.Join(root, "behavior_packs"))
		if e != nil {
			panic(e)
		}
		for _, p := range packs {
			if p.IsDir() {
				if e = catalog.AddBlockDefinitions(os.DirFS(filepath.Join(root, "behavior_packs", p.Name()))); e != nil {
					panic(e)
				}
			}
		}
		store = scene.NewStore(reg, catalog)
		defer store.Close()
	}
	it := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer it.Release()
	counts := map[string]*entry{}
	chunks := 0
	for it.Next() {
		chunks++
		c := it.Column().Chunk
		pos := it.Position()
		if store != nil && pos[0] >= -1 && pos[0] <= 1 && pos[1] >= 1 && pos[1] <= 3 {
			store.Put(pos, c)
		}
		for sy, sub := range c.Sub() {
			for _, layer := range sub.Layers() {
				p := layer.Palette()
				relevant := false
				for i := 0; i < p.Len(); i++ {
					n, _, _ := reg.RuntimeIDToState(p.Value(uint16(i)))
					_, custom := definitions[n]
					if custom || strings.Contains(n, "light_block") || reg.LightBlock(p.Value(uint16(i))) > 0 {
						relevant = true
						break
					}
				}
				if !relevant {
					continue
				}
				for x := byte(0); x < 16; x++ {
					for z := byte(0); z < 16; z++ {
						for y := byte(0); y < 16; y++ {
							rid := layer.At(x, y, z)
							n, _, _ := reg.RuntimeIDToState(rid)
							l, custom := definitions[n]
							if !custom && !strings.Contains(n, "light_block") && reg.LightBlock(rid) == 0 {
								continue
							}
							e := counts[n]
							if e == nil {
								e = &entry{Level: l, IDs: map[uint32]int{}}
								counts[n] = e
							}
							e.Count++
							e.IDs[rid]++
							if uint64(rid) >= uint64(reg.BlockCount()) {
								e.Rejected++
							}
							if len(e.Examples) < 4 {
								e.Examples = append(e.Examples, [3]int{int(pos[0])*16 + int(x), c.Range()[0] + sy*16 + int(y), int(pos[1])*16 + int(z)})
							}
						}
					}
				}
			}
		}
	}
	if err := it.Error(); err != nil {
		panic(err)
	}
	result := map[string]any{"chunks": chunks, "registryBlockCount": reg.BlockCount(), "packLightDefinitions": len(definitions), "sources": counts}
	if store != nil {
		start := time.Now()
		page, e := store.Page(context.Background(), scene.PageRequest{Limit: 1, Focus: [2]int32{0, 2}}, nil)
		if e != nil {
			panic(e)
		}
		found := false
		for _, ch := range page.Chunks {
			for _, l := range ch.Lights {
				if l.Position == ([3]float32{13.5, 46.5, 4.5}) && l.Level == 15 {
					found = true
				}
			}
		}
		if !found {
			panic("captured Hive lamp missing from rendered page")
		}
		encoded, e := json.Marshal(page)
		if e != nil {
			panic(e)
		}
		if e = os.WriteFile(os.Args[2], encoded, 0644); e != nil {
			panic(e)
		}
		result["renderCheck"] = map[string]any{"customLampRecognized": found, "elapsedMs": time.Since(start).Milliseconds(), "pageBytes": len(encoded)}
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
