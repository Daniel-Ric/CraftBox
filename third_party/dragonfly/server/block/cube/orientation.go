package cube

import "math"

type Orientation int

func OrientationFromYaw(yaw float64) Orientation {
	yaw = math.Mod(yaw, 360)
	return Orientation(math.Round(yaw / 360 * 16))
}

func (o Orientation) Yaw() float64 {
	return float64(o) / 16 * 360
}

func (o Orientation) Opposite() Orientation {
	return OrientationFromYaw(o.Yaw() + 180)
}

func (o Orientation) RotateLeft() Orientation {
	return OrientationFromYaw(o.Yaw() - 90)
}

func (o Orientation) RotateRight() Orientation {
	return OrientationFromYaw(o.Yaw() + 90)
}
