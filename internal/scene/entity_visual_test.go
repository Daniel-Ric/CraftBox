package scene

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"image/color"
	"testing"
	"testing/fstest"
)

func TestCapturedActorRenderingAndPetExclusion(t *testing.T) {
	c := NewCatalog()
	err := c.AddPack(fstest.MapFS{
		"entity/test.json":             {Data: []byte(`{"minecraft:client_entity":{"description":{"identifier":"test:statue","geometry":{"first":"geometry.statue"},"textures":{"red":"textures/red","green":"textures/green"},"render_controllers":["controller.render.test"]}}}`)},
		"render_controllers/test.json": {Data: []byte(`{"render_controllers":{"controller.render.test":{"arrays":{"textures":{"Array.skins":["Texture.red","Texture.green"]}},"geometry":"Geometry.first","textures":["Array.skins[query.variant]"],"part_visibility":[{"hat":false}],"ignore_lighting":true}}}`)},
		"models/entity/test.json":      {Data: []byte(`{"minecraft:geometry":[{"description":{"identifier":"geometry.statue","texture_width":16,"texture_height":16},"bones":[{"name":"root","cubes":[{"origin":[-4,0,-4],"size":[8,16,8],"uv":[0,0]}]},{"name":"hat","cubes":[{"origin":[-4,20,-4],"size":[8,8,8],"uv":[0,0]}]}]}]}`)},
		"textures/red.png":             {Data: materialImage(16, 16, color.NRGBA{255, 0, 0, 255})},
		"textures/green.png":           {Data: materialImage(16, 16, color.NRGBA{0, 255, 0, 255})},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := Actor{ID: 1, Type: "test:statue", Position: [3]float32{-1, 20, 3}, Scale: 1, Variant: 1}
	m := c.actorModel(a)
	if len(m.quads) != 6 {
		t.Fatalf("expected visible body only, got %d faces", len(m.quads))
	}
	for _, q := range m.quads {
		if q.texture.missing || q.mode != "unlit" {
			t.Fatal("lost selected texture or unlit material")
		}
		if c.atlas.NRGBAAt(q.texture.x, q.texture.y).G != 255 {
			t.Fatal("variant selected wrong skin")
		}
	}
	for _, typ := range []string{"minecraft:player", "cubecraft:buddy.epicdragon", "cubecraft:buddy_waterdragon"} {
		a.Type = typ
		if actorAllowed(a) {
			t.Fatal("player/pet allowed", typ)
		}
	}
	a.Type = "cubecraft:lobby.epic_dragon"
	if !actorAllowed(a) {
		t.Fatal("lobby decoration filtered")
	}
	a.OwnerID = 123
	if actorAllowed(a) {
		t.Fatal("owned companion allowed")
	}
}

func TestTerrainRenderOmitsActorsWithoutPollutingLiveCache(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	s.PutActor(Actor{ID: 1, Type: "test:hologram", Scale: 1, Name: "Hidden in renders", NameVisible: true})
	live, e := s.Page(context.Background(), PageRequest{}, nil)
	if e != nil || len(live.Chunks[0].Parts) == 0 {
		t.Fatal("live label missing", e)
	}
	terrain, e := s.Page(context.Background(), PageRequest{TerrainOnly: true}, nil)
	if e != nil || len(terrain.Chunks[0].Parts) != 0 {
		t.Fatal("render contains entity", e)
	}
	live, e = s.Page(context.Background(), PageRequest{}, nil)
	if e != nil || len(live.Chunks[0].Parts) == 0 {
		t.Fatal("render cache hid live entities", e)
	}
}

func TestActorDeltaCopiesMetadataAndErasesOldChunk(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	a := Actor{ID: 1, Type: "test:hologram", Position: [3]float32{-1, 10, 0}, Scale: 1, Name: "§aHallo äöü", NameVisible: true, Properties: map[string]any{"state": 1}}
	s.PutActor(a)
	a.Properties["state"] = 99
	first, err := s.Snapshot(0, nil)
	if err != nil || len(first.Chunks) != 1 || len(first.Chunks[0].Parts) != 1 {
		t.Fatalf("actor-only chunk missing: %v", err)
	}
	if first.Chunks[0].Parts[0].Mode != "billboard" {
		t.Fatal("hologram does not face viewer")
	}
	if numeric(s.actors[1].Properties["state"]) != 1 {
		t.Fatal("snapshot aliases mutable metadata")
	}
	a.Position[0] = 17
	s.PutActor(a)
	delta, err := s.Snapshot(first.Sequence, nil)
	if err != nil || len(delta.Chunks) != 2 {
		t.Fatal("moving actor left stale chunk", err)
	}
	a.Type = "minecraft:player"
	s.PutActor(a)
	last, err := s.Snapshot(delta.Sequence, nil)
	if err != nil || len(last.Chunks) != 1 || len(last.Chunks[0].Parts) != 0 {
		t.Fatal("player replacement left geometry", err)
	}
}

func TestSignTextFollowsBoardAndBackFace(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	c := NewCatalog()
	s := newTestStore(t, reg, c)
	ch := chunk.New(reg, cube.Range{0, 15})
	rid, _ := reg.StateToRuntimeID("minecraft:wall_sign", map[string]any{"facing_direction": int32(4)})
	ch.SetBlock(2, 2, 2, 0, rid)
	s.Put(world.ChunkPos{}, ch, map[cube.Pos]map[string]any{{2, 2, 2}: {"id": "Sign", "FrontText": map[string]any{"Text": "Vorne", "IgnoreLighting": uint8(1)}, "BackText": map[string]any{"Text": "Hinten", "SignTextColor": int32(-16777216)}}})
	snap, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	unlit := 0
	for _, p := range snap.Chunks[0].Parts {
		if p.Mode == "unlit" {
			unlit += len(p.Indices)
		}
	}
	if unlit != 6 {
		t.Fatal("front text missing")
	}

	tile, _ := c.textTile("äöü", color.NRGBA{255, 255, 255, 255}, false)
	pixels := 0
	for y := 0; y < tile.h; y++ {
		for x := 0; x < tile.w; x++ {
			if c.atlas.NRGBAAt(tile.x+x, tile.y+y).A > 0 {
				pixels++
			}
		}
	}
	if pixels == 0 {
		t.Fatal("UTF8 text blank")
	}
}

func TestHologramUpdatesReuseAtlasAndKeepInitialStyle(t *testing.T) {
	c := NewCatalog()
	first, _ := c.textTileSlot("Hello §cRed", color.NRGBA{255, 255, 255, 255}, true, "1")
	allocated := c.next
	history := len(c.changes)
	for i := 0; i < 100; i++ {
		next, _ := c.textTileSlot(fmt.Sprintf("Hello §c%03d", i), color.NRGBA{255, 255, 255, 255}, true, "1")
		if next.x != first.x || next.y != first.y {
			t.Fatal("text update leaked atlas space")
		}
	}
	if c.next != allocated {
		t.Fatal("dynamic text exhausts atlas")
	}
	if len(c.changes) != history {
		t.Fatal("dynamic text accumulates unbounded patch history")
	}
	c.textTileSlot("Hello §cRed", color.NRGBA{255, 255, 255, 255}, true, "1")
	white := false
	for y := 0; y < first.h; y++ {
		for x := 0; x < first.w/2; x++ {
			p := c.atlas.NRGBAAt(first.x+x, first.y+y)
			white = white || (p.R > 200 && p.G > 200 && p.B > 200)
		}
	}
	if !white {
		t.Fatal("width measurement changed leading text style")
	}
}

func TestPlayerLinksHideAttachedEntitiesRegardlessOfArrivalOrder(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	a := Actor{ID: 2, Type: "test:cosmetic", Scale: 1, Name: "attached", NameVisible: true}
	s.PutActor(a)
	s.LinkActor(1, 2, true)
	s.PutActor(Actor{ID: 1, Type: "minecraft:player", Scale: 1})
	snapshot, e := s.Snapshot(0, nil)
	if e != nil {
		t.Fatal(e)
	}
	for _, ch := range snapshot.Chunks {
		if len(ch.Parts) != 0 {
			t.Fatal("attached model escaped player filter")
		}
	}
	s.LinkActor(1, 2, false)
	snapshot, e = s.Snapshot(0, nil)
	if e != nil || len(snapshot.Chunks[0].Parts) != 1 {
		t.Fatal("detached decoration not restored", e)
	}
	a = ActorFromNBT(3, map[string]any{"identifier": "test:pet", "OwnerNew": int64(-9223372036854775807)})
	if actorAllowed(a) {
		t.Fatal("negative owner ID lost")
	}
	a = ActorFromNBT(4, map[string]any{"identifier": "test:attachment", "CraftBoxExcludeFrom3D": uint8(1)})
	if actorAllowed(a) {
		t.Fatal("saved player attachment exclusion lost")
	}
}

func TestNativeSignAndChainGeometry(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	rid, ok := reg.StateToRuntimeID("minecraft:standing_sign", map[string]any{"ground_sign_direction": int32(0)})
	if !ok {
		t.Fatal("fixture")
	}
	m := s.model(rid)
	if m.full || len(m.quads) == 0 {
		t.Fatal("standing sign still a cube")
	}
	rid, ok = reg.StateToRuntimeID("minecraft:iron_chain", map[string]any{"pillar_axis": "y"})
	if !ok {
		t.Fatal("chain fixture")
	}
	m = s.model(rid)
	if len(m.quads) != 2 {
		t.Fatal("chain must use two explicit cutout planes")
	}
}

func TestActorModelFrontMatchesBedrockYaw(t *testing.T) {
	c := NewCatalog()
	c.AddPack(fstest.MapFS{"textures/nose.png": {Data: materialImage(16, 16, color.NRGBA{255, 255, 255, 255})}})
	c.actors["test:npc"] = actorDefinition{Geometry: map[string]string{"default": "geometry.nose"}, Textures: map[string]string{"default": "textures/nose"}, Controllers: []any{"controller.render.nose"}}
	c.renderControllers["controller.render.nose"] = map[string]any{"geometry": "Geometry.default", "textures": []any{"Texture.default"}}
	g := blockGeometry{}
	g.Description.Width = 16
	g.Description.Height = 16
	g.Bones = []geometryBone{{Name: "nose", Cubes: []geometryCube{{Origin: [3]float32{-1, 0, -10}, Size: [3]float32{2, 2, 2}, UV: json.RawMessage(`[0,0]`)}}}}
	c.geometries["geometry.nose"] = g
	for i, want := range [][2]float32{{0, 1}, {-1, 0}, {0, -1}, {1, 0}} {
		m := c.actorModel(Actor{Type: "test:npc", Yaw: float32(i) * 90, Scale: 1})
		x, z := float32(0), float32(0)
		for _, q := range m.quads {
			for _, p := range q.points {
				x += p[0]
				z += p[2]
			}
		}
		if x*want[0]+z*want[1] <= 0 {
			t.Fatalf("yaw %d faces away from Minecraft direction", i*90)
		}
	}
}
