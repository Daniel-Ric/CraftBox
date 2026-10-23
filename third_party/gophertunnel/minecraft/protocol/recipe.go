package protocol

import (
	"github.com/google/uuid"
)

type PotionContainerChangeRecipe struct {
	InputItemID int32

	ReagentItemID int32

	OutputItemID int32
}

func (x *PotionContainerChangeRecipe) Marshal(r IO) {
	r.Varint32(&x.InputItemID)
	r.Varint32(&x.ReagentItemID)
	r.Varint32(&x.OutputItemID)
}

type PotionRecipe struct {
	InputPotionID int32

	InputPotionMetadata int32

	ReagentItemID int32

	ReagentItemMetadata int32

	OutputPotionID int32

	OutputPotionMetadata int32
}

func (x *PotionRecipe) Marshal(r IO) {
	r.Varint32(&x.InputPotionID)
	r.Varint32(&x.InputPotionMetadata)
	r.Varint32(&x.ReagentItemID)
	r.Varint32(&x.ReagentItemMetadata)
	r.Varint32(&x.OutputPotionID)
	r.Varint32(&x.OutputPotionMetadata)
}

const (
	RecipeUnlockContextNone = iota
	RecipeUnlockContextAlwaysUnlocked
	RecipeUnlockContextPlayerInWater
	RecipeUnlockContextPlayerHasManyItems
)

type RecipeUnlockRequirement struct {
	Context int32

	Ingredients []ItemDescriptorCount
}

func (x *RecipeUnlockRequirement) Marshal(r IO) {
	r.Varint32(&x.Context)
	present := x.Context == RecipeUnlockContextNone
	r.Bool(&present)
	if present {
		FuncSlice(r, &x.Ingredients, r.ItemDescriptorCount)
	}
}

type ShapelessRecipe struct {
	RecipeID string

	Input []ItemDescriptorCount

	Output []ItemStack

	UUID uuid.UUID

	Block string

	Priority          int32
	UnlockRequirement Optional[RecipeUnlockRequirement]

	RecipeNetworkID uint32
}

type UserDataShapelessRecipe struct {
	ShapelessRecipe
}

type ShapelessChemistryRecipe struct {
	ShapelessRecipe
}

type ShapedRecipe struct {
	RecipeID string

	Width int32

	Height int32

	Input []ItemDescriptorCount

	Output []ItemStack

	UUID uuid.UUID

	Block string

	Priority int32

	AssumeSymmetry    bool
	UnlockRequirement Optional[RecipeUnlockRequirement]

	RecipeNetworkID uint32
}

type ShapedChemistryRecipe struct {
	ShapedRecipe
}

type MultiRecipe struct {
	UUID uuid.UUID

	RecipeNetworkID uint32
}

type SmithingTransformRecipe struct {
	RecipeNetworkID uint32

	RecipeID string

	Template ItemDescriptorCount

	Base ItemDescriptorCount

	Addition ItemDescriptorCount

	Result ItemStack

	Block string
}

type SmithingTrimRecipe struct {
	RecipeNetworkID uint32

	RecipeID string

	Template ItemDescriptorCount

	Base ItemDescriptorCount

	Addition ItemDescriptorCount

	Block string
}

func (recipe *ShapelessRecipe) Marshal(r IO) {
	marshalShapeless(r, recipe)
}

func (recipe *UserDataShapelessRecipe) Marshal(r IO) {
	marshalShapeless(r, &recipe.ShapelessRecipe)
}

func (recipe *ShapelessChemistryRecipe) Marshal(r IO) {
	marshalShapeless(r, &recipe.ShapelessRecipe)
}

func (recipe *ShapedRecipe) Marshal(r IO) {
	marshalShaped(r, recipe)
}

func (recipe *ShapedChemistryRecipe) Marshal(r IO) {
	marshalShaped(r, &recipe.ShapedRecipe)
}

func (recipe *MultiRecipe) Marshal(r IO) {
	r.UUID(&recipe.UUID)
	r.Varuint32(&recipe.RecipeNetworkID)
}

func (recipe *SmithingTransformRecipe) Marshal(r IO) {
	r.String(&recipe.RecipeID)
	r.ItemDescriptorCount(&recipe.Template)
	r.ItemDescriptorCount(&recipe.Base)
	r.ItemDescriptorCount(&recipe.Addition)
	r.Item(&recipe.Result)
	r.String(&recipe.Block)
	r.Varuint32(&recipe.RecipeNetworkID)
}

func (recipe *SmithingTrimRecipe) Marshal(r IO) {
	r.String(&recipe.RecipeID)
	r.ItemDescriptorCount(&recipe.Template)
	r.ItemDescriptorCount(&recipe.Base)
	r.ItemDescriptorCount(&recipe.Addition)
	r.String(&recipe.Block)
	r.Varuint32(&recipe.RecipeNetworkID)
}

func marshalShaped(r IO, recipe *ShapedRecipe) {
	r.String(&recipe.RecipeID)
	r.Varint32(&recipe.Width)
	r.Varint32(&recipe.Height)
	FuncSlice(r, &recipe.Input, r.ItemDescriptorCount)
	if int64(len(recipe.Input)) != int64(recipe.Width)*int64(recipe.Height) {
		r.InvalidValue(len(recipe.Input), "shaped recipe ingredients", "must equal width multiplied by height")
	}
	FuncSlice(r, &recipe.Output, r.Item)
	r.UUID(&recipe.UUID)
	r.String(&recipe.Block)
	r.Varint32(&recipe.Priority)
	r.Bool(&recipe.AssumeSymmetry)
	OptionalMarshaler(r, &recipe.UnlockRequirement)
	r.Varuint32(&recipe.RecipeNetworkID)
}

func marshalShapeless(r IO, recipe *ShapelessRecipe) {
	r.String(&recipe.RecipeID)
	FuncSlice(r, &recipe.Input, r.ItemDescriptorCount)
	FuncSlice(r, &recipe.Output, r.Item)
	r.UUID(&recipe.UUID)
	r.String(&recipe.Block)
	r.Varint32(&recipe.Priority)
	OptionalMarshaler(r, &recipe.UnlockRequirement)
	r.Varuint32(&recipe.RecipeNetworkID)
}
