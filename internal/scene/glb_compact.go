package scene

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
)

func compactGLBPart(ctx context.Context, p Part) (Part, error) {
	if err := ctx.Err(); err != nil {
		return Part{}, err
	}
	vertices := len(p.Positions) / 3

	if vertices > 128*1024 {
		return p, nil
	}
	ids := make(map[[8]uint32]uint32, vertices)
	remap := make([]uint32, vertices)
	unique := make([]uint32, 0, vertices)
	for i := 0; i < vertices; i++ {
		if i%4096 == 0 {
			if err := ctx.Err(); err != nil {
				return Part{}, err
			}
		}
		key := [8]uint32{math.Float32bits(p.Positions[i*3]), math.Float32bits(p.Positions[i*3+1]), math.Float32bits(p.Positions[i*3+2]), math.Float32bits(p.Normals[i*3]), math.Float32bits(p.Normals[i*3+1]), math.Float32bits(p.Normals[i*3+2]), math.Float32bits(p.UVs[i*2]), math.Float32bits(p.UVs[i*2+1])}
		for _, bits := range key {
			if bits&0x7f800000 == 0x7f800000 {
				return Part{}, fmt.Errorf("Non-finite GLB geometry")
			}
		}
		id, ok := ids[key]
		if !ok {
			id = uint32(len(unique))
			ids[key] = id
			unique = append(unique, uint32(i))
		}
		remap[i] = id
	}
	if len(unique) == vertices {
		return p, nil
	}
	out := Part{Mode: p.Mode, Positions: make([]float32, len(unique)*3), Normals: make([]float32, len(unique)*3), UVs: make([]float32, len(unique)*2), Indices: make([]uint32, len(p.Indices))}
	for i, source := range unique {
		copy(out.Positions[i*3:i*3+3], p.Positions[int(source)*3:int(source)*3+3])
		copy(out.Normals[i*3:i*3+3], p.Normals[int(source)*3:int(source)*3+3])
		copy(out.UVs[i*2:i*2+2], p.UVs[int(source)*2:int(source)*2+2])
	}
	for i, index := range p.Indices {
		if i%16384 == 0 {
			if err := ctx.Err(); err != nil {
				return Part{}, err
			}
		}
		out.Indices[i] = remap[index]
	}
	return out, nil
}

func (g *GLBWriter) indices(values []uint32) (int, error) {
	maxIndex := uint32(0)
	for _, index := range values {
		maxIndex = max(maxIndex, index)
	}
	width, component := 4, 5125

	if maxIndex < math.MaxUint16 {
		width, component = 2, 5123
	}
	if err := g.align(); err != nil {
		return 0, err
	}
	start := g.size
	var buf [64 << 10]byte
	for offset := 0; offset < len(values); {
		n := min(len(values)-offset, len(buf)/width)
		for i, v := range values[offset : offset+n] {
			if width == 2 {
				binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
			} else {
				binary.LittleEndian.PutUint32(buf[i*4:], v)
			}
		}
		if err := g.write(buf[:n*width]); err != nil {
			return 0, err
		}
		offset += n
	}
	g.accessors = append(g.accessors, map[string]any{"bufferView": g.view(start, 34963), "componentType": component, "count": len(values), "type": "SCALAR"})
	return len(g.accessors) - 1, nil
}
