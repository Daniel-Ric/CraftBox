package utils

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func CapturedItemNBT(reg world.BlockRegistry, stack protocol.ItemStack, names map[int32]string, hashed bool) (map[string]any, error) {
	if stack.Count == 0 || stack.NetworkID == 0 {
		return nil, nil
	}
	if stack.Count > 255 {
		return nil, fmt.Errorf("item count %d cannot be represented in Bedrock item NBT", stack.Count)
	}
	if stack.MetadataValue > 65535 {
		return nil, fmt.Errorf("item metadata %d exceeds the Bedrock item damage field", stack.MetadataValue)
	}
	name := names[stack.NetworkID]
	if names == nil {
		if item, ok := world.ItemByRuntimeID(stack.NetworkID, int16(stack.MetadataValue)); ok {
			name, _ = item.EncodeItem()
		}
	}
	if name == "" {
		return nil, fmt.Errorf("missing item identifier for captured network ID %d", stack.NetworkID)
	}
	out := map[string]any{"Name": name, "Count": uint8(stack.Count), "Damage": int16(stack.MetadataValue), "WasPickedUp": uint8(0)}
	if len(stack.NBTData) > 0 {
		raw, err := nbt.MarshalEncoding(stack.NBTData, nbt.LittleEndian)
		if err != nil {
			return nil, fmt.Errorf("item %s NBT: %w", name, err)
		}
		var tag map[string]any
		if err = nbt.UnmarshalEncoding(raw, &tag, nbt.LittleEndian); err != nil {
			return nil, err
		}
		out["tag"] = tag
	}
	if len(stack.CanBePlacedOn) > 0 {
		out["CanPlaceOn"] = append([]string(nil), stack.CanBePlacedOn...)
	}
	if len(stack.CanBreak) > 0 {
		out["CanDestroy"] = append([]string(nil), stack.CanBreak...)
	}
	if stack.BlockRuntimeID != 0 {
		rid := uint32(stack.BlockRuntimeID)
		if hashed {
			mapped, found := reg.HashToRuntimeID(rid)
			if !found {
				return nil, fmt.Errorf("item %s has unknown block hash %d", name, rid)
			}
			rid = mapped
		}
		blockName, props, ok := reg.RuntimeIDToState(rid)
		if !ok {
			return nil, fmt.Errorf("item %s has unknown block state %d", name, rid)
		}
		out["Block"] = map[string]any{"name": blockName, "states": props, "version": int32(chunk.CurrentBlockVersion)}
	}
	return out, nil
}
func CapturedInventoryNBT(reg world.BlockRegistry, stacks []protocol.ItemInstance, names map[int32]string, hashed bool) ([]map[string]any, error) {
	if len(stacks) > 256 {
		return nil, fmt.Errorf("inventory has %d slots; Bedrock item slots support at most 256", len(stacks))
	}
	out := []map[string]any{}
	for slot, entry := range stacks {
		data, err := CapturedItemNBT(reg, entry.Stack, names, hashed)
		if err != nil {
			return nil, fmt.Errorf("inventory slot %d: %w", slot, err)
		}
		if data != nil {
			data["Slot"] = uint8(slot)
			out = append(out, data)
		}
	}
	return out, nil
}
