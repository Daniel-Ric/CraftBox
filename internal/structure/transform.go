package structure

import (
	"context"
	"fmt"
	"math"
	"strings"
)

func transformedPoint(p, size Vec3, turns int, mirror string) (Vec3, Vec3) {
	if mirror == "x" {
		p[0] = size[0] - 1 - p[0]
	}
	if mirror == "z" {
		p[2] = size[2] - 1 - p[2]
	}
	for i := 0; i < turns; i++ {
		p = Vec3{size[2] - 1 - p[2], p[1], p[0]}
		size = Vec3{size[2], size[1], size[0]}
	}
	return p, size
}
func cardinal(v string, turns int, mirror string) string {
	if mirror == "x" {
		switch v {
		case "east":
			v = "west"
		case "west":
			v = "east"
		}
	}
	if mirror == "z" {
		switch v {
		case "north":
			v = "south"
		case "south":
			v = "north"
		}
	}
	directions := []string{"north", "east", "south", "west"}
	for i, s := range directions {
		if v == s {
			return directions[(i+turns)%4]
		}
	}
	return v
}
func stateTransform(s State, turns int, mirror string) (State, []string) {
	out := s
	out.States = cloneMap(s.States)
	warnings := map[string]bool{}
	for key := range s.States {
		k := strings.TrimPrefix(key, "minecraft:")
		if strings.HasPrefix(k, "connection_") || strings.HasPrefix(k, "wall_connection_type_") {
			delete(out.States, key)
		}
	}
	for key, value := range s.States {
		k := strings.TrimPrefix(key, "minecraft:")
		handled := false
		switch k {
		case "cardinal_direction", "torch_facing_direction", "block_face":
			if str, ok := value.(string); ok {
				out.States[key] = cardinal(str, turns, mirror)
				handled = true
			}
		case "facing_direction":
			if str, ok := value.(string); ok {
				out.States[key] = cardinal(str, turns, mirror)
				handled = true
			} else if n, ok := integer(value); ok && n >= 0 && n < 6 {
				dirs := []string{"down", "up", "north", "south", "west", "east"}
				dir := cardinal(dirs[n], turns, mirror)
				for i, v := range dirs {
					if v == dir {
						out.States[key] = int32(i)
						if _, byteType := value.(uint8); byteType {
							out.States[key] = uint8(i)
						}
					}
				}
				handled = true
			}
		case "direction", "weirdo_direction", "coral_direction":
			if n, ok := value.(int32); ok && n >= 0 && n < 4 {
				dirs := []string{"south", "west", "north", "east"}
				if k == "weirdo_direction" {
					dirs = []string{"east", "west", "south", "north"}
				}
				dir := cardinal(dirs[n], turns, mirror)
				for i, v := range dirs {
					if v == dir {
						out.States[key] = int32(i)
					}
				}
				handled = true
			}
		case "pillar_axis", "axis":
			if str, ok := value.(string); ok {
				if turns%2 == 1 {
					if str == "x" {
						str = "z"
					} else if str == "z" {
						str = "x"
					}
				}
				out.States[key] = str
				handled = true
			}
		case "ground_sign_direction", "sixteen_way_rotation":
			if n, ok := value.(int32); ok && n >= 0 && n < 16 {
				if mirror == "x" {
					n = (16 - n) % 16
				}
				if mirror == "z" {
					n = (8 - n + 16) % 16
				}
				out.States[key] = (n + int32(turns*4)) % 16
				handled = true
			}
		case "door_hinge_bit":
			if mirror != "" {
				if n, ok := value.(uint8); ok {
					out.States[key] = n ^ 1
				}
			}
			handled = true
		case "rail_direction":
			if n, ok := value.(int32); ok && n >= 0 && n <= 9 {
				// Straight, ascending and the four corner rail shapes.
				r := int(n)
				if mirror == "x" {
					r = []int{0, 1, 3, 2, 4, 5, 7, 6, 9, 8}[r]
				}
				if mirror == "z" {
					r = []int{0, 1, 2, 3, 5, 4, 9, 8, 7, 6}[r]
				}
				for i := 0; i < turns; i++ {
					r = []int{1, 0, 5, 4, 2, 3, 7, 8, 9, 6}[r]
				}
				out.States[key] = int32(r)
				handled = true
			}
		case "corner", "corner_and_cardinal_direction":
			if str, ok := value.(string); ok {
				if mirror != "" {
					str = strings.ReplaceAll(str, "left", "$right")
					str = strings.ReplaceAll(str, "right", "left")
					str = strings.ReplaceAll(str, "$left", "right")
				}
				out.States[key] = str
				handled = true
			}
		}
		for _, dir := range []string{"north", "east", "south", "west"} {
			if k == "connection_"+dir || k == "wall_connection_type_"+dir {
				prefix := strings.TrimSuffix(key, dir)
				out.States[prefix+cardinal(dir, turns, mirror)] = value
				handled = true
			}
		}
		if !handled && (strings.Contains(k, "direction") || strings.Contains(k, "facing") || strings.Contains(k, "rotation") || strings.Contains(k, "orientation") || k == "axis" || k == "huge_mushroom_bits") {
			warnings[s.Name+": unsupported orientation state "+key] = true
		}
	}
	return out, uniqueWarnings(warnings)
}
func Transform(ctx context.Context, d *Document, turns int, mirror string) (*Document, []string, error) {
	turns = ((turns % 4) + 4) % 4
	if mirror != "" && mirror != "x" && mirror != "z" {
		return nil, nil, fmt.Errorf("Mirror axis must be X or Z")
	}
	_, size := transformedPoint(Vec3{}, d.Size, turns, mirror)
	out, err := New(d.Name, size)
	if err != nil {
		return nil, nil, err
	}
	out.Origin = d.Origin
	out.Packs = append([]Dependency(nil), d.Packs...)
	out.RootExtra = cloneMap(d.RootExtra)
	out.StructureExtra = cloneMap(d.StructureExtra)
	out.PaletteExtra = cloneMap(d.PaletteExtra)
	out.Default = d.Default
	out.DefaultKnown = d.DefaultKnown
	out.Palette = make([]State, len(d.Palette))
	warnings := map[string]bool{}
	for i, s := range d.Palette {
		state, w := stateTransform(s, turns, mirror)
		out.Palette[i] = state
		for _, v := range w {
			warnings[v] = true
		}
	}
	err = each(ctx, d.Bounds(), func(p Vec3) error {
		q, _ := transformedPoint(p, d.Size, turns, mirror)
		c := d.Cell(p)
		if c.Layers != out.Default.Layers || c.Known != out.DefaultKnown {
			out.Set(q, c)
		}
		if data := d.BlockData[Key(p)]; data != nil {
			copy := cloneMap(data)
			if be, ok := copy["block_entity_data"].(map[string]any); ok {
				transformNBTFacing(be, turns, mirror)
				if px, ok := integer(be["pairx"]); ok {
					if pz, ok := integer(be["pairz"]); ok {
						pair, _ := transformedPoint(Vec3{px - d.Origin[0], p[1], pz - d.Origin[2]}, d.Size, turns, mirror)
						be["pairx"], be["pairz"] = pair[0]+d.Origin[0], pair[2]+d.Origin[2]
					}
				}
			}
			out.BlockData[Key(q)] = copy
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	out.Anchor, _ = transformedPoint(d.Anchor, d.Size, turns, mirror)
	for _, r := range d.Regions {
		a, _ := transformedPoint(r.Bounds.Min, d.Size, turns, mirror)
		b, _ := transformedPoint(r.Bounds.Max, d.Size, turns, mirror)
		r.Bounds = (Bounds{a, b}).Normalized()
		out.Regions = append(out.Regions, r)
	}
	for _, e := range d.Entities {
		copy := cloneMap(e)
		p, ok := entityPosition(copy)
		if !ok {
			continue
		}
		sx, sz := float64(d.Size[0]), float64(d.Size[2])
		if mirror == "x" {
			p[0] = sx - p[0]
		}
		if mirror == "z" {
			p[2] = sz - p[2]
		}
		for i := 0; i < turns; i++ {
			p[0], p[2] = sz-p[2], p[0]
			sx, sz = sz, sx
		}
		setEntityPosition(copy, p)
		if a := reflectSlice(copy["Rotation"]); len(a) == 2 {
			yaw, _ := number(a[0])
			pitch, _ := number(a[1])
			if mirror == "x" {
				yaw = -yaw
			}
			if mirror == "z" {
				yaw = 180 - yaw
			}
			yaw = math.Mod(yaw+float64(turns*90)+360, 360)
			copy["Rotation"] = []float32{float32(yaw), float32(pitch)}
		}
		out.Entities = append(out.Entities, copy)
	}
	return out, uniqueWarnings(warnings), nil
}
func transformNBTFacing(data map[string]any, turns int, mirror string) {
	for _, k := range []string{"facing", "Facing"} {
		if v, ok := data[k]; ok {
			state, _ := stateTransform(State{Name: "minecraft:entity", States: map[string]any{"facing_direction": v}}, turns, mirror)
			data[k] = state.States["facing_direction"]
		}
	}
}
