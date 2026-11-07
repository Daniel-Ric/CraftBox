package scene

import (
	"math"
	"slices"
	"strings"
)

// Attachment tests need the real boundary surfaces, including native models
// represented by textured quads instead of collision-style boxes.
func connectionSurfaceBoxes(m *blockModel, axis int, positive bool) []box {
	if m == nil {
		return nil
	}
	if m.full {
		return []box{unit}
	}
	out := append([]box{}, m.boxes...)
	boundary, normal := float32(0), float32(-1)
	if positive {
		boundary, normal = 1, 1
	}
	for _, q := range m.quads {
		if q.mode != "opaque" || math.Abs(float64(q.normal[axis]-normal)) > .0001 {
			continue
		}
		b := box{min: [3]float32{1, 1, 1}}
		valid := true
		for _, p := range q.points {
			if math.Abs(float64(p[axis]-boundary)) > .0001 {
				valid = false
				break
			}
			for a := range p {
				b.min[a] = min(b.min[a], p[a])
				b.max[a] = max(b.max[a], p[a])
			}
		}
		if valid {
			out = append(out, b)
		}
	}
	return out
}

func cropConnectionModel(m *blockModel, clipped [4]bool) *blockModel {
	if clipped == [4]bool{} || m.connections == nil && m.wallHeights == nil && m.stairs == nil {
		return m
	}
	copy := *m
	if m.connections != nil {
		connections := *m.connections
		for i, clip := range clipped {
			if clip {
				connections[i] = false
			}
		}
		copy.connections = &connections
	}
	if m.wallHeights != nil {
		heights := *m.wallHeights
		changed := false
		for i, clip := range clipped {
			if clip && heights[i] != 0 {
				heights[i] = 0
				changed = true
			}
		}
		copy.wallHeights = &heights
		if changed {
			copy.wallPost = nil
		}
	}
	if m.stairs != nil && m.stairs.explicit {
		shape := *m.stairs
		shape.explicit, shape.Corner, shape.Inner, shape.Left = false, false, false, false
		copy.stairs = &shape
		copy.boxes = stairBoxes(shape.Stair)
		copy.connect = "stairs"
	}
	return &copy
}

func connectionAllowed(source, neighbour *blockModel, direction int) bool {
	if neighbour == nil {
		return false
	}
	axis := 0
	if direction >= 2 {
		axis = 2
	}
	gate := strings.HasSuffix(neighbour.name, "_fence_gate") || neighbour.name == "minecraft:fence_gate"
	if gate {
		return (source.connect == "fence" || source.connect == "wall") && axis != neighbour.gateAxis
	}
	switch source.connect {
	case "fence":
		if neighbour.connect == "fence" {
			return strings.Contains(source.name, "nether_brick") == strings.Contains(neighbour.name, "nether_brick")
		}
	case "pane", "wall":
		if neighbour.connect == "pane" || neighbour.connect == "wall" {
			return true
		}
	}
	return fullConnectionFace(neighbour, axis, direction%2 != 0)
}

func fullConnectionFace(m *blockModel, axis int, positive bool) bool {
	if m == nil || m.cross || m.fluid != "" || strings.Contains(m.name, "leaves") {
		return false
	}
	if m.full {
		return true
	}
	axes := []int{}
	for i := 0; i < 3; i++ {
		if i != axis {
			axes = append(axes, i)
		}
	}
	type rectangle struct{ u0, v0, u1, v1 float32 }
	rectangles := []rectangle{}
	us, vs := []float32{0, 1}, []float32{0, 1}
	for _, b := range connectionSurfaceBoxes(m, axis, positive) {
		if positive && b.max[axis] < .99999 || !positive && b.min[axis] > .00001 {
			continue
		}
		r := rectangle{max(0, b.min[axes[0]]), max(0, b.min[axes[1]]), min(1, b.max[axes[0]]), min(1, b.max[axes[1]])}
		if r.u0 >= r.u1 || r.v0 >= r.v1 {
			continue
		}
		rectangles = append(rectangles, r)
		us = append(us, r.u0, r.u1)
		vs = append(vs, r.v0, r.v1)
	}
	if len(rectangles) == 0 {
		return false
	}
	slices.Sort(us)
	slices.Sort(vs)
	us = slices.Compact(us)
	vs = slices.Compact(vs)
	for u := 0; u < len(us)-1; u++ {
		for v := 0; v < len(vs)-1; v++ {
			cx, cy := (us[u]+us[u+1])/2, (vs[v]+vs[v+1])/2
			covered := false
			for _, r := range rectangles {
				if cx >= r.u0 && cx <= r.u1 && cy >= r.v0 && cy <= r.v1 {
					covered = true
					break
				}
			}
			if !covered {
				return false
			}
		}
	}
	return true
}
