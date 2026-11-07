package scene

import "strings"

func connectedBoxes(m *blockModel, neighbours [4]*blockModel, above ...*blockModel) []box {
	if m.connect == "stairs" {
		return connectedStairs(m, neighbours)
	}
	if m.connect == "wall" && m.wallHeights == nil {
		copy := *m
		var heights [4]float32
		var top *blockModel
		if len(above) > 0 {
			top = above[0]
		}
		for i, n := range neighbours {
			if !connectionAllowed(m, n, i) {
				continue
			}
			heights[i] = .8125
			if top != nil {
				for _, b := range connectionSurfaceBoxes(top, 1, false) {
					if b.min[1] > 0 {
						continue
					}
					xOverlap, zOverlap := b.min[0] < .75 && b.max[0] > .25, b.min[2] < .75 && b.max[2] > .25
					tall := false
					switch i {
					case 0:
						tall = b.max[0] > .75 && zOverlap
					case 1:
						tall = b.min[0] < .25 && zOverlap
					case 2:
						tall = xOverlap && b.max[2] > .75
					case 3:
						tall = xOverlap && b.min[2] < .25
					}
					if tall {
						heights[i] = 1
						break
					}
				}
			}
		}
		post := wallPostForConnections(heights, top)
		if m.wallPost != nil {
			post = *m.wallPost
		}
		copy.wallHeights = &heights
		copy.wallPost = &post
		return connectedBoxes(&copy, neighbours)
	}
	if m.connect == "wall" && m.wallHeights != nil {
		boxes := []box{}
		var top *blockModel
		if len(above) > 0 {
			top = above[0]
		}
		post := wallPostForConnections(*m.wallHeights, top)
		if m.wallPost != nil {
			post = *m.wallPost
		}
		if post {
			boxes = append(boxes, box{[3]float32{.25, 0, .25}, [3]float32{.75, 1, .75}})
		}
		for i, h := range *m.wallHeights {
			if h == 0 {
				continue
			}
			b := box{[3]float32{.3125, 0, .3125}, [3]float32{.6875, h, .6875}}
			axis := 0
			if i >= 2 {
				axis = 2
			}
			if i%2 == 0 {
				b.max[axis] = 1
			} else {
				b.min[axis] = 0
			}
			boxes = append(boxes, b)
		}
		return boxes
	}
	boxes := append([]box{}, m.boxes...)
	for i, n := range neighbours {
		if n == nil {
			n = &emptyBlockModel
		}
		axis := 0
		if i >= 2 {
			axis = 2
		}
		connected := connectionAllowed(m, n, i)
		if m.connections != nil {
			connected = m.connections[i]
		}
		if !connected {
			continue
		}
		if m.connections == nil && m.connect == "fence" && n.connect == "fence" && strings.Contains(m.name, "nether_brick") != strings.Contains(n.name, "nether_brick") {
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
			appendArm(0, .8125, .375)
		case "pane":
			appendArm(0, 1, .125)
		}
	}
	return boxes
}

func wallPostForConnections(heights [4]float32, top *blockModel) bool {
	if top != nil && top.connect != "wall" {
		for _, b := range connectionSurfaceBoxes(top, 1, false) {
			if b.min[1] <= 0 && b.min[0] <= .25 && b.max[0] >= .75 && b.min[2] <= .25 && b.max[2] >= .75 {
				return true
			}
		}
	}
	count := 0
	for _, h := range heights {
		if h > 0 {
			count++
		}
	}
	straight := count == 2 && (heights[0] > 0 && heights[1] > 0 || heights[2] > 0 && heights[3] > 0)
	return !straight || top != nil && (top.connect == "wall" && top.wallPost != nil && *top.wallPost || strings.Contains(top.name, "lantern") || strings.HasSuffix(top.name, "torch") || strings.HasSuffix(top.name, "_sign"))
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
