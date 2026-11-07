package scene

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

func (c *Catalog) lilyPadModel(name string) blockModel {
	m := blockModel{name: name, mode: "mask"}
	m.quads = []modelQuad{texturedQuad([4][3]float32{{0, 1.0 / 64, 1}, {1, 1.0 / 64, 1}, {1, 1.0 / 64, 0}, {0, 1.0 / 64, 0}}, c.texture(name, 2))}
	return m
}

func (c *Catalog) redstoneTexture(power int) tile {
	return c.redstoneTextureFor(power, false)
}
func (c *Catalog) redstoneTextureFor(power int, line bool) tile {
	power = max(0, min(15, power))
	base := c.texture("minecraft:redstone_wire", 2)
	if line {
		base = c.texture("minecraft:redstone_wire", 3)
	}
	key := fmt.Sprintf("redstone:%d:%d:%d#tint:none#", base.x, base.y, power)
	if tex, ok := c.tiles[key]; ok {
		return tex
	}
	if base.missing {
		return base
	}
	f := float32(power) / 15
	r := f*.6 + .4
	if power == 0 {
		r = .3
	}
	g := max(float32(0), f*f*.7-.5)
	b := max(float32(0), f*f*.6-.7)
	img := image.NewNRGBA(image.Rect(0, 0, base.w, base.h))
	for y := 0; y < base.h; y++ {
		for x := 0; x < base.w; x++ {
			v := c.atlas.NRGBAAt(base.x+x, base.y+y)
			img.SetNRGBA(x, y, color.NRGBA{uint8(float32(v.R) * r), uint8(float32(v.G) * g), uint8(float32(v.B) * b), v.A})
		}
	}
	return c.addTileSized(key, img, false, base.w, base.h)
}
func (c *Catalog) redstoneModel(props map[string]any) blockModel {
	m := blockModel{name: "minecraft:redstone_wire", mode: "mask", connect: "redstone"}
	var sides [4]string
	explicit := false
	for i, key := range []string{"redstone_east", "redstone_west", "redstone_south", "redstone_north"} {
		if value, ok := props[key]; ok {
			explicit = true
			sides[i], _ = value.(string)
			if sides[i] == "" && flag(value) {
				sides[i] = "side"
			}
		}
	}
	if explicit {
		m.wireConnections = &sides
	} else {
		sides = [4]string{"side", "side", "side", "side"}
	}
	textures := [2]tile{c.redstoneTexture(number(props["redstone_signal"])), c.redstoneTextureFor(number(props["redstone_signal"]), true)}
	m.wireTextures = &textures
	m.quads = redstoneQuads(textures[0], sides, textures[1])
	return m
}

func redstoneQuads(tex tile, sides [4]string, lines ...tile) []modelQuad {
	const lo, hi, y = float32(4.0 / 16), float32(12.0 / 16), float32(1.0 / 64)
	var on [4]bool
	climbing := false
	for i, s := range sides {
		on[i] = s != "" && s != "none"
		climbing = climbing || s == "up"
	}
	straightX := on == [4]bool{true, true, false, false}
	straightZ := on == [4]bool{false, false, true, true}
	if !climbing && (on == [4]bool{true, true, true, true} || len(lines) > 0 && (straightX || straightZ)) {
		if straightX || straightZ {
			tex = lines[0]
		}
		q := texturedQuad([4][3]float32{{0, y, 1}, {1, y, 1}, {1, y, 0}, {0, y, 0}}, tex)
		for i, p := range q.points {
			u, v := p[0], p[2]
			if straightZ {
				u, v = v, u
			}
			q.uv[i] = [2]float32{(float32(tex.x) + u*float32(tex.w)) / atlasSize, (float32(tex.y) + v*float32(tex.h)) / atlasSize}
		}
		return []modelQuad{q}
	}
	quads := []modelQuad{}
	flat := func(x0, z0, x1, z1 float32) {
		q := texturedQuad([4][3]float32{{x0, y, z1}, {x1, y, z1}, {x1, y, z0}, {x0, y, z0}}, tex)
		for i, p := range q.points {
			q.uv[i] = [2]float32{(float32(tex.x) + p[0]*float32(tex.w)) / atlasSize, (float32(tex.y) + p[2]*float32(tex.h)) / atlasSize}
		}
		quads = append(quads, q)
	}
	flat(lo, lo, hi, hi)
	for i, side := range sides {
		if side == "" || side == "none" {
			continue
		}
		switch i {
		case 0:
			flat(hi, lo, 1, hi)
		case 1:
			flat(0, lo, lo, hi)
		case 2:
			flat(lo, hi, hi, 1)
		case 3:
			flat(lo, 0, hi, lo)
		}
		if side != "up" {
			continue
		}
		pts := [4][3]float32{{1 - y, 0, lo}, {1 - y, 0, hi}, {1 - y, 1, hi}, {1 - y, 1, lo}}
		for j, p := range pts {
			switch i {
			case 1:
				p[0] = y
			case 2:
				p[0], p[2] = p[2], p[0]
			case 3:
				p[0], p[2] = p[2], y
			}
			pts[j] = p
		}
		q := texturedQuad(pts, tex)
		for j, p := range q.points {
			u := p[2]
			if i >= 2 {
				u = p[0]
			}
			q.uv[j] = [2]float32{(float32(tex.x) + u*float32(tex.w)) / atlasSize, (float32(tex.y) + (1-p[1])*float32(tex.h)) / atlasSize}
		}
		quads = append(quads, q)
	}
	return quads
}

func redstoneTerminal(m *blockModel) bool {
	if m == nil {
		return false
	}
	n := strings.TrimPrefix(m.name, "minecraft:")
	return m.connect == "redstone" || n == "redstone_wire" || n == "lever" || n == "redstone_block" || strings.Contains(n, "redstone_torch") || strings.Contains(n, "repeater") || strings.Contains(n, "comparator") || strings.HasSuffix(n, "_button") || strings.HasSuffix(n, "pressure_plate") || n == "tripwire_hook"
}
func connectedRedstoneQuads(m *blockModel, neighbours [4]*blockModel, above *blockModel, upper, lower [4]*blockModel) []modelQuad {
	tex, line := m.quads[0].texture, m.quads[0].texture
	if m.wireTextures != nil {
		tex, line = m.wireTextures[0], m.wireTextures[1]
	}
	if m.wireConnections != nil {
		return redstoneQuads(tex, *m.wireConnections, line)
	}
	var sides [4]string
	count := 0
	last := 0
	for i, n := range neighbours {
		if redstoneTerminal(n) {
			sides[i] = "side"
		} else if n != nil && n.full && above != nil && !above.full && upper[i] != nil && upper[i].connect == "redstone" {
			sides[i] = "up"
		} else if (n == nil || !n.full) && lower[i] != nil && lower[i].connect == "redstone" {
			sides[i] = "side"
		}
		if sides[i] != "" {
			count++
			last = i
		}
	}
	if count == 0 {
		sides = [4]string{"side", "side", "side", "side"}
	} else if count == 1 {
		sides[last^1] = "side"
	}
	return redstoneQuads(tex, sides, line)
}
func coralFanName(name string) bool {
	n := strings.TrimPrefix(name, "minecraft:")
	return n == "coral_fan" || n == "coral_fan_dead" || strings.HasPrefix(n, "coral_fan_hang") || strings.HasSuffix(n, "_coral_fan") || strings.HasSuffix(n, "_coral_wall_fan")
}
func (c *Catalog) coralTexture(name string, props map[string]any) tile {
	n := strings.TrimPrefix(name, "minecraft:")
	if n == "coral_fan" || n == "coral_fan_dead" || strings.HasPrefix(n, "coral_fan_hang") {
		index := number(props["coral_color"])
		if col, ok := props["coral_color"].(string); ok {
			for i, label := range []string{"blue", "pink", "purple", "red", "yellow"} {
				if col == label {
					index = i
				}
			}
		}
		if strings.HasPrefix(n, "coral_fan_hang") {
			index = 0
			if flag(props["coral_hang_type_bit"]) {
				index = 1
			}
			if n == "coral_fan_hang2" {
				index += 2
			}
			if n == "coral_fan_hang3" {
				index = 4
			}
		}
		labels := []string{"tube", "brain", "bubble", "fire", "horn"}
		modern := labels[max(0, min(4, index))] + "_coral_fan"
		if n == "coral_fan_dead" || flag(props["dead_bit"]) {
			modern = "dead_" + modern
		}
		return c.texture("minecraft:"+modern, 2)
	}
	return c.texture(name, 2)
}
func (c *Catalog) floorCoralModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "mask"}
	tex := c.coralTexture(name, props)
	for _, pts := range [][4][3]float32{{{0, 1.0 / 64, 0}, {1, 1.0 / 64, 1}, {1, .5, 1}, {0, .5, 0}}, {{1, 1.0 / 64, 0}, {0, 1.0 / 64, 1}, {0, .5, 1}, {1, .5, 0}}} {
		m.quads = append(m.quads, texturedQuad(pts, tex))
	}
	return m
}
func (c *Catalog) hangingVineModel(name string) blockModel {
	m := blockModel{name: name, mode: "mask"}
	tex := c.texture(name, 2)
	for _, pts := range [][4][3]float32{{{0, 0, 0}, {1, 0, 1}, {1, 1, 1}, {0, 1, 0}}, {{1, 0, 0}, {0, 0, 1}, {0, 1, 1}, {1, 1, 0}}} {
		m.quads = append(m.quads, texturedQuad(pts, tex))
	}
	return m
}
func (c *Catalog) lightningRodModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	textureName := strings.Replace(name, "minecraft:waxed_", "minecraft:", 1)
	tex := c.texture(textureName, 2)
	appendSkinBox(&m, box{[3]float32{7.0 / 16, 0, 7.0 / 16}, [3]float32{9.0 / 16, 12.0 / 16, 9.0 / 16}}, tex, [2]float32{0, 0}, 16, 16)
	appendSkinBox(&m, box{[3]float32{6.0 / 16, 12.0 / 16, 6.0 / 16}, [3]float32{10.0 / 16, 1, 10.0 / 16}}, tex, [2]float32{0, 0}, 16, 16)
	// The native rod texture contains a 4x4 cap and a 2x12 shaft strip,
	// rather than an unwrapped cube net. Keep every face within those pixels.
	for qi := range m.quads {
		width, height, v0 := float32(4), float32(4), float32(0)
		if qi < 6 {
			width, height, v0 = 2, 12, 4
			if qi == 2 || qi == 3 {
				height = 2
			}
		}
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			m.quads[qi].uv[i] = [2]float32{(float32(tex.x) + uv[0]*width) / atlasSize, (float32(tex.y) + v0 + uv[1]*height) / atlasSize}
		}
	}
	face := 1
	if props["facing_direction"] != nil {
		face = mountedFace(props)
	}
	mountModel(&m, face)
	return m
}
