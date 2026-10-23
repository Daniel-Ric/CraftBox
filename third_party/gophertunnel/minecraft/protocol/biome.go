package protocol

const (
	BiomeExpressionOpUnknown = iota - 1
	BiomeExpressionOpLeftBrace
	BiomeExpressionOpRightBrace
	BiomeExpressionOpLeftBracket
	BiomeExpressionOpRightBracket
	BiomeExpressionOpLeftParenthesis
	BiomeExpressionOpRightParenthesis
	BiomeExpressionOpNegate
	BiomeExpressionOpLogicalNot
	BiomeExpressionOpAbs
	BiomeExpressionOpAdd
	BiomeExpressionOpAcos
	BiomeExpressionOpAsin
	BiomeExpressionOpAtan
	BiomeExpressionOpAtan2
	BiomeExpressionOpCeil
	BiomeExpressionOpClamp
	BiomeExpressionOpCopySign
	BiomeExpressionCos
	BiomeExpressionDieRoll
	BiomeExpressionDieRollInt
	BiomeExpressionDiv
	BiomeExpressionExp
	BiomeExpressionFloor
	BiomeExpressionHermiteBlend
	BiomeExpressionLerp
	BiomeExpressionLerpRotate
	BiomeExpressionLn
	BiomeExpressionMax
	BiomeExpressionMin
	BiomeExpressionMinAngle
	BiomeExpressionMod
	BiomeExpressionMul
	BiomeExpressionPow
	BiomeExpressionRandom
	BiomeExpressionRandomInt
	BiomeExpressionRound
	BiomeExpressionSin
	BiomeExpressionSign
	BiomeExpressionSqrt
	BiomeExpressionTrunc
	BiomeExpressionQueryFunction
	BiomeExpressionArrayVariable
	BiomeExpressionContextVariable
	BiomeExpressionEntityVariable
	BiomeExpressionTempVariable
	BiomeExpressionMemberAccessor
	BiomeExpressionHashedStringHash
	BiomeExpressionGeometryVariable
	BiomeExpressionMaterialVariable
	BiomeExpressionTextureVariable
	BiomeExpressionLessThan
	BiomeExpressionLessEqual
	BiomeExpressionGreaterEqual
	BiomeExpressionGreaterThan
	BiomeExpressionLogicalEqual
	BiomeExpressionLogicalNotEqual
	BiomeExpressionLogicalOr
	BiomeExpressionLogicalAnd
	BiomeExpressionNullCoalescing
	BiomeExpressionConditional
	BiomeExpressionConditionalElse
	BiomeExpressionFloat
	BiomeExpressionPi
	BiomeExpressionArray
	BiomeExpressionGeometry
	BiomeExpressionMaterial
	BiomeExpressionTexture
	BiomeExpressionLoop
	BiomeExpressionForEach
	BiomeExpressionBreak
	BiomeExpressionContinue
	BiomeExpressionAssignment
	BiomeExpressionPointer
	BiomeExpressionSemicolon
	BiomeExpressionReturn
	BiomeExpressionComma
	BiomeExpressionThis
	BiomeExpressionNonEvaluatedArray
	BiomeExpressionInverseLerp
	BiomeExpressionEaseInQuad
	BiomeExpressionEaseOutQuad
	BiomeExpressionEaseInOutQuad
	BiomeExpressionEaseInCubic
	BiomeExpressionEaseOutCubic
	BiomeExpressionEaseInOutCubic
	BiomeExpressionEaseInQuart
	BiomeExpressionEaseOutQuart
	BiomeExpressionEaseInOutQuart
	BiomeExpressionEaseInQuint
	BiomeExpressionEaseOutQuint
	BiomeExpressionEaseInOutQuint
	BiomeExpressionEaseInSine
	BiomeExpressionEaseOutSine
	BiomeExpressionEaseInOutSine
	BiomeExpressionEaseInExpo
	BiomeExpressionEaseOutExpo
	BiomeExpressionEaseInOutExpo
	BiomeExpressionEaseInCirc
	BiomeExpressionEaseOutCirc
	BiomeExpressionEaseInOutCirc
	BiomeExpressionEaseInBounce
	BiomeExpressionEaseOutBounce
	BiomeExpressionEaseInOutBounce
	BiomeExpressionEaseInBack
	BiomeExpressionEaseOutBack
	BiomeExpressionEaseInOutBack
	BiomeExpressionEaseInElastic
	BiomeExpressionEaseOutElastic
	BiomeExpressionEaseInOutElastic
	BiomeExpressionCount
)

const (
	BiomeCoordinateEvaluationOrderXYZ = iota
	BiomeCoordinateEvaluationOrderXZY
	BiomeCoordinateEvaluationOrderYXZ
	BiomeCoordinateEvaluationOrderYZX
	BiomeCoordinateEvaluationOrderZXY
	BiomeCoordinateEvaluationOrderZYX
)

const (
	BiomeRandomDistributionTypeSingleValued = iota
	BiomeRandomDistributionTypeUniform
	BiomeRandomDistributionTypeGaussian
	BiomeRandomDistributionTypeInverseGaussian
	BiomeRandomDistributionTypeFixedGrid
	BiomeRandomDistributionTypeJitteredGrid
	BiomeRandomDistributionTypeTriangle
)

type BiomeDefinition struct {
	NameIndex int16

	BiomeID int16

	Temperature float32

	Downfall float32

	FoliageSnow float32

	Depth float32

	Scale float32

	MapWaterColour int32

	Rain bool

	Tags Optional[[]uint16]

	ChunkGeneration Optional[BiomeChunkGeneration]
}

func (x *BiomeDefinition) Marshal(r IO) {
	r.Int16(&x.NameIndex)
	r.Int16(&x.BiomeID)
	r.Float32(&x.Temperature)
	r.Float32(&x.Downfall)
	r.Float32(&x.FoliageSnow)
	r.Float32(&x.Depth)
	r.Float32(&x.Scale)
	r.Int32(&x.MapWaterColour)
	r.Bool(&x.Rain)
	OptionalFunc(r, &x.Tags, func(s *[]uint16) {
		FuncSlice(r, s, r.Uint16)
	})
	OptionalMarshaler(r, &x.ChunkGeneration)
}

type BiomeChunkGeneration struct {
	Climate Optional[BiomeClimate]

	ConsolidatedFeatures Optional[[]BiomeConsolidatedFeature]

	MountainParameters Optional[BiomeMountainParameters]

	SurfaceMaterialAdjustments Optional[[]BiomeElementData]

	SurfaceMaterials Optional[BiomeSurfaceMaterial]

	HasDefaultOverworldSurface bool

	HasSwampSurface bool

	HasFrozenOceanSurface bool

	HasEndSurface bool

	MesaSurface Optional[BiomeMesaSurface]

	CappedSurface Optional[BiomeCappedSurface]

	OverworldRules Optional[BiomeOverworldRules]

	MultiNoiseRules Optional[BiomeMultiNoiseRules]

	LegacyRules Optional[[]BiomeConditionalTransformation]

	ReplacementsData Optional[[]BiomeReplacementData]

	VillageType Optional[uint8]

	SurfaceBuilder Optional[BiomeSurfaceBuilder]

	SubsurfaceBuilder Optional[BiomeSurfaceBuilder]
}

func (x *BiomeChunkGeneration) Marshal(r IO) {
	OptionalMarshaler(r, &x.Climate)
	OptionalFunc(r, &x.ConsolidatedFeatures, func(s *[]BiomeConsolidatedFeature) {
		Slice(r, s)
	})
	OptionalMarshaler(r, &x.MountainParameters)
	OptionalFunc(r, &x.SurfaceMaterialAdjustments, func(s *[]BiomeElementData) {
		Slice(r, s)
	})
	OptionalMarshaler(r, &x.OverworldRules)
	OptionalMarshaler(r, &x.MultiNoiseRules)
	OptionalFunc(r, &x.LegacyRules, func(s *[]BiomeConditionalTransformation) {
		Slice(r, s)
	})
	OptionalFunc(r, &x.ReplacementsData, func(s *[]BiomeReplacementData) {
		Slice(r, s)
	})
	OptionalFunc(r, &x.VillageType, r.Uint8)
	OptionalMarshaler(r, &x.SurfaceBuilder)
	OptionalMarshaler(r, &x.SubsurfaceBuilder)
}

type BiomeClimate struct {
	Temperature float32

	Downfall float32

	SnowAccumulationMin float32

	SnowAccumulationMax float32
}

func (x *BiomeClimate) Marshal(r IO) {
	r.Float32(&x.Temperature)
	r.Float32(&x.Downfall)
	r.Float32(&x.SnowAccumulationMin)
	r.Float32(&x.SnowAccumulationMax)
}

type BiomeConsolidatedFeature struct {
	Scatter BiomeScatterParameter

	Feature int16

	Identifier int16

	Pass int16

	CanUseInternal bool
}

func (x *BiomeConsolidatedFeature) Marshal(r IO) {
	Single(r, &x.Scatter)
	r.Int16(&x.Feature)
	r.Int16(&x.Identifier)
	r.Int16(&x.Pass)
	r.Bool(&x.CanUseInternal)
}

type BiomeScatterParameter struct {
	Coordinates []BiomeCoordinate

	EvaluationOrder int32

	ChancePercentType int32

	ChancePercent int16

	ChanceNumerator int32

	ChanceDenominator int32

	IterationsType int32

	Iterations int16
}

func (x *BiomeScatterParameter) Marshal(r IO) {
	Slice(r, &x.Coordinates)
	r.Varint32(&x.EvaluationOrder)
	r.Varint32(&x.ChancePercentType)
	r.Int16(&x.ChancePercent)
	r.Int32(&x.ChanceNumerator)
	r.Int32(&x.ChanceDenominator)
	r.Varint32(&x.IterationsType)
	r.Int16(&x.Iterations)
}

type BiomeCoordinate struct {
	MinValueType int32

	MinValue int16

	MaxValueType int32

	MaxValue int16

	GridOffset uint32

	GridStepSize uint32

	Distribution int32
}

func (x *BiomeCoordinate) Marshal(r IO) {
	r.Varint32(&x.MinValueType)
	r.Int16(&x.MinValue)
	r.Varint32(&x.MaxValueType)
	r.Int16(&x.MaxValue)
	r.Uint32(&x.GridOffset)
	r.Uint32(&x.GridStepSize)
	r.Varint32(&x.Distribution)
}

type BiomeMountainParameters struct {
	SteepBlock int32

	NorthSlopes bool

	SouthSlopes bool

	WestSlopes bool

	EastSlopes bool

	TopSlideEnabled bool
}

func (x *BiomeMountainParameters) Marshal(r IO) {
	r.Int32(&x.SteepBlock)
	r.Bool(&x.NorthSlopes)
	r.Bool(&x.SouthSlopes)
	r.Bool(&x.WestSlopes)
	r.Bool(&x.EastSlopes)
	r.Bool(&x.TopSlideEnabled)
}

type BiomeElementData struct {
	NoiseFrequencyScale float32

	NoiseLowerBound float32

	NoiseUpperBound float32

	HeightMinType int32

	HeightMin int16

	HeightMaxType int32

	HeightMax int16

	AdjustedMaterials BiomeSurfaceMaterial
}

func (x *BiomeElementData) Marshal(r IO) {
	r.Float32(&x.NoiseFrequencyScale)
	r.Float32(&x.NoiseLowerBound)
	r.Float32(&x.NoiseUpperBound)
	r.Varint32(&x.HeightMinType)
	r.Int16(&x.HeightMin)
	r.Varint32(&x.HeightMaxType)
	r.Int16(&x.HeightMax)
	Single(r, &x.AdjustedMaterials)
}

type BiomeSurfaceMaterial struct {
	TopBlock int32

	MidBlock int32

	SeaFloorBlock int32

	FoundationBlock int32

	SeaBlock int32

	SeaFloorDepth int32
}

func (x *BiomeSurfaceMaterial) Marshal(r IO) {
	r.Int32(&x.TopBlock)
	r.Int32(&x.MidBlock)
	r.Int32(&x.SeaFloorBlock)
	r.Int32(&x.FoundationBlock)
	r.Int32(&x.SeaBlock)
	r.Int32(&x.SeaFloorDepth)
}

type BiomeSurfaceBuilder struct {
	SurfaceMaterials Optional[BiomeSurfaceMaterial]

	HasDefaultOverworldSurface bool

	HasSwampSurface bool

	HasFrozenOceanSurface bool

	HasEndSurface bool

	MesaSurface Optional[BiomeMesaSurface]

	CappedSurface Optional[BiomeCappedSurface]

	NoiseGradientSurface Optional[BiomeNoiseGradientSurface]
}

func (x *BiomeSurfaceBuilder) Marshal(r IO) {
	OptionalMarshaler(r, &x.SurfaceMaterials)
	r.Bool(&x.HasDefaultOverworldSurface)
	r.Bool(&x.HasSwampSurface)
	r.Bool(&x.HasFrozenOceanSurface)
	r.Bool(&x.HasEndSurface)
	OptionalMarshaler(r, &x.MesaSurface)
	OptionalMarshaler(r, &x.CappedSurface)
	OptionalMarshaler(r, &x.NoiseGradientSurface)
}

type BiomeNoiseGradientSurface struct {
	NonReplaceableBlocks []uint32

	GradientBlocks []NoiseBlockSpecifier

	Noise NoiseDescriptor
}

func (x *BiomeNoiseGradientSurface) Marshal(r IO) {
	FuncSlice(r, &x.NonReplaceableBlocks, r.Uint32)
	Slice(r, &x.GradientBlocks)
	Single(r, &x.Noise)
}

type FloatRange struct {
	Min float32

	Max float32
}

func (x *FloatRange) Marshal(r IO) {
	r.Float32(&x.Min)
	r.Float32(&x.Max)
}

type NoiseBlockSpecifier struct {
	Noise string

	Threshold float32

	Range FloatRange

	Block uint32
}

func (x *NoiseBlockSpecifier) Marshal(r IO) {
	r.String(&x.Noise)
	r.Float32(&x.Threshold)
	Single(r, &x.Range)
	r.Uint32(&x.Block)
}

type NoiseDescriptor struct {
	Name string

	FirstOctave int32

	Amplitudes []float32
}

func (x *NoiseDescriptor) Marshal(r IO) {
	r.String(&x.Name)
	r.Int32(&x.FirstOctave)
	FuncSlice(r, &x.Amplitudes, r.Float32)
}

type BiomeMesaSurface struct {
	ClayMaterial uint32

	HardClayMaterial uint32

	BrycePillars bool

	HasForest bool
}

func (x *BiomeMesaSurface) Marshal(r IO) {
	r.Uint32(&x.ClayMaterial)
	r.Uint32(&x.HardClayMaterial)
	r.Bool(&x.BrycePillars)
	r.Bool(&x.HasForest)
}

type BiomeCappedSurface struct {
	FloorBlocks []int32

	CeilingBlocks []int32

	SeaBlock Optional[uint32]

	FoundationBlock Optional[uint32]

	BeachBlock Optional[uint32]
}

func (x *BiomeCappedSurface) Marshal(r IO) {
	FuncSlice(r, &x.FloorBlocks, r.Int32)
	FuncSlice(r, &x.CeilingBlocks, r.Int32)
	OptionalFunc(r, &x.SeaBlock, r.Uint32)
	OptionalFunc(r, &x.FoundationBlock, r.Uint32)
	OptionalFunc(r, &x.BeachBlock, r.Uint32)
}

type BiomeOverworldRules struct {
	HillsTransformations []BiomeWeight

	MutateTransformations []BiomeWeight

	RiverTransformations []BiomeWeight

	ShoreTransformations []BiomeWeight

	PreHillsEdgeTransformations []BiomeConditionalTransformation

	PostShoreEdgeTransformations []BiomeConditionalTransformation

	ClimateTransformations []BiomeTemperatureWeight
}

func (x *BiomeOverworldRules) Marshal(r IO) {
	Slice(r, &x.HillsTransformations)
	Slice(r, &x.MutateTransformations)
	Slice(r, &x.RiverTransformations)
	Slice(r, &x.ShoreTransformations)
	Slice(r, &x.PreHillsEdgeTransformations)
	Slice(r, &x.PostShoreEdgeTransformations)
	Slice(r, &x.ClimateTransformations)
}

type BiomeMultiNoiseRules struct {
	Temperature float32

	Humidity float32

	Altitude float32

	Weirdness float32

	Weight float32
}

func (x *BiomeMultiNoiseRules) Marshal(r IO) {
	r.Float32(&x.Temperature)
	r.Float32(&x.Humidity)
	r.Float32(&x.Altitude)
	r.Float32(&x.Weirdness)
	r.Float32(&x.Weight)
}

type BiomeConditionalTransformation struct {
	WeightedBiomes []BiomeWeight

	ConditionJSON int16

	MinPassingNeighbours uint32
}

func (x *BiomeConditionalTransformation) Marshal(r IO) {
	Slice(r, &x.WeightedBiomes)
	r.Int16(&x.ConditionJSON)
	r.Uint32(&x.MinPassingNeighbours)
}

type BiomeWeight struct {
	Biome int16

	Weight uint32
}

func (x *BiomeWeight) Marshal(r IO) {
	r.Int16(&x.Biome)
	r.Uint32(&x.Weight)
}

type BiomeTemperatureWeight struct {
	Temperature int32

	Weight uint32
}

func (x *BiomeTemperatureWeight) Marshal(r IO) {
	r.Varint32(&x.Temperature)
	r.Uint32(&x.Weight)
}

type BiomeReplacementData struct {
	Biome int16

	Dimension int16

	TargetBiomes []int16

	Amount float32

	NoiseFrequencyScale float32

	ReplacementIndex uint32
}

func (x *BiomeReplacementData) Marshal(r IO) {
	r.Int16(&x.Biome)
	r.Int16(&x.Dimension)
	FuncSlice(r, &x.TargetBiomes, r.Int16)
	r.Float32(&x.Amount)
	r.Float32(&x.NoiseFrequencyScale)
	r.Uint32(&x.ReplacementIndex)
}
