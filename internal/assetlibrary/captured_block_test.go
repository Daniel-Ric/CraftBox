package assetlibrary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestCapturedBlockBindingsReachModelPreview(t *testing.T) {
	l := New(t.TempDir())
	id := strings.Repeat("1", 64)
	root, _ := l.RootFor(id)
	files := map[string]string{
		"models/gate.geo.json":          `{"format_version":"1.12.0","minecraft:geometry":[{"description":{"identifier":"geometry.test.gate","texture_width":16,"texture_height":16},"bones":[{"name":"gate","cubes":[{"origin":[0,0,0],"size":[16,16,1],"uv":[0,0]}]}]}]}`,
		"textures/terrain_texture.json": `{"texture_data":{"test_wood":{"textures":"textures/wood"}}}`,
		"textures/wood.png":             "fixture",
	}
	for file, raw := range files {
		target := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(raw), 0644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := behaviourpack.BlockRenderJSON(protocol.BlockEntry{Name: "test:gate", Properties: map[string]any{"components": map[string]any{
		"minecraft:geometry":           map[string]any{"identifier": "geometry.test.gate"},
		"minecraft:material_instances": map[string]any{"materials": map[string]any{"*": map[string]any{"texture": "test_wood", "render_method": "alpha_test"}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = l.SaveBlockDefinitions(id, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	objects, err := l.Objects(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range objects {
		if o.Kind == "block" && o.Name == "test:gate" && o.Textures["*"] == "textures/wood.png" && o.GeometryID == "geometry.test.gate" {
			return
		}
	}
	t.Fatalf("captured geometry/material binding missing: %#v", objects)
}

func TestFarmPackBundledGateBinding(t *testing.T) {
	l := New(t.TempDir())
	docs, _, err := l.blockDefinitionDocuments("1286867b533b3bc1dd22a731fffc8725149e73f87eae6781679b7af42912391b")
	if err != nil {
		t.Fatal(err)
	}
	doc := docs["captured_blocks/goe_frm/advanced_wood_barn_gate_right.json"]
	components := objectMap(objectMap(doc["minecraft:block"])["components"])
	geo := objectString(objectMap(components["minecraft:geometry"])["identifier"])
	materials := objectMap(components["minecraft:material_instances"])
	if geo != "geometry.goe_frm.advanced_wood_barn_gate_right" || objectString(objectMap(materials["*"])["texture"]) != "goe_frm_wood_fence" {
		t.Fatalf("incorrect recorded gate binding: %#v", components)
	}
}
