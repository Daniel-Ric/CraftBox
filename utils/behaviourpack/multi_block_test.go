package behaviourpack

import (
	"reflect"
	"testing"
)

func TestExportFriendWorldMultiBlockTrait(t *testing.T) {
	traits := processTraits([]any{map[string]any{
		"name": "minecraft:multi_block", "enabled_states": map[string]any{"multi_block": uint8(1)},
		"part_count": int32(3), "direction": int32(1), "place_behavior": int32(1),
	}})
	want := Trait{"enabled_states": []string{"minecraft:multi_block_part"}, "parts": int32(3), "direction": "up"}
	if !reflect.DeepEqual(traits["minecraft:multi_block"], want) {
		t.Fatalf("got %#v, want %#v", traits["minecraft:multi_block"], want)
	}
}

func TestDisabledMultiBlockTraitIsNotExported(t *testing.T) {
	for _, states := range []any{nil, map[string]any{"multi_block": uint8(0)}} {
		traits := processTraits([]any{map[string]any{"name": "minecraft:multi_block", "enabled_states": states, "part_count": int32(3)}})
		if _, ok := traits["minecraft:multi_block"]; ok {
			t.Fatal("disabled trait was exported")
		}
	}
}
