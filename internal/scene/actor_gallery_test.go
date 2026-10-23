package scene

import (
	"context"
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordedActorGallery(t *testing.T) {
	root := os.Getenv("CRAFT_BOX_CAPTURE_ASSETS")
	out := os.Getenv("CRAFT_BOX_ACTOR_GALLERY_OUT")
	if root == "" || out == "" {
		t.Skip("local visual capture")
	}
	catalog, e := LoadCatalog("../../assets/vanilla-3d")
	if e != nil {
		t.Fatal(e)
	}
	packs, e := os.ReadDir(filepath.Join(root, "resource_packs"))
	if e != nil {
		t.Fatal(e)
	}
	for _, pack := range packs {
		if pack.IsDir() {
			if e = catalog.AddPack(os.DirFS(filepath.Join(root, "resource_packs", pack.Name()))); e != nil {
				t.Fatal(e)
			}
		}
	}
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, catalog)
	stone, _ := reg.StateToRuntimeID("minecraft:smooth_stone", nil)
	for x := int32(0); x < 2; x++ {
		for z := int32(0); z < 2; z++ {
			ch := chunk.New(reg, cube.Range{0, 31})
			for i := 0; i < 16; i++ {
				for j := 0; j < 16; j++ {
					ch.SetBlock(uint8(i), 0, uint8(j), 0, stone)
				}
			}
			entities := map[cube.Pos]map[string]any{}
			if x == 0 && z == 1 {
				for i := 0; i < 4; i++ {
					rid, ok := reg.StateToRuntimeID("minecraft:standing_sign", map[string]any{"ground_sign_direction": int32(i * 4)})
					if !ok {
						t.Fatal("sign")
					}
					ch.SetBlock(uint8(2+i*3), 1, 10, 0, rid)
					entities[cube.Pos{2 + i*3, 1, 26}] = map[string]any{"id": "Sign", "FrontText": map[string]any{"Text": "§aHallo\näöü ß\n§lCraftBox", "IgnoreLighting": uint8(1)}, "BackText": map[string]any{"Text": "Rückseite", "SignTextColor": int32(-16777216)}}
				}
			}
			if x == 1 && z == 1 {
				for i, axis := range []string{"y", "x", "z"} {
					rid, ok := reg.StateToRuntimeID("minecraft:iron_chain", map[string]any{"pillar_axis": axis})
					if !ok {
						t.Fatal("chain")
					}
					for j := 0; j < 4; j++ {
						ch.SetBlock(uint8(2+i*4), int16(2+j), 10, 0, rid)
					}
				}
			}
			s.Put(world.ChunkPos{x, z}, ch, entities)
		}
	}
	raw, e := os.ReadFile("../../tmp/entity-corrections/packet-actors.json")
	if e != nil {
		t.Fatal(e)
	}
	var actors []Actor
	if e = json.Unmarshal(raw, &actors); e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	n := 0
	labels := 0
	for _, a := range actors {
		if !actorAllowed(a) {
			continue
		}
		if a.Type == "minecraft:armor_stand" {
			if a.Name != "" && a.NameVisible && labels < 4 {
				a.ID = 1000 + int64(labels)
				a.Position = [3]float32{4 + float32(labels)*7, 7, 23}
				s.PutActor(a)
				labels++
			}
			continue
		}
		if strings.HasPrefix(a.Type, "minecraft:") || seen[a.Type] {
			continue
		}
		seen[a.Type] = true
		if len(catalog.actorModel(a).quads) == 0 {
			continue
		}
		a.ID = 100 + int64(n)
		a.Position = [3]float32{3 + float32(n%5)*6, 1, 3 + float32(n/5)*6}
		a.Yaw = 0
		s.PutActor(a)
		n++
		if n == 20 {
			break
		}
	}

	for _, typ := range []string{"cubecraft:shopskin.dragon", "cubecraft:shopskin.mrkitty"} {
		a := Actor{ID: 500 + int64(n), Type: typ, Scale: 1, Position: [3]float32{4 + float32(n%5)*6, 1, 22}, Yaw: 0}
		m := catalog.actorModel(a)
		if len(m.quads) == 0 {
			t.Fatal("missing static pose", typ)
		}
		s.PutActor(a)
		n++
	}
	pages := []Page{}
	request := PageRequest{Limit: 8}
	for {
		page, e := s.Page(context.Background(), request, nil)
		if e != nil {
			t.Fatal(e)
		}
		pages = append(pages, page)
		if page.Complete {
			break
		}
		request.Target = page.Sequence
		request.Cursor = page.Next
		request.AtlasRevision = page.AtlasRevision
	}
	raw, e = json.Marshal(pages)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(out+".json", raw, 0600); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Snapshot(0, nil)
	if e != nil {
		t.Fatal(e)
	}
	glb, e := EncodeGLB(snap)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(out+".glb", glb, 0600); e != nil {
		t.Fatal(e)
	}
	if missing := catalog.MissingTexturePaths(); len(missing) > 0 {
		t.Fatal("missing textures", missing)
	}
	t.Logf("rendered %d recorded model types and %d holograms; atlas entries=%d; GLB bytes=%d", n, labels, len(catalog.tiles), len(glb))
}
