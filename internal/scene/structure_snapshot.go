package scene

import (
	"context"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/util"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"math"
)

func floorChunk(v float32) int32 { return int32(math.Floor(float64(v) / 16)) }
func nbtPos(v map[string]any) cube.Pos {
	return cube.Pos{number(v["x"]), number(v["y"]), number(v["z"])}
}

// VisitColumns holds a database snapshot, not the capture mutex. Capturing may
// continue while an independent blueprint is being extracted. The render gate
// protects the snapshot's database lifetime from Close.
func (s *Store) VisitColumns(ctx context.Context, keep func(world.ChunkPos) bool, visit func(world.ChunkPos, *chunk.Column, world.BlockRegistry) error) error {
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.release()
	s.mu.Lock()
	if s.closed || s.storageError != nil {
		s.mu.Unlock()
		return fmt.Errorf("Capture scene is unavailable")
	}
	snap, err := s.db.GetSnapshot()
	positions := make([]world.ChunkPos, 0, len(s.captured))
	for p := range s.captured {
		if keep == nil || keep(p) {
			positions = append(positions, p)
		}
	}
	reg := s.registry
	s.mu.Unlock()
	if err != nil {
		return err
	}
	defer snap.Release()
	actors := map[world.ChunkPos][]chunk.Entity{}
	it := snap.NewIterator(util.BytesPrefix([]byte("entity-nbt:")), nil)
	for it.Next() {
		if err = ctx.Err(); err != nil {
			break
		}
		var data map[string]any
		if err = nbt.UnmarshalEncoding(it.Value(), &data, nbt.LittleEndian); err != nil {
			break
		}
		p := vectorValue(data["Pos"], [3]float32{})
		cp := world.ChunkPos{int32(floorChunk(p[0])), int32(floorChunk(p[2]))}
		if keep == nil || keep(cp) {
			actors[cp] = append(actors[cp], chunk.Entity{ID: rawEntityID(it.Key(), data), Data: data})
		}
	}
	if err == nil {
		err = it.Error()
	}
	it.Release()
	if err != nil {
		return err
	}
	for _, p := range positions {
		if err = ctx.Err(); err != nil {
			return err
		}
		c, e := compositionChunk(compositionSnapshot{db: snap, reg: reg}, p)
		if e != nil {
			return e
		}
		col := &chunk.Column{Chunk: c, Entities: actors[p]}
		raw, e := snap.Get(fullBlockNBTKey(p), nil)
		if e == nil {
			var data struct {
				Entries []map[string]any `nbt:"entries"`
			}
			if e = nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); e != nil {
				return e
			}
			for _, v := range data.Entries {
				col.BlockEntities = append(col.BlockEntities, chunk.BlockEntity{Pos: nbtPos(v), Data: v})
			}
		} else if e != leveldb.ErrNotFound {
			return e
		}
		if e = visit(p, col, reg); e != nil {
			return e
		}
	}
	return nil
}
