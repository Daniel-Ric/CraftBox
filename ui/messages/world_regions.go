package messages

import (
	"errors"
	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"strings"
	"sync"
)

type ChunkBounds struct {
	MinX int32 `json:"minX"`
	MinZ int32 `json:"minZ"`
	MaxX int32 `json:"maxX"`
	MaxZ int32 `json:"maxZ"`
}

func (b ChunkBounds) Contains(x, z int32) bool {
	return x >= b.MinX && x <= b.MaxX && z >= b.MinZ && z <= b.MaxZ
}
func (b ChunkBounds) Overlaps(other ChunkBounds) bool {
	return b.MinX <= other.MaxX && b.MaxX >= other.MinX && b.MinZ <= other.MaxZ && b.MaxZ >= other.MinZ
}

type CreateWorldRegion struct {
	WorldID string      `json:"worldId"`
	Name    string      `json:"name"`
	Bounds  ChunkBounds `json:"bounds"`
	Move    bool        `json:"move"`
}

func (r CreateWorldRegion) Validate() error {
	if strings.TrimSpace(r.WorldID) == "" || strings.TrimSpace(r.Name) == "" || len([]rune(r.Name)) > 80 {
		return errors.New("A world and a name of up to 80 characters are required")
	}
	if r.Bounds.MinX > r.Bounds.MaxX || r.Bounds.MinZ > r.Bounds.MaxZ {
		return errors.New("Invalid region bounds")
	}
	return nil
}

type WorldRegion struct {
	ID       string      `json:"id"`
	ParentID string      `json:"parentId"`
	Name     string      `json:"name"`
	Bounds   ChunkBounds `json:"bounds"`
	Move     bool        `json:"move"`
}

type EventWorldRegionCreated struct{ Region WorldRegion }

var worldEditors sync.Map
var metadataEditors sync.Map

func RegisterMetadataEditor(id string, edit func(EditWorld) error) { metadataEditors.Store(id, edit) }
func RequestWorldMetadata(parentID string, request EditWorld) error {
	edit, ok := metadataEditors.Load(parentID)
	if !ok {
		return errors.New("This world is no longer being captured. Wait for the completed world archive.")
	}
	return edit.(func(EditWorld) error)(request)
}

type EditWorld struct {
	WorldID string               `json:"worldId"`
	Name    *string              `json:"name,omitempty"`
	Spawn   *[3]int32            `json:"spawn,omitempty"`
	Export  *worldexport.Options `json:"export,omitempty"`
}

var worldMetadata sync.Map

func WorldMetadata(id string) EditWorld {
	value, ok := worldMetadata.Load(id)
	if !ok {
		return EditWorld{WorldID: id}
	}
	return value.(EditWorld)
}
func ClearWorldSpawn(id string) {
	edit := WorldMetadata(id)
	edit.Spawn = nil
	worldMetadata.Store(id, edit)
}
func StoreWorldMetadata(request EditWorld) {
	old := WorldMetadata(request.WorldID)
	if request.Name != nil {
		old.Name = request.Name
	}
	if request.Spawn != nil {
		old.Spawn = request.Spawn
	}
	if request.Export != nil {
		old.Export = request.Export
	}
	worldMetadata.Store(request.WorldID, old)
}

func RegisterWorldEditor(id string, create func(CreateWorldRegion) (WorldRegion, error)) {
	worldEditors.Store(id, create)
}
func UnregisterWorldEditor(id string) { worldEditors.Delete(id); metadataEditors.Delete(id) }
func RequestWorldRegion(request CreateWorldRegion) (WorldRegion, error) {
	if err := request.Validate(); err != nil {
		return WorldRegion{}, err
	}
	create, ok := worldEditors.Load(request.WorldID)
	if !ok {
		return WorldRegion{}, errors.New("Regions can be created in the world currently being captured")
	}
	return create.(func(CreateWorldRegion) (WorldRegion, error))(request)
}
