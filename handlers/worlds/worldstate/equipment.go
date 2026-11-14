package worldstate

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/nbtconv"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func emptySlot() map[string]any {
	return map[string]any{"Name": "", "Count": byte(0), "Damage": int16(0), "WasPickedUp": byte(0)}
}

func (w *World) itemNBT(it *protocol.ItemInstance) map[string]any {
	if it == nil || it.Stack.NetworkID == 0 || it.Stack.Count == 0 || w.BlockRegistry == nil {
		return emptySlot()
	}
	stack := utils.StackToItem(w.BlockRegistry, it.Stack)
	if stack.Empty() {
		return emptySlot()
	}
	return nbtconv.WriteItem(stack, true)
}

func (w *World) writeEquipment(ent *entity.Entity, data map[string]any) {
	slots := []*protocol.ItemInstance{ent.Helmet, ent.Chestplate, ent.Leggings, ent.Boots, ent.MainHand, ent.OffHand, ent.Body}
	worn := false
	for _, s := range slots {
		if s != nil && s.Stack.NetworkID != 0 {
			worn = true
			break
		}
	}
	if !worn {
		return
	}
	data["Armor"] = []map[string]any{w.itemNBT(ent.Helmet), w.itemNBT(ent.Chestplate), w.itemNBT(ent.Leggings), w.itemNBT(ent.Boots)}
	data["Mainhand"] = []map[string]any{w.itemNBT(ent.MainHand)}
	data["Offhand"] = []map[string]any{w.itemNBT(ent.OffHand)}
	data["Body"] = []map[string]any{w.itemNBT(ent.Body)}
}
