package world

import (
	"sync/atomic"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type capturedItem struct{ name string }

func (i capturedItem) EncodeItem() (string, int16) { return i.name, 0 }

type capturedItemRegistry struct {
	names map[int32]string
	ids   map[string]int32
}

var capturedItems atomic.Pointer[capturedItemRegistry]

func InsertCustomItems(entries []protocol.ItemEntry) {
	r := &capturedItemRegistry{names: make(map[int32]string, len(entries)), ids: make(map[string]int32, len(entries))}
	for _, entry := range entries {
		r.names[int32(entry.RuntimeID)] = entry.Name
		r.ids[entry.Name] = int32(entry.RuntimeID)
	}
	capturedItems.Store(r)
}

func ItemRidByName(name string) (int32, bool) {
	if r := capturedItems.Load(); r != nil {
		id, ok := r.ids[name]
		return id, ok
	}
	id, ok := itemNamesToRuntimeIDs[name]
	return id, ok
}
