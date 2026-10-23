package scene

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"
)

func TestGrassAtlasGetsTintAndColoredPackIsPreserved(t *testing.T) {
	for _, tc := range []struct {
		pixel     color.NRGBA
		wantGreen bool
	}{{color.NRGBA{R: 200, G: 200, B: 200, A: 255}, true}, {color.NRGBA{R: 220, G: 30, B: 10, A: 255}, false}} {
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				img.SetNRGBA(x, y, tc.pixel)
			}
		}
		var buf bytes.Buffer
		png.Encode(&buf, img)
		c := NewCatalog()
		err := c.AddPack(fstest.MapFS{"blocks.json": {Data: []byte(`{"grass":{"textures":{"up":"grass_top"}}}`)}, "textures/blocks/grass_top.png": {Data: buf.Bytes()}})
		if err != nil {
			t.Fatal(err)
		}
		tile := c.texture("minecraft:grass_block", 2)
		pixel := c.atlas.NRGBAAt(tile.x, tile.y)
		if tc.wantGreen && pixel.G <= pixel.R {
			t.Fatal("grayscale grass stayed gray")
		}
		if !tc.wantGreen && pixel != tc.pixel {
			t.Fatal("colored custom texture was retinted")
		}
	}
}

func TestGrassSideAlphaIsTintMaskInsteadOfTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			pixel := color.NRGBA{R: 150, G: 108, B: 74, A: 0}
			if y < 4 {
				pixel = color.NRGBA{R: 169, G: 169, B: 169, A: 255}
			}
			img.SetNRGBA(x, y, pixel)
		}
	}
	var data bytes.Buffer
	png.Encode(&data, img)
	c := NewCatalog()
	c.AddPack(fstest.MapFS{"blocks.json": {Data: []byte(`{"grass":{"textures":{"side":"grass_side"}}}`)}, "textures/blocks/grass_side.png": {Data: data.Bytes()}})
	tile := c.texture("minecraft:grass_block", 0)
	dirt := c.atlas.NRGBAAt(tile.x, tile.y+8)
	grass := c.atlas.NRGBAAt(tile.x, tile.y)
	if tile.alpha || dirt != (color.NRGBA{R: 150, G: 108, B: 74, A: 255}) || grass.G <= grass.R {
		t.Fatalf("grass alpha mask misread: dirt=%v grass=%v transparent=%v", dirt, grass, tile.alpha)
	}
}

func TestCatalogIgnoresMojangMetadataAndResolvesFaces(t *testing.T) {
	c := NewCatalog()
	err := c.AddPack(fstest.MapFS{"blocks.json": {Data: []byte(`{"format_version":"1.21.0", "stone":{"textures":{"up":"top","side":"side"}}}`)}, "textures/terrain_texture.json": {Data: []byte(`{"texture_data":{"top":{"textures":"textures/top"},"side":{"textures":"textures/side"}}}`)}})
	if err != nil {
		t.Fatal(err)
	}
	if c.blocks["stone"]["up"] != "top" || c.blocks["stone"]["side"] != "side" {
		t.Fatal("lost per-face textures")
	}
}

func TestCustomMaterialInstancesResolveFaceAndTransparency(t *testing.T) {
	c := NewCatalog()
	c.SetBlockComponents("test:panel", map[string]any{"minecraft:material_instances": map[string]any{"materials": map[string]any{"*": map[string]any{"texture": "panel_sides", "render_method": "blend"}, "up": map[string]any{"texture": "panel_top", "render_method": "alpha_test"}}}})
	if c.blocks["test:panel"]["up"] != "panel_top" || c.blocks["test:panel"]["east"] != "panel_sides" || c.modes["test:panel"] != "blend" {
		t.Fatal("custom materials lost face mapping or alpha mode")
	}
}

func TestPlantDoesNotOccludeAndUnequalWaterKeepsExposedFaces(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := newTestStore(t, reg, NewCatalog())
	poppy, ok := reg.StateToRuntimeID("minecraft:poppy", map[string]any{})
	if !ok {
		t.Fatal("missing plant fixture")
	}
	if m := s.model(poppy); m.full || !m.cross {
		t.Fatal("poppy is a solid occluder")
	}
	high, ok := reg.StateToRuntimeID("minecraft:water", map[string]any{"liquid_depth": int32(0)})
	if !ok {
		t.Fatal("missing water fixture")
	}
	low, _ := reg.StateToRuntimeID("minecraft:water", map[string]any{"liquid_depth": int32(7)})
	c := chunk.New(reg, cube.Range{0, 15})
	c.SetBlock(0, 1, 0, 0, high)
	c.SetBlock(1, 1, 0, 0, low)
	s.Put(world.ChunkPos{0, 0}, c)
	snap, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	part := snap.Chunks[0].Parts[0]
	found := false
	for i := 0; i < len(part.Positions); i += 3 {
		if part.Positions[i] == 1 && part.Positions[i+1] > 1.5 && part.Normals[i] == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("high water lost its exposed side next to shallow water")
	}
}

func TestSurfaceCullingUpdatesAndSnapshotIsolation(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	a := chunk.New(reg, cube.Range{0, 15})
	a.SetBlock(15, 1, 0, 0, stone)
	s := newTestStore(t, reg, NewCatalog())
	s.Put(world.ChunkPos{0, 0}, a)
	first, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Chunks) != 1 || len(first.Chunks[0].Parts[0].Positions) != 72 {
		t.Fatalf("single block needs six quads: %+v", first)
	}
	b := chunk.New(reg, cube.Range{0, 15})
	b.SetBlock(0, 1, 0, 0, stone)
	s.Put(world.ChunkPos{1, 0}, b)
	next, err := s.Snapshot(first.Sequence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Chunks) != 2 {
		t.Fatalf("neighbor invalidation: %d", len(next.Chunks))
	}
	for _, c := range next.Chunks {
		if len(c.Parts[0].Positions) != 60 {
			t.Fatal("internal chunk border face was not culled")
		}
	}

	b.SetBlock(0, 1, 0, 0, reg.AirRuntimeID())
	all, _ := s.Snapshot(0, nil)
	if len(all.Chunks[1].Parts) == 0 {
		t.Fatal("store retained mutable capture pointer")
	}
	s.Put(world.ChunkPos{1, 0}, b)
	removed, _ := s.Snapshot(next.Sequence, nil)
	if len(removed.Chunks) != 2 {
		t.Fatal("empty chunk update lost")
	}
}

func TestRegionCutsExposeFacesAndGLBContainsGeometry(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	s := newTestStore(t, reg, NewCatalog())
	for x := int32(0); x < 2; x++ {
		c := chunk.New(reg, cube.Range{0, 15})
		c.SetBlock(uint8(15-x*15), 1, 0, 0, stone)
		s.Put(world.ChunkPos{x, 0}, c)
	}
	snap, err := s.Snapshot(0, func(x, z int32) bool { return x == 0 })
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Chunks) != 1 || len(snap.Chunks[0].Parts[0].Positions) != 72 {
		t.Fatal("region boundary must be closed")
	}
	data, err := EncodeGLB(snap)
	if err != nil {
		t.Fatal(err)
	}
	if string(data[:4]) != "glTF" || binary.LittleEndian.Uint32(data[4:]) != 2 || int(binary.LittleEndian.Uint32(data[8:])) != len(data) {
		t.Fatal("invalid GLB container")
	}
	size := binary.LittleEndian.Uint32(data[12:])
	var doc map[string]any
	if err := json.Unmarshal(data[20:20+size], &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc["meshes"].([]any)) != 1 || len(doc["accessors"].([]any)) < 3 {
		t.Fatal("missing geometry")
	}
}

func newTestStore(t *testing.T, reg world.BlockRegistry, c *Catalog) *Store {
	t.Helper()
	s := NewStore(reg, c)
	t.Cleanup(s.Close)
	return s
}

func TestCaptureUpdatesAndSnapshotsCanRunConcurrently(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	s := newTestStore(t, reg, NewCatalog())
	c := chunk.New(reg, cube.Range{0, 15})
	c.SetBlock(0, 1, 0, 0, stone)
	s.Put(world.ChunkPos{}, c)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 30; i++ {
			c.SetBlock(1, 1, 0, 0, stone)
			s.Put(world.ChunkPos{}, c)
		}
	}()
	after := uint64(0)
	for i := 0; i < 10; i++ {
		snap, err := s.Snapshot(after, nil)
		if err != nil {
			t.Fatal(err)
		}
		if snap.Sequence < after {
			t.Fatal("snapshot sequence regressed")
		}
		after = snap.Sequence
	}
	<-done
	snap, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Chunks) != 1 || len(snap.Chunks[0].Parts[0].Positions) != 120 {
		t.Fatal("final adjacent-block geometry is inconsistent")
	}
}
