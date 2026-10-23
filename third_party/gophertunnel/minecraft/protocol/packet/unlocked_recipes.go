package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	UnlockedRecipesTypeEmpty = iota
	UnlockedRecipesTypeInitiallyUnlocked
	UnlockedRecipesTypeNewlyUnlocked
	UnlockedRecipesTypeRemoveUnlocked
	UnlockedRecipesTypeRemoveAllUnlocked
)

type UnlockedRecipes struct {
	UnlockType uint32

	Recipes []string
}

func (*UnlockedRecipes) ID() uint32 {
	return IDUnlockedRecipes
}

func (pk *UnlockedRecipes) Marshal(io protocol.IO) {
	io.Uint32(&pk.UnlockType)
	protocol.FuncSlice(io, &pk.Recipes, io.String)
}
