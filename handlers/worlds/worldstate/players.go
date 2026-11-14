package worldstate

import (
	"maps"
	"strings"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/utils/resourcepack"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const PlayerEntityType = "minecraft:player"

const PlayerEyeHeight = 1.62

type player struct {
	UUID      uuid.UUID
	Username  string
	RuntimeID entity.RuntimeID
}

func (w *World) AddPlayer(pk *packet.AddPlayer) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	w.players[pk.UUID] = &player{
		UUID:      pk.UUID,
		Username:  pk.Username,
		RuntimeID: pk.EntityRuntimeID,
	}

	ent := w.memState.GetEntity(pk.EntityRuntimeID)
	if ent == nil {
		ent = &entity.Entity{
			RuntimeID:            pk.EntityRuntimeID,
			Inventory:            make(map[byte]map[byte]protocol.ItemInstance),
			Metadata:             make(map[uint32]any),
			Properties:           make(map[string]any),
			PropertiesOverridden: make(map[string]any),
		}
	}
	ent.UniqueID = pk.AbilityData.EntityUniqueID
	if ent.UniqueID == 0 {
		ent.UniqueID = int64(pk.EntityRuntimeID)
	}
	ent.EntityType = PlayerEntityType
	ent.Position = pk.Position
	ent.Pitch = pk.Pitch
	ent.Yaw = pk.Yaw
	ent.HeadYaw = pk.HeadYaw
	ent.Velocity = pk.Velocity
	ent.DeletedDistance = -1
	ent.LastTeleport = 0
	maps.Copy(ent.Metadata, pk.EntityMetadata)

	for _, el := range pk.EntityLinks {
		w.memState.AddEntityLink(el)
	}
	if pk.HeldItem.Stack.NetworkID != 0 {
		held := pk.HeldItem
		ent.MainHand = &held
	}
	w.memState.StoreEntity(pk.EntityRuntimeID, ent)
	w.updateSceneActor(ent)
}

func PlayerEntityID(id uuid.UUID) string {
	return "craftbox_player:p_" + strings.ReplaceAll(id.String(), "-", "_")
}

type SavedPlayer struct {
	resourcepack.EntityPlayer
	Metadata protocol.EntityMetadata
}

func (w *World) preparePlayers(keep func(uuid.UUID) bool) (out []SavedPlayer) {
	byRuntimeID := make(map[entity.RuntimeID]*player, len(w.players))
	for _, p := range w.players {
		byRuntimeID[p.RuntimeID] = p
	}
	for rid, ent := range w.memState.entities {
		if ent.EntityType != PlayerEntityType {
			continue
		}
		p := byRuntimeID[rid]
		if p == nil || keep == nil || !keep(p.UUID) {
			delete(w.memState.entities, rid)
			continue
		}
		identifier := PlayerEntityID(p.UUID)
		ent.EntityType = identifier
		if _, ok := ent.Metadata[protocol.EntityDataKeyName]; !ok {
			ent.Metadata[protocol.EntityDataKeyName] = p.Username
			ent.Metadata.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAlwaysShowName)
			ent.Metadata.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagShowName)
		}
		out = append(out, SavedPlayer{
			EntityPlayer: resourcepack.EntityPlayer{Identifier: identifier, UUID: p.UUID},
			Metadata:     ent.Metadata,
		})
	}
	return out
}
