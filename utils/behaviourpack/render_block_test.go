package behaviourpack

import (
	"encoding/json"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestCapturedBlockRenderDocumentHasMinecraftBlockRoot(t *testing.T) {
	raw, err := BlockRenderJSON(protocol.BlockEntry{Name: "goe_frm:advanced_wood_barn_gate_right", Properties: map[string]any{
		"components": map[string]any{"minecraft:geometry": map[string]any{"identifier": "geometry.goe_frm.advanced_wood_barn_gate_right"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		FormatVersion string         `json:"format_version"`
		Block         MinecraftBlock `json:"minecraft:block"`
	}
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.FormatVersion == "" || doc.Block.Description.Identifier != "goe_frm:advanced_wood_barn_gate_right" {
		t.Fatalf("capture produced an invalid block document: %s", raw)
	}
}
