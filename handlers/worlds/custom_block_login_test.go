package worlds

import (
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"testing"
)

func TestCustomBlockLoginAcceptsNBTIntFlags(t *testing.T) {
	for _, flag := range []any{uint8(1), int32(1), true} {
		t.Run(fmtFlagType(flag), func(t *testing.T) {
			entry := protocol.BlockEntry{Name: "test:login", Properties: map[string]any{
				"traits": []any{map[string]any{"name": "minecraft:placement_direction", "enabled_states": map[string]any{"cardinal_direction": flag}}},
				"components": map[string]any{
					"minecraft:placement_filter": map[string]any{"conditions": []any{map[string]any{"allowed_faces": int32(63), "block_filter": []any{}}}},
					"minecraft:liquid_detection": map[string]any{"detectionRules": []any{map[string]any{"canContainLiquid": flag, "onLiquidTouches": "blocking", "liquidType": "water"}}},
				},
			}}
			bp := behaviourpack.New("login-test")
			bp.AddBlock(entry)
			reg := world.NewBlockRegistry()
			if err := AddCustomBlocks(reg, []protocol.BlockEntry{entry}); err != nil {
				t.Fatal(err)
			}
			reg.Finalize()
			ids := map[uint32]bool{}
			for _, direction := range []string{"north", "east", "south", "west"} {
				if id, ok := reg.StateToRuntimeID(entry.Name, map[string]any{"minecraft:cardinal_direction": direction}); !ok {
					t.Fatal("missing login block state", direction)
				} else {
					ids[id] = true
				}
			}
			if len(ids) != 4 {
				t.Fatal("direction states collapsed to fallback")
			}
			if _, err := behaviourpack.BlockRenderJSON(entry); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func fmtFlagType(v any) string {
	switch v.(type) {
	case uint8:
		return "byte"
	case int32:
		return "int32"
	default:
		return "bool"
	}
}
