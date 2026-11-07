package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"strings"
)

type stairShape struct {
	model.Stair
	explicit bool
}

func stairState(props map[string]any) *stairShape {
	corner, explicit := props["minecraft:corner"].(string)
	facing := cube.Direction((3 - number(props["weirdo_direction"])) & 3)
	for _, d := range cube.Directions() {
		if props["minecraft:cardinal_direction"] == d.String() {
			facing = d
		}
	}
	return &stairShape{Stair: model.Stair{Facing: facing, UpsideDown: flag(props["upside_down_bit"]) || props["minecraft:vertical_half"] == "top", Corner: strings.HasPrefix(corner, "inner_") || strings.HasPrefix(corner, "outer_"), Inner: strings.HasPrefix(corner, "inner_"), Left: strings.HasSuffix(corner, "_left")}, explicit: explicit}
}
func stairBoxes(shape model.Stair) []box {
	var out []box
	for _, b := range shape.BBox(cube.Pos{}, nil) {
		out = append(out, box{min: [3]float32{float32(b.Min()[0]), float32(b.Min()[1]), float32(b.Min()[2])}, max: [3]float32{float32(b.Max()[0]), float32(b.Max()[1]), float32(b.Max()[2])}})
	}
	return out
}
func connectedStairs(m *blockModel, neighbours [4]*blockModel) []box {
	s := m.stairs.Stair
	get := func(d cube.Direction) *blockModel {
		index := map[cube.Direction]int{cube.East: 0, cube.West: 1, cube.South: 2, cube.North: 3}[d]
		return neighbours[index]
	}
	sameHalf := func(n *blockModel) bool { return n != nil && n.stairs != nil && n.stairs.UpsideDown == s.UpsideDown }
	continued := func(d cube.Direction) bool {
		n := get(d)
		return sameHalf(n) && n.stairs.Facing == s.Facing
	}
	left, right := s.Facing.RotateLeft(), s.Facing.RotateRight()
	if n := get(s.Facing); sameHalf(n) {
		if n.stairs.Facing == left && !continued(right) {
			s.Corner = true
			s.Left = true
			return stairBoxes(s)
		}
		if n.stairs.Facing == right && !continued(left) {
			s.Corner = true
			return stairBoxes(s)
		}
	}
	if n := get(s.Facing.Opposite()); sameHalf(n) {
		if n.stairs.Facing == left && !continued(left) {
			s.Corner = true
			s.Inner = true
			s.Left = true
			return stairBoxes(s)
		}
		if n.stairs.Facing == right && !continued(right) {
			s.Corner = true
			s.Inner = true
			return stairBoxes(s)
		}
	}
	return m.boxes
}
