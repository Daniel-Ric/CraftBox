package behaviourpack

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"reflect"
	"testing"
)

func TestBlockIntegerPlacementFlagsNormalizeWithoutPanic(t *testing.T) {
	for _, value := range []any{uint8(63), int32(63)} {
		block, _ := parseBlock(protocol.BlockEntry{Name: "test:filter", Properties: map[string]any{"components": map[string]any{"minecraft:placement_filter": map[string]any{"conditions": []any{map[string]any{"allowed_faces": value, "block_filter": []any{}}}}}}})
		conditions := block.Components["minecraft:placement_filter"].(map[string]any)["conditions"].([]any)
		faces := conditions[0].(map[string]any)["allowed_faces"]
		if !reflect.DeepEqual(faces, []string{"down", "up", "north", "south", "west", "east"}) {
			t.Fatalf("lost placement mask: %v", faces)
		}
	}
}
