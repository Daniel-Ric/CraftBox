package scene

import "strings"

func connectedBoxes(m *blockModel, neighbours [4]*blockModel) []box {
	boxes := append([]box{}, m.boxes...)
	for i, n := range neighbours {
		axis := 0
		if i >= 2 {
			axis = 2
		}
		gate := m.connect == "fence" && strings.Contains(n.name, "fence_gate") && axis != n.gateAxis
		if !n.full && n.connect != m.connect && !gate {
			continue
		}
		if m.connect == "fence" && n.connect == "fence" && strings.Contains(m.name, "nether_brick") != strings.Contains(n.name, "nether_brick") {
			continue
		}
		positive := i%2 == 0
		appendArm := func(low, high, width float32) {
			b := box{min: [3]float32{.5 - width/2, low, .5 - width/2}, max: [3]float32{.5 + width/2, high, .5 + width/2}}
			if positive {
				b.max[axis] = 1
			} else {
				b.min[axis] = 0
			}
			boxes = append(boxes, b)
		}
		switch m.connect {
		case "fence":
			appendArm(.375, .5625, .125)
			appendArm(.75, .9375, .125)
		case "wall":
			appendArm(0, .8125, .5)
		case "pane":
			appendArm(0, 1, .125)
		}
	}
	return boxes
}

func emitBoxQuad(p *Part, q [4][3]float32, normal, offset [3]float32, t tile, face int) {
	start := len(p.UVs)
	emitQuad(p, q, normal, offset, t)
	for i, v := range q {
		var u, w float32
		switch face {
		case 0:
			u, w = 1-v[2], 1-v[1]
		case 1:
			u, w = v[2], 1-v[1]
		case 2:
			u, w = v[0], v[2]
		case 3:
			u, w = v[0], 1-v[2]
		case 4:
			u, w = v[0], 1-v[1]
		case 5:
			u, w = 1-v[0], 1-v[1]
		}
		p.UVs[start+i*2] = (float32(t.x) + u*16) / atlasSize
		p.UVs[start+i*2+1] = (float32(t.y) + w*16) / atlasSize
	}
}
