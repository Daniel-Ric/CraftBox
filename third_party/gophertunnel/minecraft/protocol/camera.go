package protocol

import (
	"image/color"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	AimAssistTargetModeAngle = iota
	AimAssistTargetModeDistance
)

const (
	AudioListenerCamera = iota
	AudioListenerPlayer
)

const (
	EasingTypeLinear = iota
	EasingTypeSpring
	EasingTypeInQuad
	EasingTypeOutQuad
	EasingTypeInOutQuad
	EasingTypeInCubic
	EasingTypeOutCubic
	EasingTypeInOutCubic
	EasingTypeInQuart
	EasingTypeOutQuart
	EasingTypeInOutQuart
	EasingTypeInQuint
	EasingTypeOutQuint
	EasingTypeInOutQuint
	EasingTypeInSine
	EasingTypeOutSine
	EasingTypeInOutSine
	EasingTypeInExpo
	EasingTypeOutExpo
	EasingTypeInOutExpo
	EasingTypeInCirc
	EasingTypeOutCirc
	EasingTypeInOutCirc
	EasingTypeInBounce
	EasingTypeOutBounce
	EasingTypeInOutBounce
	EasingTypeInBack
	EasingTypeOutBack
	EasingTypeInOutBack
	EasingTypeInElastic
	EasingTypeOutElastic
	EasingTypeInOutElastic
	EasingTypeInverseLerp
)

func easingTypeFromString(io IO, x *int32, s string) {
	switch s {
	case "linear":
		*x = EasingTypeLinear
	case "spring":
		*x = EasingTypeSpring
	case "in_quad":
		*x = EasingTypeInQuad
	case "out_quad":
		*x = EasingTypeOutQuad
	case "in_out_quad":
		*x = EasingTypeInOutQuad
	case "in_cubic":
		*x = EasingTypeInCubic
	case "out_cubic":
		*x = EasingTypeOutCubic
	case "in_out_cubic":
		*x = EasingTypeInOutCubic
	case "in_quart":
		*x = EasingTypeInQuart
	case "out_quart":
		*x = EasingTypeOutQuart
	case "in_out_quart":
		*x = EasingTypeInOutQuart
	case "in_quint":
		*x = EasingTypeInQuint
	case "out_quint":
		*x = EasingTypeOutQuint
	case "in_out_quint":
		*x = EasingTypeInOutQuint
	case "in_sine":
		*x = EasingTypeInSine
	case "out_sine":
		*x = EasingTypeOutSine
	case "in_out_sine":
		*x = EasingTypeInOutSine
	case "in_expo":
		*x = EasingTypeInExpo
	case "out_expo":
		*x = EasingTypeOutExpo
	case "in_out_expo":
		*x = EasingTypeInOutExpo
	case "in_circ":
		*x = EasingTypeInCirc
	case "out_circ":
		*x = EasingTypeOutCirc
	case "in_out_circ":
		*x = EasingTypeInOutCirc
	case "in_back":
		*x = EasingTypeInBack
	case "out_back":
		*x = EasingTypeOutBack
	case "in_out_back":
		*x = EasingTypeInOutBack
	case "in_elastic":
		*x = EasingTypeInElastic
	case "out_elastic":
		*x = EasingTypeOutElastic
	case "in_out_elastic":
		*x = EasingTypeInOutElastic
	case "in_bounce":
		*x = EasingTypeInBounce
	case "out_bounce":
		*x = EasingTypeOutBounce
	case "in_out_bounce":
		*x = EasingTypeInOutBounce
	case "inverse_lerp":
		*x = EasingTypeInverseLerp
	default:
		io.InvalidValue(s, "easingType", "unknown easing type")
	}
}

func easingTypeToString(x int32) string {
	switch x {
	case EasingTypeLinear:
		return "linear"
	case EasingTypeSpring:
		return "spring"
	case EasingTypeInQuad:
		return "in_quad"
	case EasingTypeOutQuad:
		return "out_quad"
	case EasingTypeInOutQuad:
		return "in_out_quad"
	case EasingTypeInCubic:
		return "in_cubic"
	case EasingTypeOutCubic:
		return "out_cubic"
	case EasingTypeInOutCubic:
		return "in_out_cubic"
	case EasingTypeInQuart:
		return "in_quart"
	case EasingTypeOutQuart:
		return "out_quart"
	case EasingTypeInOutQuart:
		return "in_out_quart"
	case EasingTypeInQuint:
		return "in_quint"
	case EasingTypeOutQuint:
		return "out_quint"
	case EasingTypeInOutQuint:
		return "in_out_quint"
	case EasingTypeInSine:
		return "in_sine"
	case EasingTypeOutSine:
		return "out_sine"
	case EasingTypeInOutSine:
		return "in_out_sine"
	case EasingTypeInExpo:
		return "in_expo"
	case EasingTypeOutExpo:
		return "out_expo"
	case EasingTypeInOutExpo:
		return "in_out_expo"
	case EasingTypeInCirc:
		return "in_circ"
	case EasingTypeOutCirc:
		return "out_circ"
	case EasingTypeInOutCirc:
		return "in_out_circ"
	case EasingTypeInBack:
		return "in_back"
	case EasingTypeOutBack:
		return "out_back"
	case EasingTypeInOutBack:
		return "in_out_back"
	case EasingTypeInElastic:
		return "in_elastic"
	case EasingTypeOutElastic:
		return "out_elastic"
	case EasingTypeInOutElastic:
		return "in_out_elastic"
	case EasingTypeInBounce:
		return "in_bounce"
	case EasingTypeOutBounce:
		return "out_bounce"
	case EasingTypeInOutBounce:
		return "in_out_bounce"
	case EasingTypeInverseLerp:
		return "inverse_lerp"
	default:
		return "unknown"
	}
}

const (
	SplineTypeCatmullRom = "catmullrom"
	SplineTypeLinear     = "linear"
)

type CameraEase struct {
	Type uint8

	Duration float32
}

func (x *CameraEase) Marshal(r IO) {
	r.Uint8(&x.Type)
	r.Float32(&x.Duration)
}

type CameraInstructionSet struct {
	Preset uint32

	Ease Optional[CameraEase]

	Position Optional[mgl32.Vec3]

	Rotation Optional[mgl32.Vec2]

	Facing Optional[mgl32.Vec3]

	ViewOffset Optional[mgl32.Vec2]

	EntityOffset Optional[mgl32.Vec3]

	Default Optional[bool]

	IgnoreStartingValuesComponent bool
}

func (x *CameraInstructionSet) Marshal(r IO) {
	r.Uint32(&x.Preset)
	OptionalMarshaler(r, &x.Ease)
	OptionalFunc(r, &x.Position, r.Vec3)
	OptionalFunc(r, &x.Rotation, r.Vec2)
	OptionalFunc(r, &x.Facing, r.Vec3)
	OptionalFunc(r, &x.ViewOffset, r.Vec2)
	OptionalFunc(r, &x.EntityOffset, r.Vec3)
	OptionalFunc(r, &x.Default, r.Bool)
	r.Bool(&x.IgnoreStartingValuesComponent)
}

type CameraFadeTimeData struct {
	FadeInDuration float32

	WaitDuration float32

	FadeOutDuration float32
}

func (x *CameraFadeTimeData) Marshal(r IO) {
	r.Float32(&x.FadeInDuration)
	r.Float32(&x.WaitDuration)
	r.Float32(&x.FadeOutDuration)
}

type CameraInstructionFade struct {
	TimeData Optional[CameraFadeTimeData]

	Colour Optional[color.RGBA]
}

func (x *CameraInstructionFade) Marshal(r IO) {
	OptionalMarshaler(r, &x.TimeData)
	OptionalFunc(r, &x.Colour, r.RGB)
}

type CameraInstructionTarget struct {
	CenterOffset Optional[mgl32.Vec3]

	EntityUniqueID int64
}

func (x *CameraInstructionTarget) Marshal(r IO) {
	OptionalFunc(r, &x.CenterOffset, r.Vec3)
	r.ActorUniqueIDInt64(&x.EntityUniqueID)
}

type CameraInstructionFieldOfView struct {
	FieldOfView float32

	EaseTime float32

	EaseType int32

	Clear bool
}

func (x *CameraInstructionFieldOfView) Marshal(r IO) {
	easingType := easingTypeToString(x.EaseType)
	r.Float32(&x.FieldOfView)
	r.Float32(&x.EaseTime)
	r.String(&easingType)
	easingTypeFromString(r, &x.EaseType, easingType)
	r.Bool(&x.Clear)
}

type CameraPreset struct {
	Name string

	Parent string

	PosX Optional[float32]

	PosY Optional[float32]

	PosZ Optional[float32]

	RotX Optional[float32]

	RotY Optional[float32]

	RotationSpeed Optional[float32]

	SnapToTarget Optional[bool]

	HorizontalRotationLimit Optional[mgl32.Vec2]

	VerticalRotationLimit Optional[mgl32.Vec2]

	ContinueTargeting Optional[bool]

	TrackingRadius Optional[float32]

	ViewOffset Optional[mgl32.Vec2]

	EntityOffset Optional[mgl32.Vec3]

	Radius Optional[float32]

	MinYawLimit Optional[float32]

	MaxYawLimit Optional[float32]

	AudioListener Optional[byte]

	PlayerEffects Optional[bool]

	AimAssist Optional[CameraPresetAimAssist]

	ControlScheme Optional[byte]

	ApplyInheritedStartingRotation bool

	StartingRotation Optional[mgl32.Vec2]
}

func (x *CameraPreset) Marshal(r IO) {
	r.String(&x.Name)
	r.String(&x.Parent)
	OptionalFunc(r, &x.PosX, r.Float32)
	OptionalFunc(r, &x.PosY, r.Float32)
	OptionalFunc(r, &x.PosZ, r.Float32)
	OptionalFunc(r, &x.RotX, r.Float32)
	OptionalFunc(r, &x.RotY, r.Float32)
	OptionalFunc(r, &x.RotationSpeed, r.Float32)
	OptionalFunc(r, &x.SnapToTarget, r.Bool)
	OptionalFunc(r, &x.HorizontalRotationLimit, r.Vec2)
	OptionalFunc(r, &x.VerticalRotationLimit, r.Vec2)
	OptionalFunc(r, &x.ContinueTargeting, r.Bool)
	OptionalFunc(r, &x.TrackingRadius, r.Float32)
	OptionalFunc(r, &x.ViewOffset, r.Vec2)
	OptionalFunc(r, &x.EntityOffset, r.Vec3)
	OptionalFunc(r, &x.Radius, r.Float32)
	OptionalFunc(r, &x.MinYawLimit, r.Float32)
	OptionalFunc(r, &x.MaxYawLimit, r.Float32)
	OptionalFunc(r, &x.AudioListener, r.Uint8)
	OptionalFunc(r, &x.PlayerEffects, r.Bool)
	OptionalMarshaler(r, &x.AimAssist)
	OptionalFunc(r, &x.ControlScheme, r.Uint8)
	r.Bool(&x.ApplyInheritedStartingRotation)
	OptionalFunc(r, &x.StartingRotation, r.Vec2)
}

type CameraPresetAimAssist struct {
	Preset Optional[string]

	TargetMode Optional[int32]

	Angle Optional[mgl32.Vec2]

	Distance Optional[float32]
}

func (x *CameraPresetAimAssist) Marshal(r IO) {
	OptionalFunc(r, &x.Preset, r.String)
	OptionalFunc(r, &x.TargetMode, r.Int32)
	OptionalFunc(r, &x.Angle, r.Vec2)
	OptionalFunc(r, &x.Distance, r.Float32)
}

type CameraAimAssistCategory struct {
	Name string

	Priorities CameraAimAssistPriorities
}

func (x *CameraAimAssistCategory) Marshal(r IO) {
	r.String(&x.Name)
	Single(r, &x.Priorities)
}

type CameraAimAssistPriorities struct {
	Entities []CameraAimAssistPriority

	Blocks []CameraAimAssistPriority

	BlockTags []CameraAimAssistPriority

	EntityTypeFamilies []CameraAimAssistPriority

	EntityDefault Optional[int32]

	BlockDefault Optional[int32]
}

func (x *CameraAimAssistPriorities) Marshal(r IO) {
	Slice(r, &x.Entities)
	Slice(r, &x.Blocks)
	Slice(r, &x.BlockTags)
	Slice(r, &x.EntityTypeFamilies)
	OptionalFunc(r, &x.EntityDefault, r.Int32)
	OptionalFunc(r, &x.BlockDefault, r.Int32)
}

type CameraAimAssistPriority struct {
	Identifier string

	Priority int32
}

func (x *CameraAimAssistPriority) Marshal(r IO) {
	r.String(&x.Identifier)
	r.Int32(&x.Priority)
}

type CameraAimAssistPreset struct {
	Identifier string

	BlockExclusions []string

	EntityExclusions []string

	BlockTagExclusions []string

	EntityTypeFamilyExclusions []string

	LiquidTargets []string

	ItemSettings []CameraAimAssistItemSettings

	DefaultItemSettings Optional[string]

	HandSettings Optional[string]
}

func (x *CameraAimAssistPreset) Marshal(r IO) {
	r.String(&x.Identifier)
	FuncSlice(r, &x.BlockExclusions, r.String)
	FuncSlice(r, &x.EntityExclusions, r.String)
	FuncSlice(r, &x.BlockTagExclusions, r.String)
	FuncSlice(r, &x.EntityTypeFamilyExclusions, r.String)
	FuncSlice(r, &x.LiquidTargets, r.String)
	Slice(r, &x.ItemSettings)
	OptionalFunc(r, &x.DefaultItemSettings, r.String)
	OptionalFunc(r, &x.HandSettings, r.String)
}

type CameraAimAssistItemSettings struct {
	Item string

	Category string
}

func (x *CameraAimAssistItemSettings) Marshal(r IO) {
	r.String(&x.Item)
	r.String(&x.Category)
}

type CameraRotationOption struct {
	Value mgl32.Vec3

	Time float32

	EaseType int32
}

func (x *CameraRotationOption) Marshal(r IO) {
	easingType := easingTypeToString(x.EaseType)
	r.Vec3(&x.Value)
	r.Float32(&x.Time)
	r.String(&easingType)
	easingTypeFromString(r, &x.EaseType, easingType)
}

type CameraProgressOption struct {
	Value float32

	Time float32

	EaseType int32
}

func (x *CameraProgressOption) Marshal(r IO) {
	easingType := easingTypeToString(x.EaseType)
	r.Float32(&x.Value)
	r.Float32(&x.Time)
	r.String(&easingType)
	easingTypeFromString(r, &x.EaseType, easingType)
}

type CameraSplineInstruction struct {
	TotalTime float32

	SplineType uint8

	Curve []mgl32.Vec3

	ProgressKeyFrames []CameraProgressOption

	RotationOptions []CameraRotationOption

	SplineIdentifier string

	LoadFromJson bool
}

func (x *CameraSplineInstruction) Marshal(r IO) {
	r.Float32(&x.TotalTime)
	r.Uint8(&x.SplineType)
	FuncSlice(r, &x.Curve, r.Vec3)
	Slice(r, &x.ProgressKeyFrames)
	Slice(r, &x.RotationOptions)
	r.String(&x.SplineIdentifier)
	r.Bool(&x.LoadFromJson)
}

type CameraSplineDefinition struct {
	Name string

	TotalTime float32

	SplineType string

	ControlPoints []mgl32.Vec3

	ProgressKeyFrames []CameraSplineProgressKeyFrame

	RotationKeyFrames []CameraSplineRotationKeyFrame
}

func (x *CameraSplineDefinition) Marshal(r IO) {
	r.String(&x.Name)
	r.Float32(&x.TotalTime)
	r.String(&x.SplineType)
	FuncSlice(r, &x.ControlPoints, r.Vec3)
	Slice(r, &x.ProgressKeyFrames)
	Slice(r, &x.RotationKeyFrames)
}

type CameraSplineProgressKeyFrame struct {
	Value float32

	Time float32

	EaseType Optional[int32]
}

func (x *CameraSplineProgressKeyFrame) Marshal(r IO) {
	r.Float32(&x.Value)
	r.Float32(&x.Time)
	optionalEasingType(r, &x.EaseType)
}

type CameraSplineRotationKeyFrame struct {
	Value mgl32.Vec3

	Time float32

	EaseType Optional[int32]
}

func (x *CameraSplineRotationKeyFrame) Marshal(r IO) {
	r.Vec3(&x.Value)
	r.Float32(&x.Time)
	optionalEasingType(r, &x.EaseType)
}

func optionalEasingType(r IO, x *Optional[int32]) {
	var name Optional[string]
	if easingType, ok := x.Value(); ok {
		name = Option(easingTypeToString(easingType))
	}
	OptionalFunc(r, &name, r.String)
	*x = Optional[int32]{}
	if s, ok := name.Value(); ok {
		var easingType int32
		easingTypeFromString(r, &easingType, s)
		*x = Option(easingType)
	}
}

type CameraAimAssistActorPriorityData struct {
	PresetIndex int32

	CategoryIndex int32

	ActorIndex int32

	Priority int32
}

func (x *CameraAimAssistActorPriorityData) Marshal(r IO) {
	r.Int32(&x.PresetIndex)
	r.Int32(&x.CategoryIndex)
	r.Int32(&x.ActorIndex)
	r.Int32(&x.Priority)
}
