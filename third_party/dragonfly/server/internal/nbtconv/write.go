package nbtconv

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func WriteBlock(b world.Block) map[string]any {
	name, properties := b.EncodeBlock()
	return map[string]any{
		"name":    name,
		"states":  properties,
		"version": chunk.CurrentBlockVersion,
	}
}
