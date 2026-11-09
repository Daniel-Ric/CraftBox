package modelpicker

import (
	"errors"
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

type BlockTarget struct {
	Identifier string         `json:"identifier"`
	States     map[string]any `json:"states"`
	RuntimeID  uint32         `json:"runtimeId"`
	Position   [3]int         `json:"position"`
	Distance   float64        `json:"distance"`
}

func PickBlock(eye mgl32.Vec3, pitch, yaw float32, read func([3]int) (BlockTarget, bool)) (BlockTarget, error) {
	p, y := float64(pitch)*math.Pi/180, float64(yaw)*math.Pi/180
	direction := [3]float64{-math.Sin(y) * math.Cos(p), -math.Sin(p), math.Cos(y) * math.Cos(p)}
	var cell, step [3]int
	var next, delta [3]float64
	for axis := 0; axis < 3; axis++ {
		cell[axis] = int(math.Floor(float64(eye[axis])))
		d := direction[axis]
		if math.Abs(d) < 1e-9 {
			next[axis] = math.Inf(1)
			delta[axis] = math.Inf(1)
			continue
		}
		step[axis] = 1
		boundary := cell[axis] + 1
		if d < 0 {
			step[axis] = -1
			boundary = cell[axis]
		}
		next[axis] = (float64(boundary) - float64(eye[axis])) / d
		delta[axis] = math.Abs(1 / d)
	}
	distance := 0.0
	for attempts := 0; attempts < 512 && distance <= 96; attempts++ {
		hit, known := read(cell)
		if !known {
			return BlockTarget{}, errors.New("Terrain under the crosshair is not captured yet")
		}
		switch hit.Identifier {
		case "", "minecraft:air", "minecraft:cave_air", "minecraft:void_air", "minecraft:structure_void", "minecraft:light_block":
		default:
			hit.Position = cell
			hit.Distance = distance
			return hit, nil
		}
		axis := 0
		if next[1] < next[axis] {
			axis = 1
		}
		if next[2] < next[axis] {
			axis = 2
		}
		distance = next[axis]
		cell[axis] += step[axis]
		next[axis] += delta[axis]
	}
	return BlockTarget{}, errors.New("No captured block under the crosshair")
}
