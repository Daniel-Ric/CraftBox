package sound

import "github.com/df-mc/dragonfly/server/world"

type ItemBreak struct{ sound }

type ItemThrow struct{ sound }

type ItemUseOn struct {
	Block world.Block

	sound
}

type EquipItem struct {
	Item world.Item

	sound
}

type BucketFill struct {
	Liquid world.Liquid

	sound
}

type BucketEmpty struct {
	Liquid world.Liquid

	sound
}

type BowShoot struct{ sound }

type CrossbowLoad struct {
	Stage int

	QuickCharge bool

	sound
}

type CrossbowShoot struct{ sound }

const (
	CrossbowLoadingStart = iota

	CrossbowLoadingMiddle

	CrossbowLoadingEnd
)

type ArrowHit struct{ sound }

type Teleport struct{ sound }

type UseSpyglass struct{ sound }

type StopUsingSpyglass struct{ sound }

type GoatHorn struct {
	Horn Horn

	sound
}

type FireCharge struct{ sound }

type Totem struct{ sound }
