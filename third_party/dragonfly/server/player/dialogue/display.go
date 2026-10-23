package dialogue

import (
	"encoding/json"
	"github.com/go-gl/mathgl/mgl64"
)

type DisplaySettings struct {
	EntityScale mgl64.Vec3

	EntityOffset mgl64.Vec3

	EntityRotation mgl64.Vec3
}

func (d DisplaySettings) MarshalJSON() ([]byte, error) {

	d.EntityRotation[0], d.EntityRotation[1] = d.EntityRotation[1], d.EntityRotation[0]-32
	m := map[string]any{

		"translate": d.EntityOffset.Mul(-32),

		"rotate": d.EntityRotation,
		"scale":  [3]float64{1, 1, 1},
	}
	if (d.EntityScale != mgl64.Vec3{}) {
		m["scale"] = d.EntityScale
	}
	return json.Marshal(m)
}
