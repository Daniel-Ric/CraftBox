package worldstate

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"testing"
)

func TestSceneFiltersPlayerAttachmentWithDistinctRuntimeAndUniqueIDs(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := scene.NewStore(reg, scene.NewCatalog())
	defer s.Close()
	w := &World{scene: s, memState: newWorldState(), players: map[uuid.UUID]*player{}}
	w.AddPlayer(&packet.AddPlayer{UUID: uuid.New(), EntityRuntimeID: 1})
	metadata := protocol.NewEntityMetadata()
	metadata[protocol.EntityDataKeyName] = "attached model"
	metadata[protocol.EntityDataKeyAlwaysShowNameTag] = int32(1)
	w.StoreEntity(2, &entity.Entity{UniqueID: 222, RuntimeID: 2, EntityType: "test:attachment", Metadata: metadata})
	w.AddEntityLink(protocol.EntityLink{RiddenEntityUniqueID: 501, RiderEntityUniqueID: 222, Type: protocol.EntityLinkPassenger})
	w.RegisterScenePlayerID(501)
	snap, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, chunk := range snap.Chunks {
		if len(chunk.Parts) > 0 {
			t.Fatal("runtime/unique mismatch leaked player attachment")
		}
	}
}

func TestCustomEntityCaptureCanBeDisabled(t *testing.T) {
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	s := scene.NewStore(reg, scene.NewCatalog())
	defer s.Close()
	w := &World{scene: s, memState: newWorldState(), SkipCustomEntities: true}
	md := protocol.NewEntityMetadata()
	md[protocol.EntityDataKeyName] = "NPC"
	md[protocol.EntityDataKeyAlwaysShowNameTag] = int32(1)
	w.StoreEntity(1, &entity.Entity{RuntimeID: 1, UniqueID: 1, EntityType: "test:npc", Metadata: md})
	if len(w.memState.entities) != 0 {
		t.Fatal("disabled custom entity was recorded")
	}
	err := w.ActEntity(2, true, func(e *entity.Entity) error { e.EntityType = "test:npc"; e.UniqueID = 2; e.Metadata = md; return nil })
	if err != nil || len(w.memState.entities) != 0 {
		t.Fatal("packet path recorded disabled custom entity")
	}
	w.SkipCustomEntities = false
	w.StoreEntity(1, &entity.Entity{RuntimeID: 1, UniqueID: 1, EntityType: "test:npc", Metadata: md})
	if len(w.memState.entities) != 1 {
		t.Fatal("enabled custom entity missing")
	}
}
