package scene

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"os"
	"sort"
	"sync"
)

type Part struct {
	Mode      string    `json:"mode"`
	Positions []float32 `json:"positions"`
	Normals   []float32 `json:"normals"`
	UVs       []float32 `json:"uvs"`
	Indices   []uint32  `json:"indices"`
}
type ChunkMesh struct {
	Lights            []SceneLight `json:"lights,omitempty"`
	X                 int32        `json:"x"`
	Z                 int32        `json:"z"`
	Parts             []Part       `json:"parts"`
	MissingTextures   int          `json:"missingTextures"`
	ApproximateBlocks int          `json:"approximateBlocks"`
}
type Snapshot struct {
	Sequence      uint64      `json:"sequence"`
	Reset         bool        `json:"reset"`
	Chunks        []ChunkMesh `json:"chunks"`
	Atlas         string      `json:"atlas"`
	AtlasRevision uint64      `json:"atlasRevision"`
	Source        string      `json:"source"`
}

const maxMeshBytes = 64 << 20
const maxSnapshotBytes = 256 << 20

type Store struct {
	mu         sync.Mutex
	renderGate chan struct{}
	registry   world.BlockRegistry
	catalog    *Catalog

	chunks         map[world.ChunkPos]*chunk.Chunk
	potPlants      map[cube.Pos]string
	signs          map[cube.Pos]map[string]any
	visualActors   []Actor
	omitActors     bool
	actors         map[int64]Actor
	excludedActors map[int64]bool
	actorLinks     map[int64]map[int64]bool
	versions       map[world.ChunkPos]uint64
	positions      map[world.ChunkPos]struct{}
	cache          map[world.ChunkPos]ChunkMesh
	cacheBytes     int
	cacheScope     map[world.ChunkPos]string
	decoded        map[world.ChunkPos]decodedEntry
	models         map[uint32]*blockModel
	ctx            context.Context
	sequence       uint64
	directory      string
	db             *leveldb.DB
	storageError   error
	closed         bool
}

func NewStore(reg world.BlockRegistry, c *Catalog) *Store {
	s := &Store{renderGate: make(chan struct{}, 1), cacheScope: map[world.ChunkPos]string{}, decoded: map[world.ChunkPos]decodedEntry{}, models: map[uint32]*blockModel{}, registry: reg, catalog: c, versions: map[world.ChunkPos]uint64{}, positions: map[world.ChunkPos]struct{}{}, cache: map[world.ChunkPos]ChunkMesh{}, sequence: 1}
	s.directory, s.storageError = os.MkdirTemp("", "craftbox-scene-")
	if s.storageError == nil {
		s.db, s.storageError = leveldb.OpenFile(s.directory, &opt.Options{BlockCacheCapacity: 8 << 20, WriteBuffer: 4 << 20})
	}
	return s
}
func chunkKey(p world.ChunkPos) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b, uint32(p[0]))
	binary.LittleEndian.PutUint32(b[4:], uint32(p[1]))
	return b
}

func (s *Store) Put(pos world.ChunkPos, c *chunk.Chunk, entities ...map[cube.Pos]map[string]any) {
	data := chunk.Encode(c, chunk.NetworkEncoding)
	var encoded bytes.Buffer
	_ = binary.Write(&encoded, binary.LittleEndian, int32(c.Range()[0]))
	_ = binary.Write(&encoded, binary.LittleEndian, int32(c.Range()[1]))
	for _, sub := range data.SubChunks {
		encoded.Write(sub)
	}
	encoded.Write(data.Biomes)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.storageError != nil {
		return
	}
	var blockEntities map[cube.Pos]map[string]any
	if len(entities) > 0 {
		blockEntities = entities[0]
	}
	batch := new(leveldb.Batch)
	batch.Put(chunkKey(pos), encoded.Bytes())
	batch.Put(plantKey(pos), encodePotPlants(blockEntities))
	if e := s.db.Write(batch, nil); e != nil {
		s.storageError = e
		return
	}
	s.sequence++
	s.positions[pos] = struct{}{}
	for _, d := range [][2]int32{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		p := world.ChunkPos{pos[0] + d[0], pos[1] + d[1]}
		s.versions[p] = s.sequence
		if old, ok := s.cache[p]; ok {
			s.cacheBytes -= meshBytes(old)
			delete(s.cache, p)
		}
	}
}
func (s *Store) Invalidate() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	for p := range s.positions {
		s.versions[p] = s.sequence
	}
	clear(s.cache)
	s.cacheBytes = 0
	return s.sequence
}
func meshBytes(m ChunkMesh) int {
	n := len(m.Lights) * 40
	for _, p := range m.Parts {
		n += (len(p.Positions) + len(p.Normals) + len(p.UVs) + len(p.Indices)) * 4
	}
	return n
}

func (s *Store) Snapshot(after uint64, keep func(x, z int32) bool) (Snapshot, error) {
	return s.SnapshotContext(context.Background(), after, keep)
}

func (s *Store) SnapshotContext(ctx context.Context, after uint64, keep func(int32, int32) bool) (Snapshot, error) {
	if err := s.acquire(ctx); err != nil {
		return Snapshot{}, err
	}
	defer s.release()
	s.mu.Lock()
	out := Snapshot{Sequence: s.sequence, Reset: after == 0 || after > s.sequence, Chunks: []ChunkMesh{}, Source: s.catalog.Source}
	if s.closed || s.storageError != nil {
		e := s.storageError
		s.mu.Unlock()
		return out, fmt.Errorf("3D cache unavailable: %v", e)
	}
	var positions []world.ChunkPos
	versions := map[world.ChunkPos]uint64{}
	cached := map[world.ChunkPos]ChunkMesh{}
	for p := range s.positions {
		if keep == nil || keep(p[0], p[1]) {
			positions = append(positions, p)
			versions[p] = s.versions[p]
			if c, ok := s.cache[p]; ok && keep == nil && s.cacheScope[p] == "" {
				cached[p] = c
			}
		}
	}
	if len(positions) > 4096 {
		s.mu.Unlock()
		return out, fmt.Errorf("The 3D region contains %d chunks; select no more than 4096 chunks", len(positions))
	}
	snapshot, err := s.db.GetSnapshot()
	s.mu.Unlock()
	if err != nil {
		return out, err
	}
	defer snapshot.Release()
	sort.Slice(positions, func(i, j int) bool {
		if positions[i][0] == positions[j][0] {
			return positions[i][1] < positions[j][1]
		}
		return positions[i][0] < positions[j][0]
	})
	total := 0
	for _, p := range positions {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if !out.Reset && versions[p] <= after {
			continue
		}
		mesh, ok := cached[p]
		if !ok {
			view := &Store{ctx: ctx, models: s.models, registry: s.registry, catalog: s.catalog, chunks: map[world.ChunkPos]*chunk.Chunk{}}
			if err := view.loadPotPlants(snapshot, p); err != nil {
				return out, err
			}
			for _, d := range [][2]int32{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				neighbor := world.ChunkPos{p[0] + d[0], p[1] + d[1]}
				if keep != nil && !keep(neighbor[0], neighbor[1]) {
					continue
				}
				raw, e := snapshot.Get(chunkKey(neighbor), nil)
				if e == leveldb.ErrNotFound {
					continue
				}
				if e != nil {
					return out, e
				}
				if len(raw) < 8 {
					return out, fmt.Errorf("Corrupted 3D chunk")
				}
				r := cube.Range{int(int32(binary.LittleEndian.Uint32(raw))), int(int32(binary.LittleEndian.Uint32(raw[4:])))}
				c, e := chunk.NetworkDecode(s.registry, raw[8:], (r.Height()>>4)+1, r)
				if e != nil {
					return out, e
				}
				view.chunks[neighbor] = c
			}
			if view.chunks[p] == nil {
				view.chunks[p] = chunk.New(s.registry, cube.Range{-64, 319})
			}
			mesh = view.mesh(p, keep)
			if err := ctx.Err(); err != nil {
				return out, err
			}
			if keep == nil {
				s.mu.Lock()
				if s.versions[p] == versions[p] {
					size := meshBytes(mesh)
					if old, exists := s.cache[p]; exists {
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
						s.cacheScope[p] = ""
						s.cacheBytes += size
					}
				}
				s.mu.Unlock()
			}
		}
		total += meshBytes(mesh)
		if total > maxSnapshotBytes {
			return out, fmt.Errorf("3D model geometry exceeds 256 MiB; select a smaller region")
		}
		out.Chunks = append(out.Chunks, mesh)
	}
	out.AtlasRevision = s.catalog.revision
	if out.Reset || len(out.Chunks) > 0 {
		out.Atlas = s.catalog.atlasURL()
	}
	return out, nil
}
func (s *Store) Close() {
	if err := s.acquire(context.Background()); err != nil {
		return
	}
	defer s.release()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.db != nil {
		_ = s.db.Close()
	}
	if s.directory != "" {
		_ = os.RemoveAll(s.directory)
	}
	clear(s.cache)
	clear(s.decoded)
	clear(s.models)
	s.cacheBytes = 0
}

func (s *Store) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case s.renderGate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Store) release() { <-s.renderGate }
