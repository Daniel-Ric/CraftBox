package protocol

import (
	"github.com/go-gl/mathgl/mgl32"
)

const (
	MemoryCategoryUnknown = iota
	MemoryCategoryInvalidSizeUnknown
	MemoryCategoryActor
	MemoryCategoryActorAnimation
	MemoryCategoryActorRendering
	MemoryCategoryBalancer
	MemoryCategoryBlockTickingQueues
	MemoryCategoryBiomeStorage
	MemoryCategoryBlobs
	MemoryCategoryCereal
	MemoryCategoryCircuitSystem
	MemoryCategoryClient
	MemoryCategoryCommands
	MemoryCategoryDBStorage
	MemoryCategoryDebug
	MemoryCategoryDocumentation
	MemoryCategoryECSSystems
	MemoryCategoryFMOD
	MemoryCategoryFonts
	MemoryCategoryImGUI
	MemoryCategoryInput
	MemoryCategoryJsonUI
	MemoryCategoryJsonUIControlFactoryJson
	MemoryCategoryJsonUIControlTree
	MemoryCategoryJsonUIControlTreeControlElement
	MemoryCategoryJsonUIControlTreePopulateDataBinding
	MemoryCategoryJsonUIControlTreePopulateFocus
	MemoryCategoryJsonUIControlTreePopulateLayout
	MemoryCategoryJsonUIControlTreePopulateOther
	MemoryCategoryJsonUIControlTreePopulateSprite
	MemoryCategoryJsonUIControlTreePopulateText
	MemoryCategoryJsonUIControlTreePopulateTTS
	MemoryCategoryJsonUIControlTreeVisibility
	MemoryCategoryJsonUICreateUI
	MemoryCategoryJsonUIDefs
	MemoryCategoryJsonUILayoutManager
	MemoryCategoryJsonUILayoutManagerRemoveDependencies
	MemoryCategoryJsonUILayoutManagerInitVariable
	MemoryCategoryLanguages
	MemoryCategoryLevel
	MemoryCategoryLevelStructures
	MemoryCategoryLevelChunk
	MemoryCategoryLevelChunkGen
	MemoryCategoryLevelChunkGenThreadLocal
	MemoryCategoryNetwork
	MemoryCategoryMarketplace
	MemoryCategoryMaterialDragonCompiledDefinition
	MemoryCategoryMaterialDragonMaterial
	MemoryCategoryMaterialDragonResource
	MemoryCategoryMaterialDragonUniformMap
	MemoryCategoryMaterialRenderMaterial
	MemoryCategoryMaterialRenderMaterialGroup
	MemoryCategoryMaterialVariationManager
	MemoryCategoryMolang
	MemoryCategoryOreUI
	MemoryCategoryOreUIClient
	MemoryCategoryPersonaPieces
	MemoryCategoryPersonaAnimations
	MemoryCategoryPersonaCharacters
	MemoryCategoryPersonaSkinPacks
	MemoryCategoryPersonaRepo
	MemoryCategoryPlayer
	MemoryCategoryRenderChunk
	MemoryCategoryRenderChunkIndexBuffer
	MemoryCategoryRenderChunkVertexBuffer
	MemoryCategoryRendering
	MemoryCategoryRenderingBGFXInit
	MemoryCategoryRenderingBGFXStartFrame
	MemoryCategoryRenderingBlockTessellator
	MemoryCategoryRenderingEndFrame
	MemoryCategoryRenderingGraphicsTasksInit
	MemoryCategoryRenderingLibrary
	MemoryCategoryRenderingPolygonOperatorPool
	MemoryCategoryRenderingPBRTextureData
	MemoryCategoryRenderingRenderRegistry
	MemoryCategoryRenderingSetup
	MemoryCategoryRenderingVertices
	MemoryCategoryRequestLog
	MemoryCategoryResourcePacks
	MemoryCategorySound
	MemoryCategorySubChunkBiomeData
	MemoryCategorySubChunkBlockData
	MemoryCategorySubChunkLightData
	MemoryCategoryTextures
	MemoryCategoryWeatherRenderer
	MemoryCategoryWorldGenerator
	MemoryCategoryTasks
	MemoryCategoryTest
	MemoryCategoryTestLoadTestTags
	MemoryCategoryScripting
	MemoryCategoryScriptingRuntime
	MemoryCategoryScriptingContext
	MemoryCategoryScriptingContextBindingsMC
	MemoryCategoryScriptingContextBindingsGT
	MemoryCategoryScriptingContextRun
	MemoryCategoryDataDrivenUI
	MemoryCategoryDataDrivenUIDefs
	MemoryCategoryGameface
	MemoryCategoryGamefaceSystem
	MemoryCategoryGamefaceDOM
	MemoryCategoryGamefaceCSS
	MemoryCategoryGamefaceDisplay
	MemoryCategoryGamefaceTempAllocator
	MemoryCategoryGamefacePoolAllocator
	MemoryCategoryGamefaceDump
	MemoryCategoryGamefaceMedia
	MemoryCategoryGamefaceJSON
	MemoryCategoryGamefaceScriptEngine
	MemoryCategoryGamefaceScript
	MemoryCategoryGamefaceLayout
	MemoryCategoryVR
)

type MemoryCategoryCounter struct {
	Category uint8

	Bytes uint64
}

func (x *MemoryCategoryCounter) Marshal(r IO) {
	r.Uint8(&x.Category)
	r.Uint64(&x.Bytes)
}

type EntityDiagnosticTimingInfo struct {
	DisplayName string

	Entity string

	DurationNanos uint64

	PercentOfTotal byte

	Position mgl32.Vec3

	Dimension string
}

func (x *EntityDiagnosticTimingInfo) Marshal(r IO) {
	r.String(&x.DisplayName)
	r.String(&x.Entity)
	r.Uint64(&x.DurationNanos)
	r.Uint8(&x.PercentOfTotal)
	r.Vec3(&x.Position)
	r.String(&x.Dimension)
}

type SystemDiagnosticTimingInfo struct {
	DisplayName string

	SystemIndex uint64

	DurationNanos uint64

	PercentOfTotal byte
}

func (x *SystemDiagnosticTimingInfo) Marshal(r IO) {
	r.String(&x.DisplayName)
	r.Uint64(&x.SystemIndex)
	r.Uint64(&x.DurationNanos)
	r.Uint8(&x.PercentOfTotal)
}

type SystemCategory struct {
	CategoryName string
	SystemIndex  uint64
}

func (x *SystemCategory) Marshal(r IO) {
	r.String(&x.CategoryName)
	r.Uint64(&x.SystemIndex)
}

type WhiskerScopeDataSummary struct {
	Label string

	Indentation string

	TotalHighCostNS uint64

	TotalMidCostNS uint64

	TotalLowCostNS uint64
}

func (x *WhiskerScopeDataSummary) Marshal(r IO) {
	r.String(&x.Label)
	r.String(&x.Indentation)
	r.Uint64(&x.TotalHighCostNS)
	r.Uint64(&x.TotalMidCostNS)
	r.Uint64(&x.TotalLowCostNS)
}
