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

const (
	EnvironmentAttributePayloadTypeConstant = iota
	EnvironmentAttributePayloadTypeTransition
	EnvironmentAttributePayloadTypeNoiseTransition
)

type AttributeTransitionSettings struct {
	TotalTransitionTicks uint32

	CurrentTransitionTicks uint32

	EaseType int32

	ClockName string
}

func (x *AttributeTransitionSettings) Marshal(r IO) {
	r.Varuint32(&x.TotalTransitionTicks)
	r.Varuint32(&x.CurrentTransitionTicks)
	r.Varint32(&x.EaseType)
	r.String(&x.ClockName)
}

type AttributeNoiseTransitionSettings struct {
	TotalTransitionTicks uint32

	CurrentTransitionTicks uint32

	EaseType int32

	ClockName string

	LocalTransitionTicks uint32

	NoiseName string

	NoiseAlignment NoiseAlignment
}

func (x *AttributeNoiseTransitionSettings) Marshal(r IO) {
	r.Varuint32(&x.TotalTransitionTicks)
	r.Varuint32(&x.CurrentTransitionTicks)
	r.Varint32(&x.EaseType)
	r.String(&x.ClockName)
	r.Varuint32(&x.LocalTransitionTicks)
	r.String(&x.NoiseName)
	Single(r, &x.NoiseAlignment)
}

type EnvironmentAttributeData struct {
	AttributeName string

	PayloadType uint32

	Attribute AttributeData

	FromAttribute AttributeData

	ToAttribute AttributeData

	TransitionSettings AttributeTransitionSettings

	NoiseTransitionSettings AttributeNoiseTransitionSettings
}

func (x *EnvironmentAttributeData) Marshal(r IO) {
	r.String(&x.AttributeName)
	r.Varuint32(&x.PayloadType)
	switch x.PayloadType {
	case EnvironmentAttributePayloadTypeConstant:
		Single(r, &x.Attribute)
	case EnvironmentAttributePayloadTypeTransition:
		Single(r, &x.FromAttribute)
		Single(r, &x.ToAttribute)
		Single(r, &x.TransitionSettings)
	case EnvironmentAttributePayloadTypeNoiseTransition:
		Single(r, &x.FromAttribute)
		Single(r, &x.ToAttribute)
		Single(r, &x.NoiseTransitionSettings)
	default:
		r.UnknownEnumOption(x.PayloadType, "environment attribute payload type")
	}
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

	DimensionID int32

	Settings AttributeLayerSettings

	EnvironmentAttributes []EnvironmentAttributeData
}

func (x *AttributeLayerData) Marshal(r IO) {
	r.String(&x.Name)
	r.Varint32(&x.DimensionID)
	Single(r, &x.Settings)
	Slice(r, &x.EnvironmentAttributes)
}
