package scene

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
)

// Distant editor terrain uses sampled height columns, never the export mesh.
// All original chunks remain in the scene database for full-quality rendering.
func (s *Store) loadViewportMesh(ctx context.Context, snapshot *leveldb.Snapshot, p world.ChunkPos, step int) (ChunkMesh, error) {
	raw, e := snapshot.Get(chunkKey(p), nil)
	if e == leveldb.ErrNotFound {
		return ChunkMesh{X: p[0], Z: p[1]}, nil
	}
	if e != nil {
		return ChunkMesh{}, e
	}
	if len(raw) < 8 {
		return ChunkMesh{}, fmt.Errorf("Corrupted 3D chunk")
	}
	r := cube.Range{int(int32(binary.LittleEndian.Uint32(raw))), int(int32(binary.LittleEndian.Uint32(raw[4:])))}
	c, e := chunk.NetworkDecode(s.registry, raw[8:], (r.Height()>>4)+1, r)
	if e != nil {
		return ChunkMesh{}, e
	}
	return s.viewportMesh(ctx, p, c, step)
}
func (s *Store) viewportMesh(ctx context.Context, p world.ChunkPos, c *chunk.Chunk, step int) (ChunkMesh, error) {
	if step != 4 {
		return ChunkMesh{}, fmt.Errorf("Unsupported viewport detail step")
	}
	out := ChunkMesh{X: p[0], Z: p[1], Parts: []Part{}}
	var heights [4][4]int
	var names [4][4]string
	var water [4][4]bool
	bottom := c.Range().Min()
	// Keep proxy columns within received terrain instead of the dimension floor.
	air, _ := s.registry.StateToRuntimeID("minecraft:air", nil)
findBottom:
	for i, sub := range c.Sub() {
		if sub.Empty() {
			continue
		}
		for y := bottom + i*16; y < bottom+(i+1)*16; y++ {
			if e := ctx.Err(); e != nil {
				return out, e
			}
			for z := uint8(0); z < 16; z++ {
				for x := uint8(0); x < 16; x++ {
					if c.Block(x, int16(y), z, 0) != air {
						bottom = y
						break findBottom
					}
				}
			}
		}
	}
	for z := 0; z < 4; z++ {
		for x := 0; x < 4; x++ {
			if e := ctx.Err(); e != nil {
				return out, e
			}
			heights[z][x] = bottom
			// Preserve the tallest received surface in each cell, including thin models.
			for dz := 0; dz < step; dz++ {
				for dx := 0; dx < step; dx++ {
					sx, sz := uint8(x*step+dx), uint8(z*step+dz)
					top := int(c.HighestBlock(sx, sz))
					for y := top; y >= heights[z][x] && y >= bottom; y-- {
						m := s.cachedModel(c.Block(sx, int16(y), sz, 0))
						if len(m.boxes) == 0 && len(m.quads) == 0 && !m.cross {
							continue
						}
						heights[z][x] = y + 1
						names[z][x] = m.name
						water[z][x] = m.fluid == "water"
						break
					}
				}
			}
		}
	}
	parts := map[string]*Part{"opaque": {Mode: "opaque"}, "water": {Mode: "water"}}
	for z := 0; z < 4; z++ {
		for x := 0; x < 4; x++ {
			h := heights[z][x]
			if names[z][x] == "" {
				continue
			}
			a, b, d, e := float32(x*step), float32((x+1)*step), float32(z*step), float32((z+1)*step)
			y := float32(h)
			offset := [3]float32{}
			mode := "opaque"
			if water[z][x] {
				mode = "water"
			}
			part := parts[mode]
			emitQuad(part, [4][3]float32{{a, y, e}, {b, y, e}, {b, y, d}, {a, y, d}}, [3]float32{0, 1, 0}, offset, s.catalog.texture(names[z][x], 2))
			edges := []struct {
				nx, nz int
				normal [3]float32
				q      [4][3]float32
			}{{x - 1, z, [3]float32{-1, 0, 0}, [4][3]float32{{a, y, d}, {a, y, e}, {a, 0, e}, {a, 0, d}}}, {x + 1, z, [3]float32{1, 0, 0}, [4][3]float32{{b, y, e}, {b, y, d}, {b, 0, d}, {b, 0, e}}}, {x, z - 1, [3]float32{0, 0, -1}, [4][3]float32{{b, y, d}, {a, y, d}, {a, 0, d}, {b, 0, d}}}, {x, z + 1, [3]float32{0, 0, 1}, [4][3]float32{{a, y, e}, {b, y, e}, {b, 0, e}, {a, 0, e}}}}
			for _, edge := range edges {
				low := bottom
				if edge.nx >= 0 && edge.nx < 4 && edge.nz >= 0 && edge.nz < 4 {
					low = heights[edge.nz][edge.nx]
				}
				if low >= h {
					continue
				}
				edge.q[2][1] = float32(low)
				edge.q[3][1] = float32(low)
				edge.q[0], edge.q[1] = edge.q[1], edge.q[0]
				edge.q[2], edge.q[3] = edge.q[3], edge.q[2]
				emitQuad(part, edge.q, edge.normal, offset, s.catalog.texture(names[z][x], 0))
			}
			out.ApproximateBlocks += step * step
		}
	}
	for _, mode := range []string{"opaque", "water"} {
		if part := parts[mode]; len(part.Indices) > 0 {
			out.Parts = append(out.Parts, *part)
		}
	}
	return out, nil
}
