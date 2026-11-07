package item

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestVerticalThrowableDirectionStaysFinite(t *testing.T) {
	for _, pitch := range []float64{-90, -89.9, 0, 89.9, 90} {
		direction := throwableOffset(cube.Rotation{0, pitch}).Vec3()
		for _, v := range direction {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("pitch %v produced invalid direction %v", pitch, direction)
			}
		}
	}
}
