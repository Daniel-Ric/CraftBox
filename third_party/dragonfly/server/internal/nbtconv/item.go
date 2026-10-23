package nbtconv

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
)

func InvFromNBT(inv *inventory.Inventory, items []any) {
	for _, itemData := range items {
		data, _ := itemData.(map[string]any)
		it := item.ReadNBT(data, nil)
		if it.Empty() {
			continue
		}
		_ = inv.SetItem(int(Uint8(data, "Slot")), it)
	}
}

func InvToNBT(inv *inventory.Inventory) []any {
	var items []any
	for index, i := range inv.Slots() {
		if i.Empty() {
			continue
		}
		data := item.WriteNBT(i, true)
		data["Slot"] = byte(index)
		items = append(items, data)
	}
	return items
}
