package sound

type Attack struct {
	Damage bool

	sound
}

type Drowning struct{ sound }

type Burning struct{ sound }

type Fall struct {
	Distance float64

	sound
}

type Burp struct{ sound }

type Pop struct{ sound }

type Explosion struct{ sound }

type Thunder struct{ sound }

type LevelUp struct{ sound }

type Experience struct{ sound }

type GhastWarning struct{ sound }

type GhastShoot struct{ sound }

type FireworkLaunch struct{ sound }

type FireworkHugeBlast struct{ sound }

type FireworkBlast struct{ sound }

type FireworkTwinkle struct{ sound }
