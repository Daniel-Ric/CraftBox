package scene

import "github.com/df-mc/dragonfly/server/world/chunk"

var emptyBlockModel blockModel

func (s *Store) cachedModel(rid uint32) *blockModel {
	if s.models == nil {
		s.models = map[uint32]*blockModel{}
	}
	if m, ok := s.models[rid]; ok {
		return m
	}
	value := s.model(rid)
	s.models[rid] = &value
	return &value
}

func newPart(mode string) *Part {

	return &Part{Mode: mode, Positions: make([]float32, 0, 6144), Normals: make([]float32, 0, 6144), UVs: make([]float32, 0, 4096), Indices: make([]uint32, 0, 3072)}
}

func (s *Store) solidField(c *chunk.Chunk, get func(int, int, int) *blockModel) []byte {
	r := c.Range()
	data := make([]byte, (r.Height()+3)*324)
	for sy, sub := range c.Sub() {
		if s.ctx != nil && s.ctx.Err() != nil {
			return data
		}
		if sub.Empty() || len(sub.Layers()) == 0 {
			continue
		}
		storage := sub.Layers()[0]
		palette := storage.Palette()
		uniform := palette.Len() == 1
		if uniform && !s.cachedModel(palette.Value(0)).full {
			continue
		}
		for dy := 0; dy < 16; dy++ {
			y := r[0] + sy*16 + dy
			for z := 0; z < 16; z++ {
				offset := (y-r[0]+1)*324 + (z+1)*18 + 1
				for x := 0; x < 16; x++ {
					if uniform || s.cachedModel(storage.At(uint8(x), uint8(dy), uint8(z))).full {
						data[offset+x] = 1
					}
				}
			}
		}
	}
	for y := r[0]; y <= r[1]; y++ {
		base := (y - r[0] + 1) * 324
		for v := 0; v < 16; v++ {
			if get(-1, y, v).full {
				data[base+(v+1)*18] = 1
			}
			if get(16, y, v).full {
				data[base+(v+1)*18+17] = 1
			}
			if get(v, y, -1).full {
				data[base+v+1] = 1
			}
			if get(v, y, 16).full {
				data[base+17*18+v+1] = 1
			}
		}
	}
	return data
}
