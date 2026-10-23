package entity

import "time"

type Flammable interface {
	OnFireDuration() time.Duration

	SetOnFire(duration time.Duration)

	Extinguish()
}
