package particle

import "image/color"

type HugeExplosion struct{ particle }

type EndermanTeleport struct{ particle }

type SnowballPoof struct{ particle }

type EggSmash struct{ particle }

type Splash struct {
	particle

	Colour color.RGBA
}

type Effect struct {
	particle

	Colour color.RGBA
}

type EntityFlame struct{ particle }
