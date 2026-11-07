package merge

import (
	"maps"
	"math"
	"reflect"
	"strings"
	"sync"
	_ "unsafe"

	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

type BlockRegistry struct {
	world.BlockRegistry
	Rids map[uint32]Block
	mu   sync.RWMutex
}

type Block struct {
	name       string
	properties map[string]any
}

//go:linkname networkBlockHash github.com/df-mc/dragonfly/server/world.networkBlockHash
func networkBlockHash(name string, properties map[string]any, scratch []byte) (uint32, []byte)

func (b *BlockRegistry) RuntimeIDToState(runtimeID uint32) (name string, properties map[string]any, found bool) {
	name, properties, found = b.BlockRegistry.RuntimeIDToState(runtimeID)
	if found {
		return
	}
	b.mu.RLock()
	block, found := b.Rids[runtimeID]
	b.mu.RUnlock()
	if !found {
		return "", nil, false
	}
	return block.name, maps.Clone(block.properties), true
}

func (b *BlockRegistry) StateToRuntimeID(name string, properties map[string]any) (runtimeID uint32, found bool) {
	if name == "" || !strings.Contains(name, ":") {
		return 0, false
	}

	var defaults map[string]any
	if rid, ok := b.BlockRegistry.StateToRuntimeID(name, nil); ok {
		_, defaults, _ = b.BlockRegistry.RuntimeIDToState(rid)
	}
	state := maps.Clone(defaults)
	for key := range state {
		if _, supplied := properties[key]; !supplied && (key == "minecraft:corner" || strings.HasPrefix(key, "minecraft:connection_") || key == "wall_post_bit" || strings.HasPrefix(key, "wall_connection_type_")) {
			delete(state, key)
		}
	}
	if len(properties) > 0 && state == nil {
		state = make(map[string]any, len(properties))
	}
	for key, value := range properties {
		value, ok := normaliseStateValue(value, defaults[key])
		if !ok {
			return 0, false
		}
		state[key] = value
	}
	if rid, ok := b.BlockRegistry.StateToRuntimeID(name, state); ok {
		actualName, actualProps, _ := b.BlockRegistry.RuntimeIDToState(rid)
		if actualName == name && sameState(actualProps, state) {
			return rid, true
		}
	}
	runtimeID, _ = networkBlockHash(name, state, nil)

	if _, _, exists := b.BlockRegistry.RuntimeIDToState(runtimeID); exists {
		return 0, false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if old, exists := b.Rids[runtimeID]; exists {
		return runtimeID, old.name == name && sameState(old.properties, state)
	}
	if b.Rids == nil {
		b.Rids = map[uint32]Block{}
	}
	b.Rids[runtimeID] = Block{name, state}
	return runtimeID, true
}

func sameState(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for key, left := range a {
		right, ok := b[key]
		if !ok {
			return false
		}
		if reflect.DeepEqual(left, right) {
			continue
		}
		if v, ok := left.(bool); ok {
			if byteValue, ok := right.(uint8); ok && byteValue <= 1 && v == (byteValue != 0) {
				continue
			}
		}
		if v, ok := right.(bool); ok {
			if byteValue, ok := left.(uint8); ok && byteValue <= 1 && v == (byteValue != 0) {
				continue
			}
		}
		return false
	}
	return true
}

func normaliseStateValue(value, expected any) (any, bool) {
	if _, boolean := expected.(bool); boolean {
		if v, byteValue := value.(uint8); byteValue {
			if v > 1 {
				return nil, false
			}
			return v != 0, true
		}
	}
	if v, ok := value.(bool); ok {
		switch expected.(type) {
		case uint8:
			if v {
				return uint8(1), true
			}
			return uint8(0), true
		case int32:
			if v {
				return int32(1), true
			}
			return int32(0), true
		}
	}
	switch v := value.(type) {
	case string, bool, uint8, int32:
		return value, true
	case int:
		if int64(v) >= math.MinInt32 && int64(v) <= math.MaxInt32 {
			return int32(v), true
		}
	case int64:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) && v == math.Trunc(v) && v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	}
	return nil, false
}

func (b *BlockRegistry) BlockByRuntimeID(rid uint32) (world.Block, bool) {
	block, ok := b.BlockRegistry.BlockByRuntimeID(rid)
	if ok {
		return block, true
	}
	b.mu.RLock()
	block2, ok := b.Rids[rid]
	b.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return world.UnknownBlock{
		BlockState: world.BlockState{
			Name:       block2.name,
			Properties: maps.Clone(block2.properties),
		},
	}, ok
}
func (b *BlockRegistry) BlockRuntimeID(block world.Block) (rid uint32) {
	if block == nil {
		return b.BlockRegistry.BlockRuntimeID(nil)
	}
	name, properties := block.EncodeBlock()
	var ok bool
	rid, ok = b.StateToRuntimeID(name, properties)
	if !ok {
		panic("cannot represent block state without data loss: " + name)
	}
	return rid
}

func (b *BlockRegistry) BlockByRuntimeIDOrAir(rid uint32) world.Block {
	if block, ok := b.BlockByRuntimeID(rid); ok {
		return block
	}
	return b.BlockRegistry.Air()
}
func (b *BlockRegistry) BlockByName(name string, properties map[string]any) (world.Block, bool) {
	rid, ok := b.StateToRuntimeID(name, properties)
	if !ok {
		return nil, false
	}
	return b.BlockByRuntimeID(rid)
}
func (b *BlockRegistry) custom(rid uint32) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.Rids[rid]
	return ok
}

func (b *BlockRegistry) customBase(rid uint32) (native uint32, custom, found bool) {
	b.mu.RLock()
	state, custom := b.Rids[rid]
	b.mu.RUnlock()
	if !custom {
		return 0, false, false
	}
	base, ok := b.BlockRegistry.StateToRuntimeID(state.name, nil)
	if !ok {
		return 0, true, false
	}
	_, defaults, _ := b.BlockRegistry.RuntimeIDToState(base)
	props := maps.Clone(defaults)
	for key := range props {
		if value, ok := state.properties[key]; ok {
			props[key] = value
		}
	}
	native, ok = b.BlockRegistry.StateToRuntimeID(state.name, props)
	if !ok {
		return 0, true, false
	}
	_, actual, _ := b.BlockRegistry.RuntimeIDToState(native)
	return native, true, sameState(props, actual)
}
func (b *BlockRegistry) HashToRuntimeID(hash uint32) (uint32, bool) {
	if rid, ok := b.BlockRegistry.HashToRuntimeID(hash); ok {
		return rid, true
	}
	return hash, b.custom(hash)
}
func (b *BlockRegistry) RuntimeIDToHash(rid uint32) (uint32, bool) {
	if b.custom(rid) {
		return rid, true
	}
	return b.BlockRegistry.RuntimeIDToHash(rid)
}

func (b *BlockRegistry) BlockCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.BlockRegistry.BlockCount() + len(b.Rids)
}

func (b *BlockRegistry) RandomTickBlock(rid uint32) bool {
	if native, custom, found := b.customBase(rid); custom {
		if found {
			return b.BlockRegistry.RandomTickBlock(native)
		}
		return false
	}
	return b.BlockRegistry.RandomTickBlock(rid)
}

func (b *BlockRegistry) FilteringBlock(rid uint32) uint8 {
	if native, custom, found := b.customBase(rid); custom {
		if found {
			return b.BlockRegistry.FilteringBlock(native)
		}
		return 15
	}
	return b.BlockRegistry.FilteringBlock(rid)
}

func (b *BlockRegistry) LightBlock(rid uint32) uint8 {
	if native, custom, found := b.customBase(rid); custom {
		if found {
			return b.BlockRegistry.LightBlock(native)
		}
		return 0
	}
	return b.BlockRegistry.LightBlock(rid)
}

func (b *BlockRegistry) NBTBlock(rid uint32) bool {
	if native, custom, found := b.customBase(rid); custom {
		if found {
			return b.BlockRegistry.NBTBlock(native)
		}
		return false
	}
	return b.BlockRegistry.NBTBlock(rid)
}

func (b *BlockRegistry) LiquidDisplacingBlock(rid uint32) bool {
	if native, custom, found := b.customBase(rid); custom {
		if found {
			return b.BlockRegistry.LiquidDisplacingBlock(native)
		}
		return true
	}
	return b.BlockRegistry.LiquidDisplacingBlock(rid)
}

func (b *BlockRegistry) LiquidBlock(rid uint32) bool {
	if native, custom, found := b.customBase(rid); custom {
		if found {
			return b.BlockRegistry.LiquidBlock(native)
		}
		return false
	}
	return b.BlockRegistry.LiquidBlock(rid)
}
