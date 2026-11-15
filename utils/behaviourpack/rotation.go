package behaviourpack

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
)

const cardinalDirection = "minecraft:cardinal_direction"

var cardinalRotations = []struct {
	direction string
	y         float32
}{
	{"north", 0},
	{"west", 90},
	{"south", 180},
	{"east", -90},
}

const cardinalDefault = "north"
const cardinalYOffset = 180

func addCardinalRotation(b *MinecraftBlock, version string) bool {
	if version < "1.20.20" || !hasGeometry(b) {
		return false
	}
	for name := range b.Description.Traits {
		if name == "minecraft:placement_direction" || name == "minecraft:placement_position" {
			return false
		}
	}
	if _, ok := b.Description.States[cardinalDirection]; ok {
		return false
	}
	for _, p := range b.Permutations {
		if strings.Contains(p.Condition, "cardinal_direction") || strings.Contains(p.Condition, "facing_direction") {
			return false
		}
		if _, ok := p.Components["minecraft:transformation"]; ok {
			return false
		}
	}

	if b.Description.Traits == nil {
		b.Description.Traits = make(map[string]Trait)
	}
	b.Description.Traits["minecraft:placement_direction"] = Trait{
		"enabled_states":    []string{cardinalDirection},
		"y_rotation_offset": cardinalYOffset,
	}

	base, _ := b.Components["minecraft:transformation"].(map[string][]float32)
	for _, r := range cardinalRotations {
		transform := map[string][]float32{"rotation": {0, r.y, 0}}
		if base != nil {
			transform = maps.Clone(base)
			rotation := []float32{0, 0, 0}
			copy(rotation, base["rotation"])
			rotation[1] += r.y
			transform["rotation"] = rotation
		}
		b.Permutations = append(b.Permutations, permutation{
			Condition:  fmt.Sprintf("q.block_state('%s') == '%s'", cardinalDirection, r.direction),
			Components: map[string]any{"minecraft:transformation": transform},
		})
	}
	return true
}
func hasGeometry(b *MinecraftBlock) bool {
	if _, ok := b.Components["minecraft:geometry"]; ok {
		return true
	}
	for _, p := range b.Permutations {
		if _, ok := p.Components["minecraft:geometry"]; ok {
			return true
		}
	}
	return false
}

var stateCondition = regexp.MustCompile(`^\s*(?:q|query)\.block_(?:state|property)\(\s*'([^']+)'\s*\)\s*==\s*(\d+)\s*$`)

type placeRotation struct {
	State string         `json:"state"`
	Yaw   map[string]int `json:"yaw,omitempty"`
	Axis  map[string]int `json:"axis,omitempty"`
}

func findPlaceRotation(b *MinecraftBlock) *placeRotation {
	for name := range b.Description.Traits {
		if name == "minecraft:placement_direction" || name == "minecraft:placement_position" {
			return nil
		}
	}
	rotations := map[int][3]float32{}
	state := ""
	for _, p := range b.Permutations {
		m := stateCondition.FindStringSubmatch(p.Condition)
		if m == nil {
			continue
		}
		if state != "" && m[1] != state {
			return nil
		}
		state = m[1]
		value, _ := strconv.Atoi(m[2])
		var rot [3]float32
		if t, ok := p.Components["minecraft:transformation"].(map[string][]float32); ok {
			copy(rot[:], t["rotation"])
		}
		rotations[value] = rot
	}
	if state == "" || len(rotations) < 2 {
		return nil
	}
	_, property := splitNamespace(state)
	switch property {
	case "facing_direction", "cardinal_direction", "direction", "orientation", "rotation", "axis", "pillar_axis":
	default:
		return nil
	}
	norm := func(d float32) int { return ((int(d) % 360) + 360) % 360 }

	yaw := map[string]int{}
	for v, r := range rotations {
		if norm(r[0]) == 0 && norm(r[2]) == 0 {
			key := strconv.Itoa(norm(r[1]))
			if previous, dup := yaw[key]; dup && previous <= v {
				continue
			}
			yaw[key] = v
		}
	}
	if len(yaw) >= 2 {
		return &placeRotation{State: state, Yaw: yaw}
	}

	axis := map[string]int{}
	for v, r := range rotations {
		x, y, z := norm(r[0]), norm(r[1]), norm(r[2])
		switch {
		case y != 0:
			return nil
		case x == 0 && z == 0:
			axis["y"] = v
		case (x == 90 || x == 270) && z == 0:
			axis["z"] = v
		case x == 0 && (z == 90 || z == 270):
			axis["x"] = v
		}
	}
	if len(axis) >= 2 {
		return &placeRotation{State: state, Axis: axis}
	}
	return nil
}

const placementScriptFile = "rotate_on_place.js"

func placementScript(blocks map[string]*placeRotation) string {
	table, _ := json.Marshal(blocks)
	return `import { world } from "@minecraft/server";
const rotations = ` + string(table) + `;
try {
world.afterEvents.playerPlaceBlock.subscribe(({ block, player }) => {
  const r = rotations[block.typeId];
  if (!r) return;
  let value;
  if (r.yaw) {
    const deg = (((-Math.round(player.getRotation().y / 90) * 90) % 360) + 360) % 360;
    value = r.yaw[String(deg)];
  } else if (r.axis) {
    const v = player.getViewDirection();
    const axis = Math.abs(v.y) > 0.7 ? "y" : Math.abs(v.x) > Math.abs(v.z) ? "x" : "z";
    value = r.axis[axis];
  }
  if (value === undefined) return;
  try {
    block.setPermutation(block.permutation.withState(r.state, value));
  } catch (e) {
    console.warn("rotate " + block.typeId + ": " + e);
  }
});
} catch (e) {
  console.warn("rotate on place: " + e);
}
`
}
