package protocol

import (
	"github.com/google/uuid"
)

const (
	CommandPermissionLevelAny = iota
	CommandPermissionLevelGameDirectors
	CommandPermissionLevelAdmin
	CommandPermissionLevelHost
	CommandPermissionLevelOwner
	CommandPermissionLevelInternal
)

type Command struct {
	Name string

	Description string

	Flags uint16

	PermissionLevel byte

	AliasesOffset uint32

	ChainedSubcommandOffsets []uint32

	Overloads []CommandOverload
}

func (c *Command) Marshal(r IO) {
	permissionStr := commandPermissionToString(c.PermissionLevel)
	r.String(&c.Name)
	r.String(&c.Description)
	r.Uint16(&c.Flags)
	r.String(&permissionStr)
	commandPermissionFromString(r, &c.PermissionLevel, permissionStr)
	r.Uint32(&c.AliasesOffset)
	FuncSlice(r, &c.ChainedSubcommandOffsets, r.Uint32)
	Slice(r, &c.Overloads)
}

type CommandOverload struct {
	Chaining bool

	Parameters []CommandParameter
}

func (c *CommandOverload) Marshal(r IO) {
	r.Bool(&c.Chaining)
	Slice(r, &c.Parameters)
}

const (
	CommandArgValid    = 0x100000
	CommandArgEnum     = 0x200000
	CommandArgSuffixed = 0x1000000
	CommandArgSoftEnum = 0x4000000
)

const (
	CommandArgTypeInt = iota + 1
	_
	CommandArgTypeFloat
	CommandArgTypeRValue
	CommandArgTypeWildcardInt
	CommandArgTypeOperator
	CommandArgTypeCompareOperator
	CommandArgTypeTarget
	CommandArgTypeStandaloneTarget
	CommandArgTypeWildcardTarget
	CommandArgTypeNonIDTarget
	CommandArgTypeScoreArg
	CommandArgTypeScoreArgs
	CommandArgTypeScoreSelectParam
	CommandArgTypeScoreSelector
	CommandArgTypeTagSelector
	CommandArgTypeFilepath
	CommandArgTypeFilePathVal
	CommandArgTypeFilePathCont
	CommandArgTypeIntRangeVal
	CommandArgTypeIntRangePostVal
	CommandArgTypeIntRange
	CommandArgTypeIntegerRange
	CommandArgTypeRationalRangeVal
	CommandArgTypeRationalRangePostVal
	CommandArgTypeRationalRange
	CommandArgTypeRationalRangeFull
	CommandArgTypeSelectionArgs
	CommandArgTypeArgs
	CommandArgTypeArg
	CommandArgTypeMArg
	CommandArgTypeMValue
	CommandArgTypeName
	CommandArgTypeType
	CommandArgTypeFamily
	CommandArgTypePermission
	CommandArgTypePermissions
	CommandArgTypePermissionSelector
	CommandArgTypePermissionElement
	CommandArgTypePermissionElements
	CommandArgTypeTag
	CommandArgTypeHasItemElement
	CommandArgTypeHasItemElements
	CommandArgTypeHasItem
	CommandArgTypeHasItems
	CommandArgTypeHasItemSelector
	CommandArgTypeEquipmentSlots
	CommandArgTypePropertyValue
	CommandArgTypeHasPropertyParamValue
	CommandArgTypeHasPropertyParamEnumValue
	CommandArgTypeHasPropertyArg
	CommandArgTypeHasPropertyArgs
	CommandArgTypeHasPropertyElement
	CommandArgTypeHasPropertyElements
	CommandArgTypeHasPropertySelector
	CommandArgTypeString
	CommandArgTypeIDContinuation
	CommandArgTypeCoordXInt
	CommandArgTypeCoordYInt
	CommandArgTypeCoordZInt
	CommandArgTypeCoordXFloat
	CommandArgTypeCoordYFloat
	CommandArgTypeCoordZFloat
	CommandArgTypeBlockPosition
	CommandArgTypePosition
	CommandArgTypeMessageXP
	CommandArgTypeMessage
	CommandArgTypeMessageRoot
	CommandArgTypePostSelector
	CommandArgTypeRawText
	CommandArgTypeTextContinuation
	CommandArgTypeJSONValue
	CommandArgTypeJSONField
	CommandArgTypeJSON
	CommandArgTypeJSONObjectFields
	CommandArgTypeJSONObjectCont
	CommandArgTypeJSONArray
	CommandArgTypeJSONArrayValues
	CommandArgTypeJSONArrayCont
	CommandArgTypeBlockState
	CommandArgTypeBlockStateKey
	CommandArgTypeBlockStateValue
	CommandArgTypeBlockStateValues
	CommandArgTypeBlockStates
	CommandArgTypeBlockStatesCont
	CommandArgTypeClockTimeMarker
	CommandArgTypeCommand
	CommandArgTypeSlashCommand
	CommandArgTypeCodeBuilderArg
	CommandArgTypeCodeBuilderArgs
	CommandArgTypeCodeBuilderSelectParam
	CommandArgTypeCodeBuilderSelector

	CommandArgTypeChainedCommand = 0x8000000
)

const (
	ParamOptionCollapseEnum = iota + 1
	ParamOptionHasSemanticConstraint
	_
	ParamOptionAsChainedCommand
)

type CommandParameter struct {
	Name string

	Type uint32

	Optional bool

	Options byte
}

func (c *CommandParameter) Marshal(r IO) {
	r.String(&c.Name)
	r.Uint32(&c.Type)
	r.Bool(&c.Optional)
	r.Uint8(&c.Options)
}

type CommandEnum struct {
	Type string

	ValueIndices []uint32
}

type CommandEnumContext struct {
	EnumValues []string
}

func (ctx CommandEnumContext) Marshal(r IO, x *CommandEnum) {
	r.String(&x.Type)
	FuncSlice(r, &x.ValueIndices, r.Uint32)
}

type ChainedSubcommand struct {
	Name string

	Values []ChainedSubcommandValue
}

func (x *ChainedSubcommand) Marshal(r IO) {
	r.String(&x.Name)
	Slice(r, &x.Values)
}

type ChainedSubcommandValue struct {
	Index uint32

	Value uint32
}

func (x *ChainedSubcommandValue) Marshal(r IO) {
	r.Varuint32(&x.Index)
	r.Varuint32(&x.Value)
}

type DynamicEnum struct {
	Type string

	Values []string
}

func (x *DynamicEnum) Marshal(r IO) {
	r.String(&x.Type)
	FuncSlice(r, &x.Values, r.String)
}

const (
	CommandEnumConstraintCheatsEnabled = iota
	CommandEnumConstraintOperatorPermissions
	CommandEnumConstraintHostPermissions
	_
)

type CommandEnumConstraint struct {
	EnumValueIndex uint32

	EnumIndex uint32

	Constraints []byte
}

func (c *CommandEnumConstraint) Marshal(r IO) {
	r.Uint32(&c.EnumValueIndex)
	r.Uint32(&c.EnumIndex)
	r.ByteSlice(&c.Constraints)
}

const (
	CommandOriginPlayer = iota
	CommandOriginBlock
	CommandOriginMinecartBlock
	CommandOriginDevConsole
	CommandOriginTest
	CommandOriginAutomationPlayer
	CommandOriginClientAutomation
	CommandOriginDedicatedServer
	CommandOriginEntity
	CommandOriginVirtual
	CommandOriginGameArgument
	CommandOriginEntityServer
	CommandOriginPrecompiled
	CommandOriginGameDirectorEntityServer
	CommandOriginScript
	CommandOriginExecutor
)

type CommandOrigin struct {
	Origin uint32

	UUID uuid.UUID

	RequestID string

	PlayerUniqueID int64
}

func CommandOriginData(r IO, x *CommandOrigin) {
	originStr := commandOriginToString(x.Origin)
	r.String(&originStr)
	r.UUID(&x.UUID)
	r.String(&x.RequestID)
	r.ActorUniqueIDInt64(&x.PlayerUniqueID)
	commandOriginFromString(r, &x.Origin, originStr)
}

type CommandOutputMessage struct {
	Success bool

	Message string

	Parameters []string
}

func (x *CommandOutputMessage) Marshal(r IO) {
	r.String(&x.Message)
	r.Bool(&x.Success)
	FuncSlice(r, &x.Parameters, r.String)
}

func commandOriginToString(x uint32) string {
	switch x {
	case CommandOriginPlayer:
		return "player"
	case CommandOriginBlock:
		return "commandblock"
	case CommandOriginMinecartBlock:
		return "minecartcommandblock"
	case CommandOriginDevConsole:
		return "devconsole"
	case CommandOriginTest:
		return "test"
	case CommandOriginAutomationPlayer:
		return "automationplayer"
	case CommandOriginClientAutomation:
		return "clientautomation"
	case CommandOriginDedicatedServer:
		return "dedicatedserver"
	case CommandOriginEntity:
		return "entity"
	case CommandOriginVirtual:
		return "virtual"
	case CommandOriginGameArgument:
		return "gameargument"
	case CommandOriginEntityServer:
		return "entityserver"
	case CommandOriginPrecompiled:
		return "precompiled"
	case CommandOriginGameDirectorEntityServer:
		return "gamedirectorentityserver"
	case CommandOriginScript:
		return "scripting"
	case CommandOriginExecutor:
		return "executecontext"
	default:
		return "unknown"
	}
}

func commandOriginFromString(r IO, x *uint32, s string) {
	switch s {
	case "player":
		*x = CommandOriginPlayer
	case "commandblock":
		*x = CommandOriginBlock
	case "minecartcommandblock":
		*x = CommandOriginMinecartBlock
	case "devconsole":
		*x = CommandOriginDevConsole
	case "test":
		*x = CommandOriginTest
	case "automationplayer":
		*x = CommandOriginAutomationPlayer
	case "clientautomation":
		*x = CommandOriginClientAutomation
	case "dedicatedserver":
		*x = CommandOriginDedicatedServer
	case "entity":
		*x = CommandOriginEntity
	case "virtual":
		*x = CommandOriginVirtual
	case "gameargument":
		*x = CommandOriginGameArgument
	case "entityserver":
		*x = CommandOriginEntityServer
	case "precompiled":
		*x = CommandOriginPrecompiled
	case "gamedirectorentityserver":
		*x = CommandOriginGameDirectorEntityServer
	case "scripting":
		*x = CommandOriginScript
	case "executecontext":
		*x = CommandOriginExecutor
	default:
		r.InvalidValue(s, "origin", "unknown origin")
	}
}

func commandPermissionToString(x byte) string {
	switch x {
	case CommandPermissionLevelAny:
		return "any"
	case CommandPermissionLevelGameDirectors:
		return "gamedirectors"
	case CommandPermissionLevelAdmin:
		return "admin"
	case CommandPermissionLevelHost:
		return "host"
	case CommandPermissionLevelOwner:
		return "owner"
	case CommandPermissionLevelInternal:
		return "internal"
	default:
		return "unknown"
	}
}

func commandPermissionFromString(r IO, x *byte, s string) {
	switch s {
	case "any":
		*x = CommandPermissionLevelAny
	case "gamedirectors":
		*x = CommandPermissionLevelGameDirectors
	case "admin":
		*x = CommandPermissionLevelAdmin
	case "host":
		*x = CommandPermissionLevelHost
	case "owner":
		*x = CommandPermissionLevelOwner
	case "internal":
		*x = CommandPermissionLevelInternal
	default:
		r.InvalidValue(s, "permissionLevel", "unknown permission level")
	}
}
