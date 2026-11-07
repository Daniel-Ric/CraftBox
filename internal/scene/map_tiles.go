package scene

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

type MapRequest struct {
	OmitCoverage bool      `json:"omitCoverage,omitempty"`
	After        uint64    `json:"after"`
	Target       uint64    `json:"target"`
	Cursor       *[2]int32 `json:"cursor"`
	Pixels       int       `json:"pixels"`
	Y            *int      `json:"y"`
	Limit        int       `json:"limit"`
	Bounds       *[4]int32 `json:"bounds"`
}
type MapTile struct {
	X          int32             `json:"x"`
	Z          int32             `json:"z"`
	Image      string            `json:"image"`
	Heights    []int16           `json:"heights"`
	Biomes     []uint32          `json:"biomes"`
	BiomeNames map[uint32]string `json:"biomeNames,omitempty"`
	Blocks     []string          `json:"blocks"`
	Version    uint64            `json:"version"`
}
type MapPage struct {
	Sequence uint64     `json:"sequence"`
	Complete bool       `json:"complete"`
	Next     *[2]int32  `json:"next"`
	Total    int        `json:"total"`
	Coverage [][2]int32 `json:"coverage"`
	Tiles    []MapTile  `json:"tiles"`
}

func (s *Store) MapPage(ctx context.Context, r MapRequest, keep func(int32, int32) bool) (MapPage, error) {
	out := MapPage{Tiles: []MapTile{}}
	if r.Cursor == nil && !r.OmitCoverage {
		out.Coverage = make([][2]int32, 0)
	}
	if err := s.acquire(ctx); err != nil {
		return out, err
	}
	defer s.release()
	if r.Y != nil && (*r.Y < -32768 || *r.Y > 32767) {
		return out, fmt.Errorf("Y is outside the supported range")
	}
	pixels := r.Pixels
	if pixels != 4 && pixels != 16 {
		pixels = 1
	}
	limit := r.Limit
	if limit < 1 {
		limit = 16
	}
	limit = min(limit, 32)
	s.mu.Lock()
	if s.closed || s.storageError != nil {
		s.mu.Unlock()
		return out, fmt.Errorf("Map data unavailable")
	}
	out.Sequence = r.Target
	if out.Sequence == 0 || out.Sequence > s.sequence {
		out.Sequence = s.sequence
	}
	if r.After > out.Sequence {
		r.After = 0
	}
	versions := map[world.ChunkPos]uint64{}
	positions := []world.ChunkPos{}
	if len(s.mapOrder) != len(s.captured) {
		s.mapOrder = make([]world.ChunkPos, 0, len(s.captured))
		for p := range s.captured {
			s.mapOrder = append(s.mapOrder, p)
		}
		sort.Slice(s.mapOrder, func(i, j int) bool {
			a, b := s.mapOrder[i], s.mapOrder[j]
			return a[0] < b[0] || a[0] == b[0] && a[1] < b[1]
		})
	}
	out.Total = len(s.mapOrder)
	if keep != nil {
		out.Total = 0
	}
	if keep != nil || r.Cursor == nil && !r.OmitCoverage {
		for _, p := range s.mapOrder {
			if keep != nil && !keep(p[0], p[1]) {
				continue
			}
			if keep != nil {
				out.Total++
			}
			if r.Cursor == nil && !r.OmitCoverage {
				out.Coverage = append(out.Coverage, [2]int32(p))
			}
		}
	}
	startIndex := 0
	if r.Cursor != nil {
		startIndex = sort.Search(len(s.mapOrder), func(i int) bool {
			p := s.mapOrder[i]
			return p[0] > r.Cursor[0] || p[0] == r.Cursor[0] && p[1] > r.Cursor[1]
		})
	}
	if r.Bounds != nil {
		startIndex = max(startIndex, sort.Search(len(s.mapOrder), func(i int) bool { return s.mapOrder[i][0] >= r.Bounds[0] }))
	}
	for _, p := range s.mapOrder[startIndex:] {
		if r.Bounds != nil && p[0] > r.Bounds[2] {
			break
		}
		if keep != nil && !keep(p[0], p[1]) {
			continue
		}
		if r.Bounds != nil && (p[0] < r.Bounds[0] || p[1] < r.Bounds[1] || p[0] > r.Bounds[2] || p[1] > r.Bounds[3]) {
			continue
		}
		if r.After != 0 && s.versions[p] <= r.After {
			continue
		}
		if r.Cursor != nil && (p[0] < r.Cursor[0] || p[0] == r.Cursor[0] && p[1] <= r.Cursor[1]) {
			continue
		}
		versions[p] = s.versions[p]
		positions = append(positions, p)
		if len(positions) > limit {
			break
		}
	}
	snap, err := s.db.GetSnapshot()
	s.mu.Unlock()
	if err != nil {
		return out, err
	}
	defer snap.Release()
	count := min(len(positions), limit)
	processed := 0
	start := time.Now()
	for _, p := range positions[:count] {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		key := mapTileKey{Pos: p, Version: versions[p], Pixels: pixels}
		if r.Y != nil {
			key.Slice = true
			key.Y = *r.Y
		}
		s.mu.Lock()
		if s.mapTiles == nil {
			s.mapTiles = &mapTileCache{}
		}
		t, hit := s.mapTiles.get(key)
		s.mu.Unlock()
		if !hit {
			raw, err := snap.Get(chunkKey(p), nil)
			if err != nil {
				return out, err
			}
			if len(raw) < 8 {
				return out, fmt.Errorf("Invalid map chunk")
			}
			bounds := cube.Range{int(int32(binary.LittleEndian.Uint32(raw))), int(int32(binary.LittleEndian.Uint32(raw[4:])))}
			c, err := chunk.NetworkDecode(s.registry, raw[8:], (bounds.Height()>>4)+1, bounds)
			if err != nil {
				return out, err
			}
			t, err = s.mapTile(ctx, p, c, pixels, r.Y)
			if err != nil {
				return out, err
			}
			t.Version = versions[p]
			s.mu.Lock()
			if !s.closed && s.versions[p] == versions[p] {
				s.mapTiles.put(key, t)
			}
			s.mu.Unlock()
		}
		out.Tiles = append(out.Tiles, t)
		next := [2]int32(p)
		out.Next = &next
		processed++
		if time.Since(start) > 40*time.Millisecond {
			break
		}
	}
	out.Complete = processed == len(positions)
	if out.Complete {
		out.Next = nil
	}
	return out, nil
}

func (s *Store) mapTile(ctx context.Context, p world.ChunkPos, c *chunk.Chunk, pixels int, slice *int) (MapTile, error) {
	t := MapTile{X: p[0], Z: p[1], Heights: make([]int16, 256), Biomes: make([]uint32, 256), Blocks: make([]string, 256)}
	t.BiomeNames = map[uint32]string{}
	img := image.NewNRGBA(image.Rect(0, 0, 16*pixels, 16*pixels))
	s.catalog.ensureAtlas()
	for z := 0; z < 16; z++ {
		for x := 0; x < 16; x++ {
			if err := ctx.Err(); err != nil {
				return t, err
			}
			i := z*16 + x
			t.Heights[i] = -32768
			top := int(c.HighestBlock(uint8(x), uint8(z)))
			if slice != nil {
				top = min(top, *slice)
			}
			top = min(top, c.Range().Max())
			bottom := c.Range().Min()
			type entry struct {
				y     int
				m     *blockModel
				depth int
			}
			layers := []entry{}
			for y := top; y >= bottom; y-- {
				m := s.cachedModel(c.Block(uint8(x), int16(y), uint8(z), 0))
				if len(m.boxes) == 0 && len(m.quads) == 0 && !m.cross {
					continue
				}
				if t.Heights[i] == -32768 {
					t.Heights[i] = int16(y)
					t.Biomes[i] = c.Biome(uint8(x), int16(y), uint8(z))
					if biome, ok := world.BiomeByID(int(t.Biomes[i])); ok {
						t.BiomeNames[t.Biomes[i]] = biome.String()
					}
					t.Blocks[i] = m.name
				}
				layers = append(layers, entry{y: y, m: m, depth: 1})
				if m.full && m.mode == "opaque" {
					break
				}

				if m.fluid == "water" {
					for y > bottom {
						next := s.cachedModel(c.Block(uint8(x), int16(y-1), uint8(z), 0))
						if next.fluid != "water" {
							break
						}
						y--
						layers[len(layers)-1].depth++
					}
				}
			}
			for n := len(layers) - 1; n >= 0; n-- {
				e := layers[n]
				m := e.m
				for dz := 0; dz < pixels; dz++ {
					for dx := 0; dx < pixels; dx++ {
						u, v := (float32(dx)+.5)/float32(pixels), (float32(dz)+.5)/float32(pixels)
						var col color.NRGBA
						hit := false
						highest := float32(-1e9)
						if len(m.quads) > 0 {
							col = mapQuadPixel(s.catalog, m.quads, u, v)
							hit = col.A > 0

						} else {
							for _, b := range m.boxes {
								if u >= b.min[0] && u <= b.max[0] && v >= b.min[2] && v <= b.max[2] && b.max[1] >= highest {
									highest = b.max[1]
									hit = true
								}
							}
							if m.cross {
								hit = true
							}
							if hit {
								tex := s.catalog.texture(m.name, 2)
								col = s.catalog.atlas.NRGBAAt(tex.x+min(tex.w-1, int(u*float32(tex.w))), tex.y+min(tex.h-1, int(v*float32(tex.h))))
							}
						}
						if !hit || col.A == 0 {
							continue
						}
						if m.fluid == "water" {
							col.A = uint8(min(230, 110+e.depth*9) * int(col.A) / 255)
						} else if len(m.boxes) > 1 && highest > 0 {
							shade := .7 + .3*min(float32(1), highest)
							col.R = uint8(float32(col.R) * shade)
							col.G = uint8(float32(col.G) * shade)
							col.B = uint8(float32(col.B) * shade)
						}
						px, pz := x*pixels+dx, z*pixels+dz
						old := img.NRGBAAt(px, pz)
						img.SetNRGBA(px, pz, mapOver(old, col))
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return t, err
	}
	t.Image = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	return t, nil
}

func mapOver(bg, fg color.NRGBA) color.NRGBA {
	a, b := float64(fg.A)/255, float64(bg.A)/255
	out := a + b*(1-a)
	if out == 0 {
		return color.NRGBA{}
	}
	mix := func(x, y uint8) uint8 { return uint8(math.Round((float64(x)*a + float64(y)*b*(1-a)) / out)) }
	return color.NRGBA{mix(fg.R, bg.R), mix(fg.G, bg.G), mix(fg.B, bg.B), uint8(math.Round(out * 255))}
}

func mapQuadPixel(c *Catalog, quads []modelQuad, u, v float32) color.NRGBA {
	type sample struct {
		height float32
		color  color.NRGBA
	}
	samples := []sample{}
	for _, q := range quads {
		if q.normal[1] <= .001 {
			continue
		}
		for _, tri := range [][3]int{{0, 1, 2}, {0, 2, 3}} {
			a, b, d := q.points[tri[0]], q.points[tri[1]], q.points[tri[2]]
			den := (b[2]-d[2])*(a[0]-d[0]) + (d[0]-b[0])*(a[2]-d[2])
			if math.Abs(float64(den)) < 1e-7 {
				continue
			}
			wa := ((b[2]-d[2])*(u-d[0]) + (d[0]-b[0])*(v-d[2])) / den
			wb := ((d[2]-a[2])*(u-d[0]) + (a[0]-d[0])*(v-d[2])) / den
			wc := 1 - wa - wb
			if wa < -1e-6 || wb < -1e-6 || wc < -1e-6 {
				continue
			}
			tu := q.uv[tri[0]][0]*wa + q.uv[tri[1]][0]*wb + q.uv[tri[2]][0]*wc
			tv := q.uv[tri[0]][1]*wa + q.uv[tri[1]][1]*wb + q.uv[tri[2]][1]*wc
			col := c.atlas.NRGBAAt(max(0, min(atlasSize-1, int(tu*atlasSize))), max(0, min(atlasSize-1, int(tv*atlasSize))))
			if col.A > 0 {
				samples = append(samples, sample{a[1]*wa + b[1]*wb + d[1]*wc, col})
			}
			break
		}
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].height < samples[j].height })
	var col color.NRGBA
	for _, s := range samples {
		col = mapOver(col, s.color)
	}
	return col
}
