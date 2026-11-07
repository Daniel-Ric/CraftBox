package blocktraits

import "github.com/Daniel-Ric/CraftBox/utils/nbtconv"

func MultiBlockEnabled(trait map[string]any) bool {
	states, ok := trait["enabled_states"].(map[string]any)
	if !ok {
		return false
	}
	for _, key := range []string{"multi_block", "minecraft:multi_block", "multi_block_part", "minecraft:multi_block_part"} {
		if nbtconv.Flag(states[key]) {
			return true
		}
	}
	return false
}

// MultiBlockParts translates the network part_count into the supported number
// of parts. Bedrock defaults to two and clamps this trait to two through four.
func MultiBlockParts(trait map[string]any) int32 {
	parts, ok := trait["part_count"].(int32)
	if !ok {
		parts = 2
	}
	return max(int32(2), min(int32(4), parts))
}
