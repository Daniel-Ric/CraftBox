package protocol

const (
	FilterCauseServerChatPublic = iota
	FilterCauseServerChatWhisper
	FilterCauseSignText
	FilterCauseAnvilText
	FilterCauseBookAndQuillText
	FilterCauseCommandBlockText
	FilterCauseBlockActorDataText
	FilterCauseJoinEventText
	FilterCauseLeaveEventText
	FilterCauseSlashCommandChat
	FilterCauseCartographyText
	FilterCauseKickCommand
	FilterCauseTitleCommand
	FilterCauseSummonCommand
)

type ItemStackRequest struct {
	RequestID int32

	Actions []StackRequestAction

	FilterStrings []string

	FilterCause int32
}

func (x *ItemStackRequest) Marshal(r IO) {
	r.Varint32(&x.RequestID)
	FuncSlice(r, &x.Actions, r.StackRequestAction)
	FuncSlice(r, &x.FilterStrings, r.String)
	r.Int32(&x.FilterCause)
}

func stackRequestActionVariant(id uint8) uint32 {
	if id > StackRequestActionTakeOutContainer {
		return uint32(id) - 2
	}
	return uint32(id)
}

func lookupStackRequestActionType(x StackRequestAction, id *uint8) bool {
	switch x.(type) {
	case *TakeStackRequestAction:
		*id = StackRequestActionTake
	case *PlaceStackRequestAction:
		*id = StackRequestActionPlace
	case *SwapStackRequestAction:
		*id = StackRequestActionSwap
	case *DropStackRequestAction:
		*id = StackRequestActionDrop
	case *DestroyStackRequestAction:
		*id = StackRequestActionDestroy
	case *ConsumeStackRequestAction:
		*id = StackRequestActionConsume
	case *CreateStackRequestAction:
		*id = StackRequestActionCreate
	case *LabTableCombineStackRequestAction:
		*id = StackRequestActionLabTableCombine
	case *BeaconPaymentStackRequestAction:
		*id = StackRequestActionBeaconPayment
	case *MineBlockStackRequestAction:
		*id = StackRequestActionMineBlock
	case *CraftRecipeStackRequestAction:
		*id = StackRequestActionCraftRecipe
	case *AutoCraftRecipeStackRequestAction:
		*id = StackRequestActionCraftRecipeAuto
	case *CraftCreativeStackRequestAction:
		*id = StackRequestActionCraftCreative
	case *CraftRecipeOptionalStackRequestAction:
		*id = StackRequestActionCraftRecipeOptional
	case *CraftGrindstoneRecipeStackRequestAction:
		*id = StackRequestActionCraftGrindstone
	case *CraftLoomRecipeStackRequestAction:
		*id = StackRequestActionCraftLoom
	case *CraftReservedStackRequestAction:
		*id = StackRequestActionCraftReserved
	case *CraftNonImplementedStackRequestAction:
		*id = StackRequestActionCraftNonImplementedDeprecated
	case *CraftResultsDeprecatedStackRequestAction:
		*id = StackRequestActionCraftResultsDeprecated
	default:
		return false
	}
	return true
}

func lookupStackRequestAction(id uint8, x *StackRequestAction) bool {
	switch id {
	case StackRequestActionTake:
		*x = &TakeStackRequestAction{}
	case StackRequestActionPlace:
		*x = &PlaceStackRequestAction{}
	case StackRequestActionSwap:
		*x = &SwapStackRequestAction{}
	case StackRequestActionDrop:
		*x = &DropStackRequestAction{}
	case StackRequestActionDestroy:
		*x = &DestroyStackRequestAction{}
	case StackRequestActionConsume:
		*x = &ConsumeStackRequestAction{}
	case StackRequestActionCreate:
		*x = &CreateStackRequestAction{}
	case StackRequestActionLabTableCombine:
		*x = &LabTableCombineStackRequestAction{}
	case StackRequestActionBeaconPayment:
		*x = &BeaconPaymentStackRequestAction{}
	case StackRequestActionMineBlock:
		*x = &MineBlockStackRequestAction{}
	case StackRequestActionCraftRecipe:
		*x = &CraftRecipeStackRequestAction{}
	case StackRequestActionCraftRecipeAuto:
		*x = &AutoCraftRecipeStackRequestAction{}
	case StackRequestActionCraftCreative:
		*x = &CraftCreativeStackRequestAction{}
	case StackRequestActionCraftRecipeOptional:
		*x = &CraftRecipeOptionalStackRequestAction{}
	case StackRequestActionCraftGrindstone:
		*x = &CraftGrindstoneRecipeStackRequestAction{}
	case StackRequestActionCraftLoom:
		*x = &CraftLoomRecipeStackRequestAction{}
	case StackRequestActionCraftReserved:
		*x = &CraftReservedStackRequestAction{}
	case StackRequestActionCraftNonImplementedDeprecated:
		*x = &CraftNonImplementedStackRequestAction{}
	case StackRequestActionCraftResultsDeprecated:
		*x = &CraftResultsDeprecatedStackRequestAction{}
	default:
		return false
	}
	return true
}

const (
	ItemStackResponseStatusOK = iota
	ItemStackResponseStatusError
	ItemStackResponseStatusInvalidRequestActionType
	ItemStackResponseStatusActionRequestNotAllowed
	ItemStackResponseStatusScreenHandlerEndRequestFailed
	ItemStackResponseStatusItemRequestActionHandlerCommitFailed
	ItemStackResponseStatusInvalidRequestCraftActionType
	ItemStackResponseStatusInvalidCraftRequest
	ItemStackResponseStatusInvalidCraftRequestScreen
	ItemStackResponseStatusInvalidCraftResult
	ItemStackResponseStatusInvalidCraftResultIndex
	ItemStackResponseStatusInvalidCraftResultItem
	ItemStackResponseStatusInvalidItemNetId
	ItemStackResponseStatusMissingCreatedOutputContainer
	ItemStackResponseStatusFailedToSetCreatedItemOutputSlot
	ItemStackResponseStatusRequestAlreadyInProgress
	ItemStackResponseStatusFailedToInitSparseContainer
	ItemStackResponseStatusResultTransferFailed
	ItemStackResponseStatusExpectedItemSlotNotFullyConsumed
	ItemStackResponseStatusExpectedAnywhereItemNotFullyConsumed
	ItemStackResponseStatusItemAlreadyConsumedFromSlot
	ItemStackResponseStatusConsumedTooMuchFromSlot
	ItemStackResponseStatusMismatchSlotExpectedConsumedItem
	ItemStackResponseStatusMismatchSlotExpectedConsumedItemNetIdVariant
	ItemStackResponseStatusFailedToMatchExpectedSlotConsumedItem
	ItemStackResponseStatusFailedToMatchExpectedAllowedAnywhereConsumedItem
	ItemStackResponseStatusConsumedItemOutOfAllowedSlotRange
	ItemStackResponseStatusConsumedItemNotAllowed
	ItemStackResponseStatusPlayerNotInCreativeMode
	ItemStackResponseStatusInvalidExperimentalRecipeRequest
	ItemStackResponseStatusFailedToCraftCreative
	ItemStackResponseStatusFailedToGetLevelRecipe
	ItemStackResponseStatusFailedToFindRecipeByNetId
	ItemStackResponseStatusMismatchedCraftingSize
	ItemStackResponseStatusMissingInputSparseContainer
	ItemStackResponseStatusMismatchedRecipeForInputGridItems
	ItemStackResponseStatusEmptyCraftResults
	ItemStackResponseStatusFailedToEnchant
	ItemStackResponseStatusMissingInputItem
	ItemStackResponseStatusInsufficientPlayerLevelToEnchant
	ItemStackResponseStatusMissingMaterialItem
	ItemStackResponseStatusMissingActor
	ItemStackResponseStatusUnknownPrimaryEffect
	ItemStackResponseStatusPrimaryEffectOutOfRange
	ItemStackResponseStatusPrimaryEffectUnavailable
	ItemStackResponseStatusSecondaryEffectOutOfRange
	ItemStackResponseStatusSecondaryEffectUnavailable
	ItemStackResponseStatusDstContainerEqualToCreatedOutputContainer
	ItemStackResponseStatusDstContainerAndSlotEqualToSrcContainerAndSlot
	ItemStackResponseStatusFailedToValidateSrcSlot
	ItemStackResponseStatusFailedToValidateDstSlot
	ItemStackResponseStatusInvalidAdjustedAmount
	ItemStackResponseStatusInvalidItemSetType
	ItemStackResponseStatusInvalidTransferAmount
	ItemStackResponseStatusCannotSwapItem
	ItemStackResponseStatusCannotPlaceItem
	ItemStackResponseStatusUnhandledItemSetType
	ItemStackResponseStatusInvalidRemovedAmount
	ItemStackResponseStatusInvalidRegion
	ItemStackResponseStatusCannotDropItem
	ItemStackResponseStatusCannotDestroyItem
	ItemStackResponseStatusInvalidSourceContainer
	ItemStackResponseStatusItemNotConsumed
	ItemStackResponseStatusInvalidNumCrafts
	ItemStackResponseStatusInvalidCraftResultStackSize
	ItemStackResponseStatusCannotRemoveItem
	ItemStackResponseStatusCannotConsumeItem
	ItemStackResponseStatusScreenStackError
)

type ItemStackResponse struct {
	Status uint8

	RequestID int32

	ContainerInfo []StackResponseContainerInfo
}

func (x *ItemStackResponse) Marshal(r IO) {
	r.Uint8(&x.Status)
	r.Varint32(&x.RequestID)
	var containerInfo Optional[[]StackResponseContainerInfo]
	if len(x.ContainerInfo) != 0 {
		containerInfo = Option(x.ContainerInfo)
	}
	OptionalFunc(r, &containerInfo, func(containerInfo *[]StackResponseContainerInfo) {
		Slice(r, containerInfo)
	})
	if value, ok := containerInfo.Value(); ok {
		x.ContainerInfo = value
	}
}

type StackResponseContainerInfo struct {
	Container FullContainerName

	SlotInfo []StackResponseSlotInfo
}

func (x *StackResponseContainerInfo) Marshal(r IO) {
	Single(r, &x.Container)
	Slice(r, &x.SlotInfo)
}

type StackResponseSlotInfo struct {
	Slot, HotbarSlot byte

	Count byte

	StackNetworkID int32

	CustomName string

	FilteredCustomName Optional[string]

	DurabilityCorrection int32
}

func (x *StackResponseSlotInfo) Marshal(r IO) {
	r.Uint8(&x.Slot)
	r.Uint8(&x.HotbarSlot)
	r.Uint8(&x.Count)
	var stackNetworkID Optional[int32]
	if x.StackNetworkID > 0 {
		stackNetworkID = Option(x.StackNetworkID)
	}
	OptionalFunc(r, &stackNetworkID, r.Varint32)
	if value, ok := stackNetworkID.Value(); ok {
		x.StackNetworkID = value
	}
	r.String(&x.CustomName)
	OptionalFunc(r, &x.FilteredCustomName, r.String)
	r.Varint32(&x.DurabilityCorrection)
	if x.DurabilityCorrection < -32768 || x.DurabilityCorrection > 32767 {
		r.InvalidValue(x.DurabilityCorrection, "durability correction", "must fit in an int16")
	}
}

type StackRequestAction interface {
	Marshaler
}

const (
	StackRequestActionTake = iota
	StackRequestActionPlace
	StackRequestActionSwap
	StackRequestActionDrop
	StackRequestActionDestroy
	StackRequestActionConsume
	StackRequestActionCreate
	StackRequestActionPlaceInContainer
	StackRequestActionTakeOutContainer
	StackRequestActionLabTableCombine
	StackRequestActionBeaconPayment
	StackRequestActionMineBlock
	StackRequestActionCraftRecipe
	StackRequestActionCraftRecipeAuto
	StackRequestActionCraftCreative
	StackRequestActionCraftRecipeOptional
	StackRequestActionCraftGrindstone
	StackRequestActionCraftLoom
	StackRequestActionCraftReserved
	StackRequestActionCraftNonImplementedDeprecated
	StackRequestActionCraftResultsDeprecated
)

type transferStackRequestAction struct {
	Count byte

	Source, Destination StackRequestSlotInfo
}

func (a *transferStackRequestAction) Marshal(r IO) {
	r.Uint8(&a.Count)
	StackReqSlotInfo(r, &a.Source)
	StackReqSlotInfo(r, &a.Destination)
}

type TakeStackRequestAction struct {
	transferStackRequestAction
}

type PlaceStackRequestAction struct {
	transferStackRequestAction
}

type SwapStackRequestAction struct {
	Source, Destination StackRequestSlotInfo
}

func (a *SwapStackRequestAction) Marshal(r IO) {
	StackReqSlotInfo(r, &a.Source)
	StackReqSlotInfo(r, &a.Destination)
}

type DropStackRequestAction struct {
	Count byte

	Source StackRequestSlotInfo

	Randomly bool
}

func (a *DropStackRequestAction) Marshal(r IO) {
	r.Uint8(&a.Count)
	StackReqSlotInfo(r, &a.Source)
	r.Bool(&a.Randomly)
}

type DestroyStackRequestAction struct {
	Count byte

	Source StackRequestSlotInfo
}

func (a *DestroyStackRequestAction) Marshal(r IO) {
	r.Uint8(&a.Count)
	StackReqSlotInfo(r, &a.Source)
}

type ConsumeStackRequestAction struct {
	DestroyStackRequestAction
}

type CreateStackRequestAction struct {
	ResultsSlot byte
}

func (a *CreateStackRequestAction) Marshal(r IO) {
	r.Uint8(&a.ResultsSlot)
}

type LabTableCombineStackRequestAction struct{}

func (a *LabTableCombineStackRequestAction) Marshal(IO) {}

type BeaconPaymentStackRequestAction struct {
	PrimaryEffect, SecondaryEffect int32
}

func (a *BeaconPaymentStackRequestAction) Marshal(r IO) {
	r.Varint32(&a.PrimaryEffect)
	r.Varint32(&a.SecondaryEffect)
}

type MineBlockStackRequestAction struct {
	HotbarSlot int32

	PredictedDurability int32

	StackNetworkID int32
}

func (a *MineBlockStackRequestAction) Marshal(r IO) {
	r.Varint32(&a.HotbarSlot)
	r.Varint32(&a.PredictedDurability)
	r.Int32(&a.StackNetworkID)
}

type CraftRecipeStackRequestAction struct {
	RecipeNetworkID uint32

	NumberOfCrafts byte
}

func (a *CraftRecipeStackRequestAction) Marshal(r IO) {
	r.Varuint32(&a.RecipeNetworkID)
	r.Uint8(&a.NumberOfCrafts)
}

type AutoCraftRecipeStackRequestAction struct {
	RecipeNetworkID uint32

	NumberOfCrafts byte

	Ingredients []ItemDescriptorCount
}

func (a *AutoCraftRecipeStackRequestAction) Marshal(r IO) {
	r.Varuint32(&a.RecipeNetworkID)
	r.Uint8(&a.NumberOfCrafts)
	FuncIOSlice(r, &a.Ingredients, StackRequestItemDescriptorCount)
}

type CraftCreativeStackRequestAction struct {
	CreativeItemNetworkID uint32

	NumberOfCrafts byte
}

func (a *CraftCreativeStackRequestAction) Marshal(r IO) {
	r.Varuint32(&a.CreativeItemNetworkID)
	r.Uint8(&a.NumberOfCrafts)
}

type CraftRecipeOptionalStackRequestAction struct {
	RecipeNetworkID uint32

	FilterStringIndex int32
}

func (c *CraftRecipeOptionalStackRequestAction) Marshal(r IO) {
	r.Varuint32(&c.RecipeNetworkID)
	r.Int32(&c.FilterStringIndex)
}

type CraftGrindstoneRecipeStackRequestAction struct {
	RecipeNetworkID uint32

	NumberOfCrafts byte

	Cost int32
}

func (c *CraftGrindstoneRecipeStackRequestAction) Marshal(r IO) {
	IntegerFunc(&c.RecipeNetworkID, r.Int32)
	r.Uint8(&c.NumberOfCrafts)
	r.Varint32(&c.Cost)
}

type CraftLoomRecipeStackRequestAction struct {
	Pattern string

	TimesCrafted byte
}

func (c *CraftLoomRecipeStackRequestAction) Marshal(r IO) {
	r.String(&c.Pattern)
	r.Uint8(&c.TimesCrafted)
}

type CraftReservedStackRequestAction struct {
	ReservedID string

	NumberOfCrafts byte
}

func (c *CraftReservedStackRequestAction) Marshal(r IO) {
	r.String(&c.ReservedID)
	r.Uint8(&c.NumberOfCrafts)
}

type CraftNonImplementedStackRequestAction struct{}

func (*CraftNonImplementedStackRequestAction) Marshal(IO) {}

type CraftResultsDeprecatedStackRequestAction struct {
	ResultItems  []StackRequestItem
	TimesCrafted byte
}

func (a *CraftResultsDeprecatedStackRequestAction) Marshal(r IO) {
	FuncSlice(r, &a.ResultItems, r.StackRequestItem)
	r.Uint8(&a.TimesCrafted)
}

type StackRequestSlotInfo struct {
	Container FullContainerName

	Slot byte

	StackNetworkID int32
}

func StackReqSlotInfo(r IO, x *StackRequestSlotInfo) {
	Single(r, &x.Container)
	r.Uint8(&x.Slot)
	r.Int32(&x.StackNetworkID)
}
