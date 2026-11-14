package minecraft

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type GameData struct {
	WorldName string

	WorldSeed int64

	Difficulty int32

	EntityUniqueID int64

	EntityRuntimeID uint64

	PlayerGameMode int32

	PersonaDisabled bool

	CustomSkinsDisabled bool

	EmoteChatMuted bool

	BaseGameVersion          string
	GameVersion              string
	MultiPlayerCorrelationID string

	PlayerPosition mgl32.Vec3

	Pitch float32

	Yaw float32

	Dimension int32

	WorldSpawn protocol.BlockPos

	EditorWorldType int32

	CreatedInEditor bool

	ExportedFromEditor bool

	WorldGameMode int32

	Hardcore bool

	XBLBroadcastMode      int32
	PlatformBroadcastMode int32
	GameRules             []protocol.GameRule

	Time int64

	DayCycleLockTime int32

	ServerBlockStateChecksum uint64

	CustomBlocks []protocol.BlockEntry

	Items []protocol.ItemEntry

	PlayerMovementSettings protocol.PlayerMovementSettings

	ServerAuthoritativeInventory bool

	Experiments []protocol.ExperimentData

	PlayerPermissions byte

	ChunkRadius int32

	ClientSideGeneration bool

	ChatRestrictionLevel uint8

	DisablePlayerInteractions bool

	UseBlockNetworkIDHashes bool

	PropertyData map[string]any

	Dimensions []protocol.DimensionDefinition
}
