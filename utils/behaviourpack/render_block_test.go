package behaviourpack

import (
	"encoding/json"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"testing"
)

func TestBlockRenderJSONPreservesCapturedMaterialMappings(t *testing.T) {
	raw, err := BlockRenderJSON(protocol.BlockEntry{Name: "test:mapped", Properties: map[string]any{"components": map[string]any{"minecraft:material_instances": map[string]any{"mappings": map[string]any{"up": "top"}, "materials": map[string]any{"*": map[string]any{"texture": "side"}, "top": map[string]any{"texture": "top_texture"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	var block MinecraftBlock
	if err = json.Unmarshal(raw, &block); err != nil {
		t.Fatal(err)
	}
	materials := block.Components["minecraft:material_instances"].(map[string]any)
	if materials["up"] != "top" {
		t.Fatalf("captured mapping disappeared: %v", materials)
	}
}
