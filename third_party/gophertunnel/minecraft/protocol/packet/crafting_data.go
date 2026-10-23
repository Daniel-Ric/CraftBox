package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CraftingData struct {
	ShapedRecipes             []protocol.ShapedRecipe
	ShapelessRecipes          []protocol.ShapelessRecipe
	MultiRecipes              []protocol.MultiRecipe
	UserDataShapelessRecipes  []protocol.UserDataShapelessRecipe
	ShapelessChemistryRecipes []protocol.ShapelessChemistryRecipe
	ShapedChemistryRecipes    []protocol.ShapedChemistryRecipe
	SmithingTransformRecipes  []protocol.SmithingTransformRecipe
	SmithingTrimRecipes       []protocol.SmithingTrimRecipe

	PotionRecipes []protocol.PotionRecipe

	PotionContainerChangeRecipes []protocol.PotionContainerChangeRecipe

	MaterialReducers []protocol.MaterialReducer

	ClearRecipes bool
}

func (*CraftingData) ID() uint32 {
	return IDCraftingData
}

func (pk *CraftingData) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.ShapedRecipes)
	protocol.Slice(io, &pk.ShapelessRecipes)
	protocol.Slice(io, &pk.MultiRecipes)
	protocol.Slice(io, &pk.UserDataShapelessRecipes)
	protocol.Slice(io, &pk.ShapelessChemistryRecipes)
	protocol.Slice(io, &pk.ShapedChemistryRecipes)
	protocol.Slice(io, &pk.SmithingTransformRecipes)
	protocol.Slice(io, &pk.SmithingTrimRecipes)
	protocol.Slice(io, &pk.PotionRecipes)
	protocol.Slice(io, &pk.PotionContainerChangeRecipes)
	protocol.FuncSlice(io, &pk.MaterialReducers, io.MaterialReducer)
	io.Bool(&pk.ClearRecipes)
}
