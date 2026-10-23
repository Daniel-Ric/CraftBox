//go:build gui

package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestCapturedScenePageReplay(t *testing.T) {
	root := os.Getenv("CRAFT_BOX_CAPTURE_FIXTURE")
	if root == "" {
		t.Skip("set CRAFT_BOX_CAPTURE_FIXTURE to an isolated captured database")
	}
	catalog, err := scene.LoadCatalog(os.Getenv("CRAFT_BOX_3D_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	packsRoot := os.Getenv("CRAFT_BOX_CAPTURE_PACKS")
	if packsRoot != "" {
		packs, err := os.ReadDir(packsRoot)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range packs {
			start := time.Now()
			pack, err := resource.ReadPath(filepath.Join(packsRoot, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err = catalog.AddPack(pack); err != nil {
				t.Fatal(err)
			}
			t.Logf("pack loaded in %s", time.Since(start))
		}
	}
	registry := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]merge.Block{}}
	if behaviors := os.Getenv("CRAFT_BOX_CAPTURE_BEHAVIORS"); behaviors != "" {
		entries, err := os.ReadDir(behaviors)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() {
				if err = catalog.AddBlockDefinitions(os.DirFS(filepath.Join(behaviors, e.Name()))); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	db, err := (mcdb.Config{Blocks: registry, LDBOptions: &opt.Options{ReadOnly: true}}).Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := scene.NewStore(registry, catalog)
	defer store.Close()
	it := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer it.Release()
	n := 0
	blockNames := map[string]bool{}
	var focus [2]int32
	for it.Next() {
		if bounds := os.Getenv("CRAFT_BOX_CAPTURE_BOUNDS"); bounds != "" {
			var minX, minZ, maxX, maxZ int32
			if _, e := fmt.Sscanf(bounds, "%d,%d,%d,%d", &minX, &minZ, &maxX, &maxZ); e != nil {
				t.Fatal(e)
			}
			p := it.Position()
			if p[0] < minX || p[0] > maxX || p[1] < minZ || p[1] > maxZ {
				continue
			}
		}
		column := it.Column().Chunk
		selected := os.Getenv("CRAFT_BOX_CAPTURE_BLOCK") == ""
		for _, sub := range column.Sub() {
			for _, layer := range sub.Layers() {
				palette := layer.Palette()
				for i := 0; i < palette.Len(); i++ {
					name, _, _ := registry.RuntimeIDToState(palette.Value(uint16(i)))
					blockNames[name] = true
					if prefix := os.Getenv("CRAFT_BOX_CAPTURE_BLOCK"); prefix != "" && strings.HasPrefix(name, prefix) {
						selected = true
					}
				}
			}
		}
		if !selected {
			continue
		}
		entities := map[cube.Pos]map[string]any{}
		for _, e := range it.Column().BlockEntities {
			entities[e.Pos] = e.Data
		}
		store.Put(it.Position(), column, entities)
		for _, e := range it.Column().Entities {
			store.PutActor(scene.ActorFromNBT(e.ID, e.Data))
		}
		if n == 0 {
			focus = [2]int32(it.Position())
		}
		n++
	}
	if path := os.Getenv("CRAFT_BOX_CAPTURE_ACTORS"); path != "" {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		var actors []scene.Actor
		if e = json.Unmarshal(raw, &actors); e != nil {
			t.Fatal(e)
		}
		for _, a := range actors {
			if bounds := os.Getenv("CRAFT_BOX_CAPTURE_BOUNDS"); bounds != "" {
				var minX, minZ, maxX, maxZ int32
				fmt.Sscanf(bounds, "%d,%d,%d,%d", &minX, &minZ, &maxX, &maxZ)
				x, z := int32(math.Floor(float64(a.Position[0])/16)), int32(math.Floor(float64(a.Position[2])/16))
				if x < minX || x > maxX || z < minZ || z > maxZ {
					continue
				}
			}
			store.PutActor(a)
		}
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded %d captured chunks; focus=%v", n, focus)
	names := []string{}
	for name := range blockNames {
		names = append(names, name)
	}
	sort.Strings(names)
	t.Logf("Captured block types: %v", names)
	if n == 0 {
		t.Fatal("empty capture")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	page, err := store.Page(ctx, scene.PageRequest{Limit: 1, Focus: focus, TerrainOnly: os.Getenv("CRAFT_BOX_CAPTURE_TERRAIN_ONLY") == "1"}, nil)
	t.Logf("first page %s; chunks=%d; err=%v", time.Since(start), len(page.Chunks), err)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Chunks) != 1 || len(page.Chunks[0].Parts) == 0 {
		t.Fatal("first page missing geometry")
	}
	if path := os.Getenv("CRAFT_BOX_CAPTURE_PAGE_OUT"); path != "" {
		data, err := json.Marshal(page)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	app := NewApp()
	app.scenes = map[string]*scene.Store{"capture-replay": store}
	server := httptest.NewServer(app.mapWindowHandler("replay-capability"))
	defer server.Close()
	child := &MapWindowApp{ctx: context.Background(), address: server.URL + "/map-window", token: "replay-capability"}
	request := scene.PageRequest{Limit: 8, Focus: focus, TerrainOnly: os.Getenv("CRAFT_BOX_CAPTURE_TERRAIN_ONLY") == "1"}
	pages := []scene.Page{}
	seen := map[[2]int32]bool{}
	triangles := 0
	start = time.Now()
	for {
		next, err := child.GetScenePage("capture-replay", request)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, next)
		for _, c := range next.Chunks {
			key := [2]int32{c.X, c.Z}
			if seen[key] {
				t.Fatal("duplicate chunk", key)
			}
			seen[key] = true
			for _, p := range c.Parts {
				triangles += p.Indices / 3
			}
		}
		request.Target = next.Sequence
		request.Cursor = next.Next
		request.AtlasRevision = next.AtlasRevision
		if next.Complete {
			request.After = next.Sequence
			request.Target = 0
			break
		}
	}
	if len(seen) != n || triangles == 0 {
		t.Fatalf("missing geometry: %d/%d chunks; %d triangles", len(seen), n, triangles)
	}
	unchanged, err := child.GetScenePage("capture-replay", request)
	if err != nil || len(unchanged.Chunks) != 0 || len(unchanged.Patches) != 0 {
		t.Fatal("unchanged scene failed", err)
	}
	t.Logf("HTTP replay: %d chunks; %d triangles; %d pages; %s", len(seen), triangles, len(pages), time.Since(start))
	t.Logf("Unresolved image references: %v", catalog.MissingTexturePaths())
	if path := os.Getenv("CRAFT_BOX_CAPTURE_PAGES_OUT"); path != "" {
		data, err := json.Marshal(pages)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
