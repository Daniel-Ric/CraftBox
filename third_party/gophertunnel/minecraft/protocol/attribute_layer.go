package protocol

const (
	AttributeLayerPayloadTypeUpdateLayers = iota
	AttributeLayerPayloadTypeUpdateSettings
	AttributeLayerPayloadTypeUpdateEnvironment
	AttributeLayerPayloadTypeRemoveEnvironment
)

const (
	AttributeDataTypeBool = iota
	AttributeDataTypeFloat
	AttributeDataTypeColour
)

const (
	AttributeBoolOperationOverride = iota
	AttributeBoolOperationAlphaBlend
	AttributeBoolOperationAnd
	AttributeBoolOperationNand
	AttributeBoolOperationOr
	AttributeBoolOperationNor
	AttributeBoolOperationXor
	AttributeBoolOperationXnor
)

const (
	AttributeFloatOperationOverride = iota
	AttributeFloatOperationAlphaBlend
	AttributeFloatOperationAdd
	AttributeFloatOperationSubtract
	AttributeFloatOperationMultiply
	AttributeFloatOperationMinimum
	AttributeFloatOperationMaximum
)

const (
	AttributeColourOperationOverride = iota
	AttributeColourOperationAlphaBlend
	AttributeColourOperationAdd
	AttributeColourOperationSubtract
	AttributeColourOperationMultiply
)

type AttributeData struct {
	Type uint32

	BoolValue bool

	BoolOperation Optional[int32]

	FloatValue float32

	FloatOperation Optional[int32]

	FloatConstraintMin Optional[float32]

	FloatConstraintMax Optional[float32]

	ColourValue int32

	ColourOperation Optional[int32]
}

func (x *AttributeData) Marshal(r IO) {
	r.Varuint32(&x.Type)
	switch x.Type {
	case AttributeDataTypeBool:
		r.Bool(&x.BoolValue)
		OptionalFunc(r, &x.BoolOperation, r.Int32)
	case AttributeDataTypeFloat:
		r.Float32(&x.FloatValue)
		OptionalFunc(r, &x.FloatOperation, r.Int32)
		OptionalFunc(r, &x.FloatConstraintMin, r.Float32)
		OptionalFunc(r, &x.FloatConstraintMax, r.Float32)
	case AttributeDataTypeColour:
		r.Int32(&x.ColourValue)
		OptionalFunc(r, &x.ColourOperation, r.Int32)
	default:
		r.UnknownEnumOption(x.Type, "attribute data type")
	}
}

const (
	NoiseAlignmentTypeMinLocalTransitionEnd = iota
)

type NoiseAlignment struct {
	Type byte

	Value uint32
}

func (x *NoiseAlignment) Marshal(r IO) {
	r.Uint8(&x.Type)
	r.Varuint32(&x.Value)
}

type EnvironmentAttributeData struct {
	AttributeName string

	FromAttribute Optional[AttributeData]

	Attribute AttributeData

	ToAttribute Optional[AttributeData]

	CurrentTransitionTicks uint32

	TotalTransitionTicks uint32

	EaseType int32

	LocalTransitionTicks uint32

	NoiseTransition bool

	NoiseAlignment NoiseAlignment
}

func (x *EnvironmentAttributeData) Marshal(r IO) {
	easingType := easingTypeToString(x.EaseType)
	r.String(&x.AttributeName)
	OptionalMarshaler(r, &x.FromAttribute)
	Single(r, &x.Attribute)
	OptionalMarshaler(r, &x.ToAttribute)
	r.Uint32(&x.CurrentTransitionTicks)
	r.Uint32(&x.TotalTransitionTicks)
	r.String(&easingType)
	easingTypeFromString(r, &x.EaseType, easingType)
	r.Uint32(&x.LocalTransitionTicks)
	r.Bool(&x.NoiseTransition)
	Single(r, &x.NoiseAlignment)
}

type AttributeLayerSettings struct {
	Priority int32

	FloatWeight float32

	Enabled bool

	TransitionsPaused bool
}

func (x *AttributeLayerSettings) Marshal(r IO) {
	r.Int32(&x.Priority)
	r.Float32(&x.FloatWeight)
	r.Bool(&x.Enabled)
	r.Bool(&x.TransitionsPaused)
}

type AttributeLayerData struct {
	Name string

	NoiseName Optional[string]

	DimensionID int32

	Settings AttributeLayerSettings

	EnvironmentAttributes []EnvironmentAttributeData
}

func (x *AttributeLayerData) Marshal(r IO) {
	r.String(&x.Name)
	OptionalFunc(r, &x.NoiseName, r.String)
	r.Varint32(&x.DimensionID)
	Single(r, &x.Settings)
	Slice(r, &x.EnvironmentAttributes)
}
