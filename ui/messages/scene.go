package messages

import (
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type EventSceneAvailable struct {
	ResourcePacks []*resource.Pack
	BlockEntries  []protocol.BlockEntry
	WorldID       string
	Store         *scene.Store
	Retained      bool
}
