package scene

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
	"math"
	"sort"
	"time"
)

type PageRequest struct {
	PreviewStep   int       `json:"previewStep,omitempty"`
	Bounds        *Bounds   `json:"bounds,omitempty"`
	TerrainOnly   bool      `json:"terrainOnly,omitempty"`
	After         uint64    `json:"after"`
	Target        uint64    `json:"target"`
	Cursor        *[2]int32 `json:"cursor"`
	Focus         [2]int32  `json:"focus"`
	Limit         int       `json:"limit"`
	AtlasRevision uint64    `json:"atlasRevision"`
	Scope         uint64    `json:"scope"`
}
type PackedPart struct {
	BlockLight string `json:"blockLight,omitempty"`
	Mode       string `json:"mode"`
	Vertices   int    `json:"vertices"`
	Indices    int    `json:"indices"`
	Data       string `json:"data"`
}
type PackedChunk struct {
	Signals           map[string]int `json:"signals,omitempty"`
	Lights            []SceneLight   `json:"lights,omitempty"`
	X                 int32          `json:"x"`
	Z                 int32          `json:"z"`
	Parts             []PackedPart   `json:"parts"`
	MissingTextures   int            `json:"missingTextures"`
	ApproximateBlocks int            `json:"approximateBlocks"`
}
type Page struct {
	Sequence      uint64        `json:"sequence"`
	Scope         uint64        `json:"scope"`
	Reset         bool          `json:"reset"`
	Complete      bool          `json:"complete"`
	Next          *[2]int32     `json:"next"`
	Total         int           `json:"total"`
	Chunks        []PackedChunk `json:"chunks"`
	AtlasRevision uint64        `json:"atlasRevision"`
	Source        string        `json:"source"`
	Patches       []AtlasPatch  `json:"patches"`
	AtlasSize     int           `json:"atlasSize"`
}
type decodedEntry struct {
	chunk   *chunk.Chunk
	version uint64
}

func packChunk(c ChunkMesh) PackedChunk {
	out := PackedChunk{X: c.X, Z: c.Z, MissingTextures: c.MissingTextures, ApproximateBlocks: c.ApproximateBlocks, Parts: []PackedPart{}}
	out.Lights = c.Lights
	out.Signals = c.Signals
	for _, p := range c.Parts {
		vertices := len(p.Positions) / 3
		normalOffset := vertices * 12
		uvOffset := (normalOffset + vertices*3 + 3) &^ 3
		indexOffset := uvOffset + vertices*8
		data := make([]byte, indexOffset+len(p.Indices)*4)
		for i, v := range p.Positions {
			binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(v))
		}
		for i, v := range p.Normals {
			data[normalOffset+i] = byte(int8(math.Round(float64(v * 127))))
		}
		for i, v := range p.UVs {
			binary.LittleEndian.PutUint32(data[uvOffset+i*4:], math.Float32bits(v))
		}
		for i, v := range p.Indices {
			binary.LittleEndian.PutUint32(data[indexOffset+i*4:], v)
		}
		out.Parts = append(out.Parts, PackedPart{BlockLight: base64.StdEncoding.EncodeToString(p.BlockLight), Mode: p.Mode, Vertices: vertices, Indices: len(p.Indices), Data: base64.StdEncoding.EncodeToString(data)})
	}
	return out
}

func nearer(a, b world.ChunkPos, focus [2]int32) bool {

	distance := func(p world.ChunkPos) float64 {
		x, z := float64(p[0])-float64(focus[0]), float64(p[1])-float64(focus[1])
		return x*x + z*z
	}
	da, db := distance(a), distance(b)
	if da != db {
		return da < db
	}
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	return a[1] < b[1]
}

func (s *Store) Page(ctx context.Context, request PageRequest, keep func(int32, int32) bool) (Page, error) {
	return s.PageScoped(ctx, request, keep, "")
}
func (s *Store) PageScoped(ctx context.Context, request PageRequest, keep func(int32, int32) bool, scope string) (Page, error) {
	if request.PreviewStep != 0 && request.PreviewStep != 4 {
		return Page{}, fmt.Errorf("Invalid viewport detail step")
	}
	if request.Bounds != nil {
		if err := request.Bounds.Validate(); err != nil {
			return Page{}, err
		}
		bounds := *request.Bounds
		previous := keep
		keep = func(x, z int32) bool { return bounds.Contains(x, z) && (previous == nil || previous(x, z)) }
		scope += fmt.Sprintf("/bounds:%d,%d,%d,%d", bounds.MinX, bounds.MinZ, bounds.MaxX, bounds.MaxZ)
	}
	if request.TerrainOnly {
		scope += "/terrain-only"
	}
	if err := s.acquire(ctx); err != nil {
		return Page{}, err
	}
	defer s.release()
	start := time.Now()
	s.mu.Lock()
	out := Page{Sequence: request.Target, Scope: request.Scope, Chunks: []PackedChunk{}, Source: s.catalog.Source}
	if s.closed || s.storageError != nil {
		e := s.storageError
		s.mu.Unlock()
		return out, fmt.Errorf("3D cache unavailable: %v", e)
	}
	if request.Target == 0 || request.Target > s.sequence || request.After > s.sequence {
		out.Sequence = s.sequence
		if request.After > s.sequence || request.Target > s.sequence {
			request.After = 0
			request.Cursor = nil
		}
	}
	out.Reset = request.After == 0 && request.Cursor == nil
	versions := make(map[world.ChunkPos]uint64, len(s.versions))
	for p, v := range s.versions {
		versions[p] = v
	}
	positions := make([]world.ChunkPos, 0, len(s.positions))
	for p := range s.positions {
		if keep != nil && !keep(p[0], p[1]) {
			continue
		}
		out.Total++
		if request.After != 0 && versions[p] <= request.After {
			continue
		}
		if request.Cursor != nil && !nearer(world.ChunkPos(*request.Cursor), p, request.Focus) {
			continue
		}
		positions = append(positions, p)
	}
	snapshot, err := s.db.GetSnapshot()
	s.mu.Unlock()
	if err != nil {
		return out, err
	}
	defer snapshot.Release()
	sort.Slice(positions, func(i, j int) bool { return nearer(positions[i], positions[j], request.Focus) })
	limit := request.Limit
	if limit <= 0 {
		limit = 8
	}
	limit = min(limit, 16)
	processed := 0
	size := 0
	for _, p := range positions {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		var mesh ChunkMesh
		var e error
		// Legacy previewStep=4 requests retain their actual mesh and UVs. Editor
		// performance is controlled by screen-space sampling, never terrain shape.
		mesh, e = s.loadMesh(ctx, snapshot, p, versions, keep, scope, request.TerrainOnly)
		if e != nil {
			return out, e
		}
		out.Chunks = append(out.Chunks, packChunk(mesh))
		processed++
		size += meshBytes(mesh)
		next := [2]int32{p[0], p[1]}
		out.Next = &next
		if processed >= limit || size >= 2<<20 || time.Since(start) >= 35*time.Millisecond {
			break
		}
	}
	out.Complete = processed == len(positions)
	if out.Complete {
		out.Next = nil
	}
	out.AtlasRevision = s.catalog.revision
	out.AtlasSize = atlasSize
	out.Patches = s.catalog.atlasPatches(request.AtlasRevision)
	return out, nil
}

func (s *Store) loadMesh(ctx context.Context, snapshot *leveldb.Snapshot, p world.ChunkPos, versions map[world.ChunkPos]uint64, keep func(int32, int32) bool, scope string, terrainOnly bool) (ChunkMesh, error) {
	s.mu.Lock()
	cached, ok := s.cache[p]
	valid := s.versions[p] == versions[p] && s.cacheScope[p] == scope
	s.mu.Unlock()
	if ok && valid {
		return cached, nil
	}
	view := &Store{ctx: ctx, models: s.models, registry: s.registry, catalog: s.catalog, chunks: map[world.ChunkPos]*chunk.Chunk{}}
	view.omitActors = terrainOnly
	if err := view.loadPotPlants(snapshot, p); err != nil {
		return ChunkMesh{}, err
	}
	for _, d := range lightNeighbours {
		neighbor := world.ChunkPos{p[0] + d[0], p[1] + d[1]}
		if keep != nil && !keep(neighbor[0], neighbor[1]) {
			continue
		}
		if saved, ok := s.decoded[neighbor]; ok && saved.version == versions[neighbor] {
			view.chunks[neighbor] = saved.chunk
			continue
		}
		raw, e := snapshot.Get(chunkKey(neighbor), nil)
		if e == leveldb.ErrNotFound {
			continue
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
		if len(s.decoded) >= 48 {
			clear(s.decoded)
		}
		s.decoded[neighbor] = decodedEntry{chunk: c, version: versions[neighbor]}
		view.chunks[neighbor] = c
	}
	if view.chunks[p] == nil {
		view.chunks[p] = chunk.New(s.registry, cube.Range{-64, 319})
	}
	mesh := view.mesh(p, keep)
	if err := ctx.Err(); err != nil {
		return ChunkMesh{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.versions[p] == versions[p] {
		size := meshBytes(mesh)
		if old, ok := s.cache[p]; ok {
			s.cacheBytes -= meshBytes(old)
			delete(s.cache, p)
		}
		if s.cacheBytes+size > maxMeshBytes {
			clear(s.cache)
			clear(s.cacheScope)
			s.cacheBytes = 0
		}
		if size <= maxMeshBytes {
			s.cache[p] = mesh
			s.cacheScope[p] = scope
			s.cacheBytes += size
		}
	}
	return mesh, nil
}
