package worlds

import "github.com/df-mc/dragonfly/server/world"

func (s *serverState) captureBlockRegistry() world.BlockRegistry {
	if s.diskBlocks != nil {
		return s.diskBlocks
	}
	return s.blocks
}
