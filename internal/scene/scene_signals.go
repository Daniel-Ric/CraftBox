package scene

import "strings"

// Bounded aggregate metadata; never infer a holiday from a block's color.
func sceneBlockSignals(name string) []string {
	n := strings.ToLower(name)
	keys := []string{}
	add := func(key string, yes bool) {
		if yes {
			keys = append(keys, key)
		}
	}
	add("snow", strings.Contains(n, "snow") && !strings.Contains(n, "snowman"))
	add("ice", strings.HasSuffix(n, ":ice") || strings.Contains(n, "_ice"))
	add("foliage", strings.Contains(n, "leaves") || strings.Contains(n, "azalea"))
	add("flowers", strings.Contains(n, "flower") || strings.Contains(n, "tulip") || strings.Contains(n, "poppy") || strings.Contains(n, "orchid") || strings.Contains(n, "petals"))
	add("pumpkins", strings.Contains(n, "pumpkin"))
	add("jackolanterns", strings.Contains(n, "lit_pumpkin") || strings.Contains(n, "jack_o_lantern"))
	add("webs", strings.HasSuffix(n, ":web") || strings.Contains(n, "cobweb"))
	add("skulls", strings.Contains(n, "skull") || strings.Contains(n, "skeleton_head"))
	add("eggs", strings.Contains(n, "easter_egg") || strings.Contains(n, "decorated_egg"))
	add("rabbits", strings.Contains(n, "bunny") || strings.Contains(n, "rabbit_decor"))
	add("gifts", strings.Contains(n, "present_") || strings.Contains(n, "gift_") || strings.HasSuffix(n, ":present"))
	add("festive", strings.Contains(n, "christmas") || strings.Contains(n, "wreath") || strings.Contains(n, "candy_cane"))
	add("autumn", strings.Contains(n, "autumn_leaves") || strings.Contains(n, "fall_leaves"))
	return keys
}
