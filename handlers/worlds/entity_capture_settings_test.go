package worlds

import (
	"context"
	"testing"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestEntityCaptureRespectsSaveEntities(t *testing.T) {
	for _, tc := range []struct {
		name         string
		kind         string
		save, custom bool
		want         int
	}{
		{"nametag disabled", "minecraft:armor_stand", false, false, 0},
		{"custom cannot override disabled entities", "hive:nametag", false, true, 0},
		{"vanilla entity enabled", "minecraft:armor_stand", true, false, 1},
		{"custom disabled", "hive:nametag", true, false, 0},
		{"custom enabled", "hive:nametag", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := worldstate.New(context.Background(), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			handler := &worldsHandler{
				worldState: state, session: &proxy.Session{},
				settings:    WorldSettings{SaveEntities: tc.save, SaveCustomEntities: tc.custom},
				serverState: serverState{behaviorPack: behaviourpack.New("test")},
			}
			spawn := &packet.AddActor{
				EntityRuntimeID: 42, EntityUniqueID: 43, EntityType: tc.kind,
				EntityMetadata: protocol.EntityMetadata{protocol.EntityDataKeyName: "Limited Time Mode"},
			}
			forwarded, err := handler.packetHandlerIngame(spawn, false, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if forwarded != spawn {
				t.Fatal("capture settings must not hide entities from the Minecraft client")
			}
			// Metadata updates must not recreate a skipped entity.
			_, err = handler.packetHandlerIngame(&packet.SetActorData{
				EntityRuntimeID: 42,
				EntityMetadata:  protocol.EntityMetadata{protocol.EntityDataKeyName: "Updated nametag"},
			}, false, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			total, _ := state.EntityCounts(0)
			if total != tc.want {
				t.Fatalf("captured %d entities, want %d", total, tc.want)
			}
			if tc.want == 1 {
				err = state.ActEntity(42, false, func(ent *entity.Entity) error {
					if ent.Metadata[protocol.EntityDataKeyName] != "Updated nametag" {
						t.Error("enabled capture lost nametag updates")
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
