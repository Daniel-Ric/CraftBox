package scene

func (c *Catalog) beaconModel() blockModel {
	m := blockModel{name: "minecraft:beacon", mode: "opaque"}
	base := blockModel{mode: "opaque"}
	appendModelBox(&base, box{[3]float32{2.0 / 16, 0, 2.0 / 16}, [3]float32{14.0 / 16, 3.0 / 16, 14.0 / 16}}, c.texture(m.name, 3))
	m.quads = append(m.quads, base.quads...)
	core := blockModel{mode: "unlit"}
	tex := c.texture(m.name, 2)
	appendModelBox(&core, box{[3]float32{3.0 / 16, 3.0 / 16, 3.0 / 16}, [3]float32{13.0 / 16, 13.0 / 16, 13.0 / 16}}, tex)
	// The core carries the complete luminous sprite on each of its small faces.
	for qi := range core.quads {
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			core.quads[qi].uv[i] = [2]float32{(float32(tex.x) + uv[0]*float32(tex.w)) / atlasSize, (float32(tex.y) + uv[1]*float32(tex.h)) / atlasSize}
		}
	}
	m.quads = append(m.quads, core.quads...)
	shell := blockModel{mode: "blend"}
	appendModelBox(&shell, unit, c.texture(m.name, 0))
	m.quads = append(m.quads, shell.quads...)
	return m
}
