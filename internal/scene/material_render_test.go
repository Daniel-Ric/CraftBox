package scene

import (
	"bytes"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
	"testing/fstest"
)

func materialImage(w, h int, pixel color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, pixel)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func TestWaterTextureReceivesBlueTintAndKeepsAlpha(t *testing.T) {
	c := NewCatalog()
	c.AddPack(fstest.MapFS{"textures/blocks/water_still.png": {Data: materialImage(16, 16, color.NRGBA{180, 180, 180, 180})}, "blocks.json": {Data: []byte(`{"water":{"textures":"water_still"}}`)}})
	tile := c.texture("minecraft:water", 2)
	pixel := c.atlas.NRGBAAt(tile.x, tile.y)
	if pixel.B <= pixel.G || pixel.G <= pixel.R || pixel.A != 180 {
		t.Fatalf("water lacks blue tint or lost alpha: %v", pixel)
	}
}

func TestBlockConditionSupportsCapturedStatesWithoutExecutingScripts(t *testing.T) {
	states := map[string]any{"cc:facing": "north", "cc:top_bit": uint8(1), "cc:extended": uint8(1)}
	for _, tc := range []struct {
		expression       string
		value, supported bool
	}{
		{`query.block_state('cc:facing') == 'north' && query.block_state('cc:top_bit') == 1`, true, true},
		{`q.block_state('cc:extended') == true`, true, true},
		{`!(q.block_state('cc:top_bit') == 1) || false`, false, true},
		{`query.block_property('cc:top_bit') % 2 == 1 ? true : false`, true, true},
		{`query.block_state('missing') == 0`, false, false},
		{`(function(){while(true){}})()`, false, false},
	} {
		got, supported := blockCondition(tc.expression, states)
		if supported != tc.supported || (supported && got != tc.value) {
			t.Fatalf("%s: %v %v", tc.expression, got, supported)
		}
	}
}

func TestCustomPermutationsAndBoneRotationPreserveUVAndTextureResolution(t *testing.T) {
	c := NewCatalog()
	err := c.AddPack(fstest.MapFS{
		"models/blocks/test.json": {Data: []byte(`{"minecraft:geometry":[{"description":{"identifier":"geometry.panel","texture_width":32,"texture_height":16},"bones":[{"name":"root","pivot":[0,8,0],"rotation":[0,90,0],"cubes":[{"origin":[-8,0,0],"size":[16,16,0],"uv":{"north":{"uv":[16,0],"uv_size":[16,16],"material_instance":"skin"}}}]}]}]}`)},
		"textures/skin.png":       {Data: materialImage(32, 16, color.NRGBA{200, 20, 10, 255})},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = c.AddBlockDefinition([]byte(`{"description":{"identifier":"test:panel"},"components":{"minecraft:geometry":{"identifier":"geometry.panel"},"minecraft:material_instances":{"skin":{"texture":"missing"}}},"permutations":[{"condition":"q.block_state('facing') == 'west'","components":{"minecraft:material_instances":{"skin":{"texture":"textures/skin","render_method":"alpha_test"}},"minecraft:transformation":{"translation":[0,1,0]}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	m, ok := c.customModel("test:panel", map[string]any{"facing": "west"})
	if !ok || m.approx || len(m.quads) != 1 || m.quads[0].texture.missing {
		t.Fatal("lost state material or geometry")
	}
	q := m.quads[0]
	if q.texture.w != 32 || q.texture.h != 16 {
		t.Fatal("model texture downsampled/cropped", q.texture)
	}
	for _, p := range q.points {
		if math.Abs(float64(p[0]-.5)) > 1e-5 || p[1] < 1 || p[1] > 2 {
			t.Fatal("lost bone pivot or block translation", p)
		}
	}
	uMin, uMax := float32(math.Inf(1)), float32(math.Inf(-1))
	for _, uv := range q.uv {
		uMin = min(uMin, uv[0])
		uMax = max(uMax, uv[0])
	}
	if math.Abs(float64(uMax-uMin-16.0/atlasSize)) > 1e-7 {
		t.Fatal("model UVs stretched across full texture", q.uv)
	}
	if math.Abs(float64(q.normal[0])) < .99 {
		t.Fatal("normal was not rotated", q.normal)
	}
}

func TestBlockTransformationUsesBlockCenterAndExplicitPivots(t *testing.T) {
	for _, components := range []map[string]any{
		{"rotation": []any{90.0, 0.0, 0.0}},
		{"rotation": []any{0.0, 0.0, 90.0}},
	} {
		m := blockTransform(map[string]any{"minecraft:transformation": components})
		for _, v := range [][4]float32{{0, 0, 0, 1}, {1, 1, 1, 1}} {
			p := m.Mul4x1(v)
			for i := 0; i < 3; i++ {
				if p[i] < -1e-5 || p[i] > 1.00001 {
					t.Fatal("rotation moved cube outside block", p)
				}
			}
		}
	}
}

func TestCustomCubeDefaultPivotIsItsCenter(t *testing.T) {
	c := NewCatalog()
	if err := c.AddPack(fstest.MapFS{"models/blocks/test.json": {Data: []byte(`{"minecraft:geometry":[{"description":{"identifier":"geometry.offset"},"bones":[{"name":"root","cubes":[{"origin":[0,0,0],"size":[8,8,8],"rotation":[0,90,0]}]}]}]}`)}}); err != nil {
		t.Fatal(err)
	}
	c.SetBlockComponents("test:cube", map[string]any{"minecraft:geometry": map[string]any{"identifier": "geometry.offset"}})
	m, _ := c.customModel("test:cube", nil)
	for _, q := range m.quads {
		for _, p := range q.points {
			if p[0] < -.0001 || p[0] > .5001 || p[2] < .4999 || p[2] > 1.0001 {
				t.Fatal("cube rotated about model origin", p)
			}
		}
	}
}

func TestMirroredCubeExchangesEastWestMaterials(t *testing.T) {
	c := NewCatalog()
	c.AddPack(fstest.MapFS{
		"models/blocks/mirror.json": {Data: []byte(`{"minecraft:geometry":[{"description":{"identifier":"geometry.mirror"},"bones":[{"name":"root","mirror":true,"cubes":[{"origin":[-8,0,-8],"size":[16,16,16],"uv":{"east":{"uv":[0,0],"material_instance":"red"},"west":{"uv":[0,0],"material_instance":"blue"}}}]}]}]}`)},
		"textures/red.png":          {Data: materialImage(16, 16, color.NRGBA{200, 0, 0, 255})}, "textures/blue.png": {Data: materialImage(16, 16, color.NRGBA{0, 0, 200, 255})},
	})
	c.SetBlockComponents("test:mirror", map[string]any{"minecraft:geometry": map[string]any{"identifier": "geometry.mirror"}, "minecraft:material_instances": map[string]any{"red": map[string]any{"texture": "textures/red"}, "blue": map[string]any{"texture": "textures/blue"}}})
	m, _ := c.customModel("test:mirror", nil)
	if len(m.quads) != 2 {
		t.Fatal("lost mirrored sides")
	}
	for _, q := range m.quads {
		if q.normal[0] > .9 {
			tile := q.texture
			if pixel := c.atlas.NRGBAAt(tile.x, tile.y); pixel.B != 200 {
				t.Fatal("world east/west faces were not exchanged", pixel)
			}
		}
	}
}

func TestLeafTintAcceptsActualMojangNearGrayAndPreservesCutout(t *testing.T) {
	c := NewCatalog()
	data := materialImage(16, 16, color.NRGBA{104, 99, 104, 255})
	if err := c.AddPack(fstest.MapFS{"textures/blocks/leaves_oak.png": {Data: data}, "blocks.json": {Data: []byte(`{"oak_leaves":{"textures":"leaves_oak"}}`)}}); err != nil {
		t.Fatal(err)
	}
	tile := c.texture("minecraft:oak_leaves", 0)
	p := c.atlas.NRGBAAt(tile.x, tile.y)
	if p.G <= p.R || p.G <= p.B {
		t.Fatalf("Mojang oak mask stayed gray: %v", p)
	}
}

func TestLeafGrayMaskGetsTintWithoutRecoloringGoldenDetails(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{150, 150, 150, 255})
		}
	}
	gold := color.NRGBA{210, 175, 30, 255}
	img.SetNRGBA(0, 0, gold)
	c := NewCatalog()
	tile := c.addTile("textures/blocks/leaves_jungle", img, false)
	if p := c.atlas.NRGBAAt(tile.x+1, tile.y); p.G <= p.R || p.G <= p.B {
		t.Fatal("golden details disabled green tint on gray jungle leaves", p)
	}
	if p := c.atlas.NRGBAAt(tile.x, tile.y); p != gold {
		t.Fatal("golden leaf details were recolored", p)
	}
}

func TestCustomMaterialAliasAndDirectImagePath(t *testing.T) {
	c := NewCatalog()
	c.AddPack(fstest.MapFS{"textures/custom/panel.png": {Data: materialImage(32, 16, color.NRGBA{210, 35, 19, 255})}})
	c.SetBlockComponents("test:panel", map[string]any{"minecraft:material_instances": map[string]any{"*": "finish", "finish": map[string]any{"texture": "textures/custom/panel", "render_method": "alpha_test"}}})
	tile := c.texture("test:panel", 0)
	if tile.missing {
		t.Fatal("material alias/direct texture path was ignored")
	}
}

func TestCapturedCustomGeometryUsesItsOwnUVAndDimensions(t *testing.T) {
	c := NewCatalog()
	err := c.AddPack(fstest.MapFS{
		"models/blocks/panel.json":      {Data: []byte(`{"minecraft:geometry":[{"description":{"identifier":"geometry.test.panel","texture_width":32,"texture_height":16},"bones":[{"name":"root","cubes":[{"origin":[-8,0,-1],"size":[16,16,2],"uv":{"north":{"uv":[16,0],"uv_size":[16,16],"material_instance":"face"},"south":{"uv":[0,0],"uv_size":[16,16],"material_instance":"face"}}}]}]}]}`)},
		"textures/panel.png":            {Data: materialImage(32, 16, color.NRGBA{30, 170, 60, 255})},
		"textures/terrain_texture.json": {Data: []byte(`{"texture_data":{"panel":{"textures":"textures/panel"}}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	c.SetBlockComponents("test:panel", map[string]any{"minecraft:geometry": map[string]any{"identifier": "geometry.test.panel"}, "minecraft:material_instances": map[string]any{"materials": map[string]any{"face": map[string]any{"texture": "panel", "render_method": "alpha_test"}, "*": "face"}}})
	reg := world.NewBlockRegistry()
	reg.RegisterBlockState(world.BlockState{Name: "test:panel", Properties: map[string]any{}})
	reg.Finalize()
	rid, _ := reg.StateToRuntimeID("test:panel", map[string]any{})
	s := newTestStore(t, reg, c)
	ch := chunk.New(reg, cube.Range{0, 15})
	ch.SetBlock(0, 0, 0, 0, rid)
	s.Put(world.ChunkPos{0, 0}, ch)
	snapshot, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	mesh := snapshot.Chunks[0]
	if mesh.MissingTextures != 0 || mesh.ApproximateBlocks != 0 || len(mesh.Parts) != 1 || len(mesh.Parts[0].Indices) != 12 {
		t.Fatalf("custom model replaced by generic cube: missing=%d approximate=%d parts=%d", mesh.MissingTextures, mesh.ApproximateBlocks, len(mesh.Parts))
	}
	for i, v := range mesh.Parts[0].Positions {
		if i%3 == 2 && (v < .43 || v > .57) {
			t.Fatalf("lost model depth: %v", v)
		}
	}
}
