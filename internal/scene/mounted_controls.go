package scene

import (
	"github.com/go-gl/mathgl/mgl32"
	"strings"
)

func appendModelBox(m *blockModel, b box, tex tile) {
	for face, d := range directions {
		var points [4][3]float32
		for i, p := range corners[face] {
			for a := range p {
				points[i][a] = b.min[a]
				if p[a] == 1 {
					points[i][a] = b.max[a]
				}
			}
		}
		part := newPart(m.mode)
		emitBoxQuad(part, points, [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, [3]float32{}, tex, face)
		q := modelQuad{points: points, normal: [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, texture: tex, mode: m.mode}
		for i := range q.uv {
			q.uv[i] = [2]float32{part.UVs[i*2], part.UVs[i*2+1]}
		}
		m.quads = append(m.quads, q)
	}
}
func mountModel(m *blockModel, face int) {
	rotation := [3]float32{}
	switch face {
	case 0:
		rotation[0] = 180
	case 2:
		rotation[0] = -90
	case 3:
		rotation[0] = 90
	case 4:
		rotation[2] = 90
	case 5:
		rotation[2] = -90
	}
	transformQuads(m, rotationMatrix([3]float32{.5, .5, .5}, rotation))
}
func mountedFace(props map[string]any) int {
	if face, ok := props["facing_direction"].(string); ok {
		switch face {
		case "down":
			return 0
		case "up":
			return 1
		case "north":
			return 2
		case "south":
			return 3
		case "west":
			return 4
		case "east":
			return 5
		}
	}
	return number(props["facing_direction"])
}
func (c *Catalog) buttonModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	depth := float32(2.0 / 16)
	if flag(props["button_pressed_bit"]) {
		depth = 1.0 / 16
	}
	appendModelBox(&m, box{[3]float32{5.0 / 16, 0, 6.0 / 16}, [3]float32{11.0 / 16, depth, 10.0 / 16}}, c.texture(name, 2))
	face := mountedFace(props)
	if face == 4 || face == 5 {
		// A roll onto an X-facing wall would turn the six-pixel width vertical.
		transformQuads(&m, rotationMatrix([3]float32{.5, .5, .5}, [3]float32{0, 90, 0}))
	}
	mountModel(&m, face)
	return m
}
func (c *Catalog) leverModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	appendModelBox(&m, box{[3]float32{5.0 / 16, 0, 4.0 / 16}, [3]float32{11.0 / 16, 3.0 / 16, 12.0 / 16}}, c.texture("minecraft:cobblestone", 2))
	handle := blockModel{mode: "mask"}
	tex := c.materialTexture(map[string]any{"texture": "lever"}, 16, 16)
	appendModelBox(&handle, box{[3]float32{7.0 / 16, 3.0 / 16, 7.0 / 16}, [3]float32{9.0 / 16, 13.0 / 16, 9.0 / 16}}, tex)
	for qi := range handle.quads {
		for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
			v0, v1 := float32(6), float32(16)
			if qi == 2 || qi == 3 {
				v1 = 8
			}
			handle.quads[qi].uv[i] = [2]float32{(float32(tex.x) + 7 + uv[0]*2) / atlasSize, (float32(tex.y) + v0 + uv[1]*(v1-v0)) / atlasSize}
		}
	}
	angle := float32(-40)
	if flag(props["open_bit"]) {
		angle = 40
	}
	direction, _ := props["lever_direction"].(string)
	if direction == "north" || direction == "south" || direction == "west" || direction == "east" {
		angle = -angle
	}
	transformQuads(&handle, rotationMatrix([3]float32{.5, 3.0 / 16, .5}, [3]float32{angle, 0, 0}))
	m.quads = append(m.quads, handle.quads...)
	face := 1
	switch {
	case strings.HasPrefix(direction, "down_"):
		face = 0
	case direction == "north":
		face = 2
	case direction == "south":
		face = 3
	case direction == "west":
		face = 4
	case direction == "east":
		face = 5
	}
	if strings.HasSuffix(direction, "east_west") {
		transformQuads(&m, rotationMatrix([3]float32{.5, 0, .5}, [3]float32{0, 90, 0}))
	}
	// Keep plates tall and powered switch travel vertical on every wall.
	wallTurn := float32(0)
	switch face {
	case 3:
		wallTurn = 180
	case 4:
		wallTurn = 90
	case 5:
		wallTurn = -90
	}
	if wallTurn != 0 {
		transformQuads(&m, rotationMatrix([3]float32{.5, 0, .5}, [3]float32{0, wallTurn, 0}))
	}
	mountModel(&m, face)
	return m
}
func (c *Catalog) hookModel(name string, props map[string]any) blockModel {
	m := blockModel{name: name, mode: "opaque"}
	appendModelBox(&m, box{[3]float32{6.0 / 16, 0, 4.0 / 16}, [3]float32{10.0 / 16, 2.0 / 16, 12.0 / 16}}, c.texture(name, 3))
	hook := blockModel{mode: "mask"}
	tex := c.texture(name, 2)
	hook.quads = append(hook.quads, texturedQuad([4][3]float32{{.25, .125, .5}, {.75, .125, .5}, {.75, .625, .5}, {.25, .625, .5}}, tex))
	second := hook.quads[0]
	hook.quads = append(hook.quads, second)
	rotation := rotationMatrix([3]float32{.5, .125, .5}, [3]float32{0, 90, 0})
	for i, p := range hook.quads[1].points {
		v := rotation.Mul4x1(mgl32.Vec4{p[0], p[1], p[2], 1})
		hook.quads[1].points[i] = [3]float32{v[0], v[1], v[2]}
	}
	hook.quads[1].normal = [3]float32{1, 0, 0}
	angle := float32(-25)
	if flag(props["attached_bit"]) {
		angle = 0
	}
	if flag(props["powered_bit"]) {
		angle = 25
	}
	transformQuads(&hook, rotationMatrix([3]float32{.5, .125, .5}, [3]float32{angle, 0, 0}))
	m.quads = append(m.quads, hook.quads...)
	face := []int{3, 4, 2, 5}[number(props["direction"])&3]
	if d, ok := props["minecraft:cardinal_direction"].(string); ok {
		for i, n := range []string{"south", "west", "north", "east"} {
			if d == n {
				face = []int{3, 4, 2, 5}[i]
			}
		}
	}
	m.tripwireHookDirection = map[int]int{3: 2, 4: 1, 2: 3, 5: 0}[face]
	mountModel(&m, face)
	return m
}
