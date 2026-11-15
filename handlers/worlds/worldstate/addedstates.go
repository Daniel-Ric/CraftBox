package worldstate

import (
	"maps"

	"github.com/df-mc/dragonfly/server/world"
)

type addedStatesRegistry struct {
	world.BlockRegistry
	added map[string]map[string]any
}

func WithAddedStates(reg world.BlockRegistry, added map[string]map[string]any) world.BlockRegistry {
	if reg == nil || len(added) == 0 {
		return reg
	}
	return addedStatesRegistry{BlockRegistry: reg, added: maps.Clone(added)}
}

func (r addedStatesRegistry) RuntimeIDToState(rid uint32) (string, map[string]any, bool) {
	name, props, ok := r.BlockRegistry.RuntimeIDToState(rid)
	extra, has := r.added[name]
	if !ok || !has {
		return name, props, ok
	}
	out := make(map[string]any, len(props)+len(extra))
	maps.Copy(out, props)
	for k, v := range extra {
		if _, exists := out[k]; !exists {
			out[k] = v
		}
	}
	return name, out, true
}

func (r addedStatesRegistry) StateToRuntimeID(name string, props map[string]any) (uint32, bool) {
	if extra, has := r.added[name]; has {
		props = maps.Clone(props)
		for k := range extra {
			delete(props, k)
		}
	}
	return r.BlockRegistry.StateToRuntimeID(name, props)
}

func (r addedStatesRegistry) IsAmbiguousNetworkHash(hash uint32) bool {
	if registry, ok := r.BlockRegistry.(interface{ IsAmbiguousNetworkHash(uint32) bool }); ok {
		return registry.IsAmbiguousNetworkHash(hash)
	}
	return false
}
