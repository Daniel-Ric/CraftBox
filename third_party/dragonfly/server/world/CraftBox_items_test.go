package world

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestCapturedCustomItemRuntimeIDs(t *testing.T) {
	previous := capturedItems.Load()
	t.Cleanup(func() { capturedItems.Store(previous) })
	InsertCustomItems([]protocol.ItemEntry{{Name: "test:captured_item", RuntimeID: 31000}})
	item, ok := ItemByRuntimeID(31000, 0)
	if !ok {
		t.Fatal("captured item missing")
	}
	name, meta := item.EncodeItem()
	if name != "test:captured_item" || meta != 0 {
		t.Fatalf("item = %q, %d", name, meta)
	}
	if id, ok := ItemRidByName("test:captured_item"); !ok || id != 31000 {
		t.Fatalf("runtime ID = %d, %v", id, ok)
	}
	InsertCustomItems([]protocol.ItemEntry{{Name: "test:captured_item", RuntimeID: 31001}})
	if _, ok := ItemByRuntimeID(31000, 0); ok {
		t.Fatal("previous server runtime ID leaked")
	}
	if _, ok := ItemByRuntimeID(31001, 0); !ok {
		t.Fatal("new runtime ID missing")
	}
}
