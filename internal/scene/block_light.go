package scene

import (
	"github.com/df-mc/dragonfly/server/world"
	"math"
	"strings"
)

var lightNeighbours = [][2]int32{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

type blockLightVolume struct {
	minY, height int
	levels       []uint8
}

func (f blockLightVolume) index(x, y, z int) int {
	if x < -16 || x >= 32 || z < -16 || z >= 32 || y < f.minY || y >= f.minY+f.height {
		return -1
	}
	return ((x+16)*48+z+16)*f.height + y - f.minY
}
func (f blockLightVolume) at(x, y, z int) uint8 {
	i := f.index(x, y, z)
	if i < 0 || len(f.levels) == 0 {
		return 0
	}
	return f.levels[i]
}

type blockOptics struct{ emission, filter uint8 }

func (s *Store) blockOptics(rid uint32) blockOptics {
	o := blockOptics{filter: 15}
	name, states, found := s.registry.RuntimeIDToState(rid)
	if !found || name == "" {
		return o
	}
	o.emission = s.emissionLevel(rid, name)
	o.filter = s.registry.FilteringBlock(rid)
	if name == "minecraft:light_block" || strings.HasPrefix(name, "minecraft:light_block_") {
		o.filter = 0
	}
	if def, exists := s.catalog.definitions[strings.TrimPrefix(name, "minecraft:")]; exists {
		value, has := def.Components["minecraft:light_dampening"]
		for _, p := range def.Permutations {
			if match, supported := blockCondition(p.Condition, states); match && supported {
				if v, yes := p.Components["minecraft:light_dampening"]; yes {
					value, has = v, true
				}
			}
		}
		if has {
			o.filter = uint8(max(0, min(15, number(value))))
		}
	}
	return o
}

func (s *Store) blockLightField(pos world.ChunkPos, keep func(int32, int32) bool) blockLightVolume {
	c := s.chunks[pos]
	f := blockLightVolume{minY: c.Range()[0] - 1, height: c.Range().Height() + 3}
	optics := map[uint32]blockOptics{}
	low, high := int(^uint(0)>>1), -int(^uint(0)>>1)

	for _, d := range lightNeighbours {
		cp := world.ChunkPos{pos[0] + d[0], pos[1] + d[1]}
		if keep != nil && !keep(cp[0], cp[1]) {
			continue
		}
		other := s.chunks[cp]
		if other == nil {
			continue
		}
		for sy, sub := range other.Sub() {
			for _, layer := range sub.Layers() {
				p := layer.Palette()
				for i := 0; i < p.Len(); i++ {
					rid := p.Value(uint16(i))
					o, ok := optics[rid]
					if !ok {
						o = s.blockOptics(rid)
						optics[rid] = o
					}
					if o.emission > 0 {
						y := other.Range()[0] + sy*16
						low = min(low, y-14)
						high = max(high, y+15+14)
					}
				}
			}
		}
	}
	if high < low {
		return f
	}
	minY, maxY := max(f.minY, low), min(f.minY+f.height-1, high)
	if minY > maxY {
		return f
	}
	f.minY, f.height = minY, maxY-minY+1
	f.levels = make([]uint8, 48*48*f.height)
	filters := make([]uint8, len(f.levels))
	for i := range filters {
		filters[i] = 15
	}
	var buckets [16][]int
	for _, d := range lightNeighbours {
		cp := world.ChunkPos{pos[0] + d[0], pos[1] + d[1]}
		if keep != nil && !keep(cp[0], cp[1]) {
			continue
		}
		other := s.chunks[cp]
		if other == nil {
			continue
		}
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				rx, rz := int(d[0])*16+x, int(d[1])*16+z
				for y := max(f.minY, other.Range()[0]); y <= min(f.minY+f.height-1, other.Range()[1]); y++ {
					if s.ctx != nil && s.ctx.Err() != nil {
						return f
					}
					var filter, emission uint8
					for layer := uint8(0); layer < 2; layer++ {
						rid := other.Block(uint8(x), int16(y), uint8(z), layer)
						o, ok := optics[rid]
						if !ok {
							o = s.blockOptics(rid)
							optics[rid] = o
						}
						filter = max(filter, o.filter)
						emission = max(emission, o.emission)
					}
					i := f.index(rx, y, rz)
					filters[i] = filter
					if emission > 0 {
						f.levels[i] = emission
						buckets[emission] = append(buckets[emission], i)
					}
				}
			}
		}
	}
	for level := 15; level > 1; level-- {
		for _, i := range buckets[level] {
			if int(f.levels[i]) != level {
				continue
			}
			if s.ctx != nil && s.ctx.Err() != nil {
				return f
			}
			y := i % f.height
			xz := i / f.height
			x, z := xz/48, xz%48
			for _, d := range [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
				nx, ny, nz := x+d[0], y+d[1], z+d[2]
				if nx < 0 || nx >= 48 || nz < 0 || nz >= 48 || ny < 0 || ny >= f.height {
					continue
				}
				n := (nx*48+nz)*f.height + ny
				next := level - 1 - int(filters[n])
				if next > int(f.levels[n]) {
					f.levels[n] = uint8(next)
					buckets[next] = append(buckets[next], n)
				}
			}
		}
	}
	return f
}

func (f blockLightVolume) illuminate(p *Part) {
	p.BlockLight = make([]uint8, len(p.Positions)/3)
	for t := 0; t < len(p.Indices); t += 3 {
		var center, normal [3]float64
		for k := 0; k < 3; k++ {
			v := int(p.Indices[t+k])
			for a := 0; a < 3; a++ {
				center[a] += float64(p.Positions[v*3+a]) / 3
				normal[a] += float64(p.Normals[v*3+a]) / 3
			}
		}
		sample := func(sign float64) uint8 {
			return f.at(int(math.Floor(center[0]+normal[0]*sign*.01)), int(math.Floor(center[1]+normal[1]*sign*.01)), int(math.Floor(center[2]+normal[2]*sign*.01)))
		}
		level := max(sample(1), sample(-1))
		for k := 0; k < 3; k++ {
			v := p.Indices[t+k]
			p.BlockLight[v] = max(p.BlockLight[v], level)
		}
	}
}
