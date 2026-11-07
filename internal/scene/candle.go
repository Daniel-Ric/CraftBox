package scene

import "strings"

func (c *Catalog) candleModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	count := max(1, min(4, number(props["candles"])+1))
	positions := [][3]float32{{8, 8, 6}}
	switch count {
	case 2:
		positions = [][3]float32{{6, 7, 6}, {10, 8, 3}}
	case 3:
		positions = [][3]float32{{6, 7, 6}, {9, 6, 3}, {8, 10, 5}}
	case 4:
		positions = [][3]float32{{6, 6, 6}, {10, 6, 3}, {6, 10, 5}, {10, 10, 4}}
	}
	tex := c.texture(name, 0)
	lit := flag(props["lit"])
	if lit {
		if variants := c.terrainVariants[strings.TrimPrefix(name, "minecraft:")]; len(variants) > 1 && variants[1] != "" {
			tex = c.materialTexture(map[string]any{"texture": variants[1]}, 16, 16)
		}
	}
	for _, p := range positions {
		x, z, h := p[0]/16, p[1]/16, p[2]/16
		start := len(m.quads)
		appendModelBox(&m, box{[3]float32{x - 1.0/16, 0, z - 1.0/16}, [3]float32{x + 1.0/16, h, z + 1.0/16}}, tex)
		for face := 0; face < 6; face++ {
			v0, v1 := 16-p[2], float32(16)
			if face == 2 || face == 3 {
				v0, v1 = 6, 8
			}
			cropQuadUV(&m.quads[start+face], 0, v0, 2, v1, 16, 16)
		}

		for axis := 0; axis < 2; axis++ {
			points := [4][3]float32{{x - .5/16, h, z}, {x + .5/16, h, z}, {x + .5/16, h + 1.0/16, z}, {x - .5/16, h + 1.0/16, z}}
			if axis == 1 {
				for i := range points {
					points[i][0], points[i][2] = x, z+points[i][0]-x
				}
			}
			q := texturedQuad(points, tex)
			cropQuadUV(&q, 0, 5, 1, 6, 16, 16)
			m.quads = append(m.quads, q)
		}
		if lit {
			c.appendCandleFlame(&m, x, h+1.0/16, z)
		}
	}
	return m
}

func cropQuadUV(q *modelQuad, u0, v0, u1, v1, tw, th float32) {
	tex := q.texture
	for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
		q.uv[i] = [2]float32{(float32(tex.x) + (u0+(u1-u0)*uv[0])/tw*float32(tex.w)) / atlasSize, (float32(tex.y) + (v0+(v1-v0)*uv[1])/th*float32(tex.h)) / atlasSize}
	}
}

func (c *Catalog) appendCandleFlame(m *blockModel, x, y, z float32) {

	tex := c.materialTexture(map[string]any{"texture": "textures/particle/particles"}, 128, 128)
	if tex.missing {
		return
	}
	for axis := 0; axis < 2; axis++ {
		points := [4][3]float32{{x - 1.0/16, y - .5/16, z}, {x + 1.0/16, y - .5/16, z}, {x + 1.0/16, y + 2.5/16, z}, {x - 1.0/16, y + 2.5/16, z}}
		if axis == 1 {
			for i := range points {
				points[i][0], points[i][2] = x, z+points[i][0]-x
			}
		}
		q := texturedQuad(points, tex)
		q.mode = "unlit"
		cropQuadUV(&q, 0, 24, 8, 32, 128, 128)
		m.quads = append(m.quads, q)
	}
}

func (c *Catalog) candleCakeModel(name string, props map[string]any) blockModel {
	cake := c.cakeModel("minecraft:cake", nil)
	candle := c.candleModel(strings.TrimSuffix(name, "_cake"), map[string]any{"candles": int32(0), "lit": props["lit"]})
	for _, q := range candle.quads {
		for i := range q.points {
			q.points[i][1] += .5
		}
		cake.quads = append(cake.quads, q)
	}
	cake.name = name
	return cake
}
