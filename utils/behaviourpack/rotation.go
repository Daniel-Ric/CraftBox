package behaviourpack

import (
	"encoding/json"
	"regexp"
	"strconv"
)

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

	tiltX, tiltZ := -1, -1
	for _, r := range rotations {
		x, z := norm(r[0]), norm(r[2])
		if tiltX == -1 {
			tiltX, tiltZ = x, z
		} else if x != tiltX || z != tiltZ {
			tiltX, tiltZ = 0, 0
			break
		}
	}
	yaw := map[string]int{}
	for v, r := range rotations {
		if norm(r[0]) == tiltX && norm(r[2]) == tiltZ {
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
