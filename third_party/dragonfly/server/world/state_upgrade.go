package world

import (
	"github.com/df-mc/dragonfly/server/block/cube"
)

type loadedSource struct{ w *World }

func (s loadedSource) Block(pos cube.Pos) Block {
	b, _ := s.w.blockLoaded(pos)
	return b
}

type derivedBlock struct {
	pos cube.Pos
	b   StateDeriver
}

func (w *World) deriveStates(centre ChunkPos) {
	for x := int32(-1); x <= 1; x++ {
		for z := int32(-1); z <= 1; z++ {
			pos := ChunkPos{centre[0] + x, centre[1] + z}
			if c, ok := w.chunks[pos]; ok && c.LegacyStates() {
				w.deriveChunkStates(pos, c)
			}
		}
	}
}

func (w *World) deriveChunkStates(pos ChunkPos, c *Column) {
	for x := int32(-1); x <= 1; x++ {
		for z := int32(-1); z <= 1; z++ {
			if _, ok := w.chunks[ChunkPos{pos[0] + x, pos[1] + z}]; !ok {
				return
			}
		}
	}
	found := w.derivedBlocks(pos, c)
	c.MarkStatesUpgraded()
	if len(found) == 0 {
		return
	}

	src := loadedSource{w: w}
	for _, f := range found {
		derived := f.b.DeriveState(f.pos, src)
		if derived == Block(f.b) {
			continue
		}

		c.SetBlock(uint8(f.pos[0]), int16(f.pos[1]), uint8(f.pos[2]), 0, w.conf.Blocks.BlockRuntimeID(derived))
		c.modified = true

		for _, v := range c.viewers {
			v.ViewBlockUpdate(f.pos, derived, 0)
		}
	}
}

func (w *World) derivedBlocks(pos ChunkPos, c *Column) []derivedBlock {
	var found []derivedBlock
	baseX, baseZ := int(pos[0])<<4, int(pos[1])<<4
	for index, sub := range c.Sub() {
		if sub.Empty() {
			continue
		}
		storage := sub.Layer(0)
		palette := storage.Palette()

		var derivers map[uint16]StateDeriver
		for i := 0; i < palette.Len(); i++ {
			b, ok := w.conf.Blocks.BlockByRuntimeID(palette.Value(uint16(i)))
			if !ok {
				continue
			}
			if d, ok := b.(StateDeriver); ok {
				if derivers == nil {
					derivers = make(map[uint16]StateDeriver, 4)
				}
				derivers[uint16(i)] = d
			}
		}
		if derivers == nil {
			continue
		}

		baseY := int(c.SubY(int16(index)))
		for x := byte(0); x < 16; x++ {
			for y := byte(0); y < 16; y++ {
				for z := byte(0); z < 16; z++ {
					if d, ok := derivers[storage.PaletteIndex(x, y, z)]; ok {
						found = append(found, derivedBlock{
							pos: cube.Pos{baseX + int(x), baseY + int(y), baseZ + int(z)},
							b:   d,
						})
					}
				}
			}
		}
	}
	return found
}
