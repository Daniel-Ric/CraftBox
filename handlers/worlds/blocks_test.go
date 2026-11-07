package worlds

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Wire shape from Farm It, 2026-11-05 12:27 UTC. The wire flag is
// multi_block, but the actual permutation state is multi_block_part.
func TestFriendWorldMultiBlockStates(t *testing.T) {
	for _, count := range []int32{2, 3, 4} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			reg := world.NewBlockRegistry()
			entry := protocol.BlockEntry{Name: "test:friend_multiblock", Properties: map[string]any{
				"traits": []any{map[string]any{
					"name": "minecraft:multi_block", "enabled_states": map[string]any{"multi_block": uint8(1)},
					"part_count": count, "direction": int32(1), "place_behavior": int32(1),
				}},
			}}
			if err := AddCustomBlocks(reg, []protocol.BlockEntry{entry}); err != nil {
				t.Fatal(err)
			}
			reg.Finalize()
			for part := int32(0); part < count; part++ {
				if _, ok := reg.BlockByName(entry.Name, map[string]any{"minecraft:multi_block_part": part}); !ok {
					t.Fatalf("missing part %d", part)
				}
			}
			if _, ok := reg.BlockByName(entry.Name, map[string]any{"minecraft:multi_block_part": count}); ok {
				t.Fatal("registered nonexistent part")
			}
		})
	}
}

func TestFriendWorldNetworkHashCollisionKeepsDistinctStates(t *testing.T) {
	reg := world.NewBlockRegistry()
	states := []world.BlockState{
		{Name: "goe_frm:farmland", Properties: map[string]any{"goe_frm:direction": "southwest", "goe_frm:level_digit1": int32(4), "goe_frm:level_digit10": int32(2), "goe_frm:variant": int32(6)}},
		{Name: "goe_frm:plot_sign", Properties: map[string]any{"goe_frm:level_digit1": int32(9), "goe_frm:level_digit10": int32(3), "goe_frm:level_digit100": int32(1), "goe_frm:unlockable": uint8(1), "minecraft:cardinal_direction": "west", "minecraft:multi_block_part": int32(1)}},
	}
	for _, state := range states {
		reg.RegisterBlockState(state)
	}
	reg.Finalize()
	var hashes []uint32
	var ids []uint32
	for _, state := range states {
		rid, ok := reg.StateToRuntimeID(state.Name, state.Properties)
		if !ok {
			t.Fatal("missing state")
		}
		ids = append(ids, rid)
		hash, ok := reg.RuntimeIDToHash(rid)
		if !ok {
			t.Fatal("missing hash")
		}
		hashes = append(hashes, hash)
	}
	if ids[0] == ids[1] || hashes[0] != hashes[1] {
		t.Fatal("fixture must contain distinct states with the same wire hash")
	}
	if _, ok := reg.HashToRuntimeID(hashes[0]); ok {
		t.Fatal("ambiguous network hash must not silently select a state")
	}
	clone := reg.(*world.BasicBlockRegistry).Clone()
	if !clone.IsAmbiguousNetworkHash(hashes[0]) || clone.NetworkHashCollisionCount() != 1 {
		t.Fatal("cloning lost collision metadata")
	}
	buf := bytes.NewBuffer([]byte{9, 1, 0, 1})
	if err := protocol.WriteVarint32(buf, int32(hashes[0])); err != nil {
		t.Fatal(err)
	}
	wire := buf.Bytes()
	originalWire := append([]byte(nil), wire...)
	index := byte(0)
	sub, err := chunk.DecodeCapturedSubChunk(buf, reg, cube.Range{0, 15}, &index, chunk.NetworkEncoding, true)
	if err != nil {
		t.Fatal(err)
	}
	rid := sub.Layer(0).At(0, 0, 0)
	name, _, _ := reg.RuntimeIDToState(rid)
	if name != "minecraft:unknown" {
		t.Fatalf("ambiguous hash silently selected %s", name)
	}
	if !bytes.Equal(wire, originalWire) {
		t.Fatal("capture decoder modified the original wire payload")
	}
	if hash, ok := reg.RuntimeIDToHash(ids[0]); !ok || hash != hashes[0] {
		t.Fatal("capture fallback changed the original state's wire hash")
	}
}
