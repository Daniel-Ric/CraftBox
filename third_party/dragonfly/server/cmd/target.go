package cmd

import (
	"github.com/df-mc/dragonfly/server/internal/sliceutil"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"slices"
)

type Target interface {
	Position() mgl64.Vec3
}

type NamedTarget interface {
	Target

	Name() string
}

func targets(tx *world.Tx) (entities []Target, players []NamedTarget) {
	if tx == nil {
		return nil, nil
	}
	ent := sliceutil.Convert[Target](slices.Collect(tx.Entities()))
	pl := sliceutil.Convert[NamedTarget](slices.Collect(tx.Players()))
	return ent, pl
}
