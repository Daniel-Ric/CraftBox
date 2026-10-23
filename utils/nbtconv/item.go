package nbtconv

import (
	"github.com/df-mc/dragonfly/server/item/inventory"
)


func InvFromNBT(inv *inventory.Inventory, items []any) {
	for _, itemData := range items {
		data, _ := itemData.(map[string]any)
		it := Item(data, nil)
		if it.Empty() {
			continue
		}
		_ = inv.SetItem(int(Uint8(data, "Slot")), it)
	}
}


func InvToNBT(inv *inventory.Inventory) []map[string]any {
	var items []map[string]any
	for index, i := range inv.Slots() {
		if i.Empty() {
			continue
		}
		data := WriteItem(i, true)
		data["Slot"] = byte(index)
		items = append(items, data)
	}
	return items
}
