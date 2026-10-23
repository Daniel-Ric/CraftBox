package world

import (
	_ "embed"
	"fmt"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"image"
)

type Item interface {
	EncodeItem() (name string, meta int16)
}

type CustomItem interface {
	Item

	Name() string

	Texture() image.Image

	Category() category.Category
}

func RegisterItem(item Item) {
	name, meta := item.EncodeItem()
	h := itemHash{name: name, meta: meta}

	if _, ok := items[h]; ok {
		panic(fmt.Sprintf("item registered with name %v and meta %v already exists", name, meta))
	}
	if c, ok := item.(CustomItem); ok {
		nextRID := int32(len(itemNamesToRuntimeIDs))
		itemRuntimeIDsToNames[nextRID] = name
		itemNamesToRuntimeIDs[name] = nextRID

		customItems = append(customItems, c)
	}
	if _, ok := itemNamesToRuntimeIDs[name]; !ok {
		panic(fmt.Sprintf("item name %v does not have a runtime ID", name))
	}
	items[h] = item
}

type itemHash struct {
	name string
	meta int16
}

var (
	//go:embed vanilla_items.nbt
	itemRuntimeIDData []byte

	items = map[itemHash]Item{}

	customItems []CustomItem

	itemRuntimeIDsToNames = map[int32]string{}

	itemNamesToRuntimeIDs = map[string]int32{}
)

func init() {
	var m map[string]struct {
		RuntimeID      int32          `nbt:"runtime_id"`
		ComponentBased bool           `nbt:"component_based"`
		Version        int32          `nbt:"version"`
		Data           map[string]any `nbt:"data,omitempty"`
	}
	err := nbt.Unmarshal(itemRuntimeIDData, &m)
	if err != nil {
		panic(err)
	}
	for name, e := range m {
		itemNamesToRuntimeIDs[name] = e.RuntimeID
		itemRuntimeIDsToNames[e.RuntimeID] = name
	}
}

func ItemByName(name string, meta int16) (Item, bool) {
	it, ok := items[itemHash{name: name, meta: meta}]
	if !ok {

		it, ok = items[itemHash{name: name}]
	}
	if !ok {
		if r := capturedItems.Load(); r != nil {
			if _, captured := r.ids[name]; captured {
				return capturedItem{name: name}, true
			}
		}
	}
	return it, ok
}

func ItemRuntimeID(i Item) (rid int32, meta int16, ok bool) {
	name, meta := i.EncodeItem()
	rid, ok = ItemRidByName(name)
	return rid, meta, ok
}

func ItemByRuntimeID(rid int32, meta int16) (Item, bool) {
	if r := capturedItems.Load(); r != nil {
		name, ok := r.names[rid]
		if !ok {
			return nil, false
		}
		return ItemByName(name, meta)
	}
	name, ok := itemRuntimeIDsToNames[rid]
	if !ok {
		return nil, false
	}
	return ItemByName(name, meta)
}

func Items() []Item {
	m := make([]Item, 0, len(items))
	for _, i := range items {
		m = append(m, i)
	}
	return m
}

func CustomItems() []CustomItem {
	return customItems
}
