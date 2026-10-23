package merge

import (
	"github.com/df-mc/dragonfly/server/world"
	"testing"
)

func TestCustomBlockStateRoundTrip(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	registry := &BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]Block{}}
	props := map[string]any{"cc:facing": "east", "cc:top_bit": int32(1)}
	rid, ok := registry.StateToRuntimeID("cubecraft:portal_corner", props)
	if !ok {
		t.Fatal("missing runtime id")
	}
	name, got, ok := registry.RuntimeIDToState(rid)
	if !ok || name != "cubecraft:portal_corner" || got["cc:facing"] != "east" || got["cc:top_bit"] != int32(1) {
		t.Fatal(name, got, ok)
	}
	b, ok := registry.BlockByRuntimeID(rid)
	if !ok || registry.BlockRuntimeID(b) != rid {
		t.Fatal("custom block lost its state hash")
	}
}
