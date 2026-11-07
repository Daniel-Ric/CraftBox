package structure

import (
	"context"
	"errors"
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/util"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"math"
)

// PlaceInDatabase must only receive a database belonging to a NEW world copy.
// Original worlds are opened read-only by the application and never passed here.
func PlaceInDatabase(ctx context.Context, d *Document, db *mcdb.DB, reg world.BlockRegistry, anchor Vec3, mode string, dim world.Dimension, entities bool, progress func(int, int)) error {
	if mode != "replace" && mode != "air-only" && mode != "ignore-air" {
		return fmt.Errorf("Choose a placement mode")
	}
	offset := Vec3{}
	for i := range offset {
		offset[i] = anchor[i] - d.Anchor[i]
	}
	r := dim.Range()
	if offset[1] < int32(r[0]) || int64(offset[1])+int64(d.Size[1])-1 > int64(r[1]) {
		return fmt.Errorf("Placement exceeds dimension height %d to %d", r[0], r[1])
	}
	if offset[0] < -30000000 || offset[2] < -30000000 || int64(offset[0])+int64(d.Size[0]) > 30000000 || int64(offset[2])+int64(d.Size[2]) > 30000000 {
		return fmt.Errorf("Placement exceeds Bedrock world bounds")
	}
	ids := make([]uint32, len(d.Palette))
	for i, s := range d.Palette {
		rid, ok := reg.StateToRuntimeID(s.Name, s.States)
		if !ok {
			return fmt.Errorf("Cannot preserve %s", s.Name)
		}
		ids[i] = rid
	}
	minX, maxX := offset[0]>>4, (offset[0]+d.Size[0]-1)>>4
	minZ, maxZ := offset[2]>>4, (offset[2]+d.Size[2]-1)>>4
	total := int((maxX - minX + 1) * (maxZ - minZ + 1))
	done := 0
	// Actor record keys are global across chunks and dimensions.
	nextEntityID := int64(-1)
	if entities && len(d.Entities) > 0 {
		// Player IDs also share the world's unique actor ID space, even though
		// players themselves are never imported into a blueprint.
		for _, prefix := range []string{"~local_player", "player_"} {
			it := db.LDB().NewIterator(util.BytesPrefix([]byte(prefix)), nil)
			for it.Next() {
				if err := ctx.Err(); err != nil {
					it.Release()
					return err
				}
				var data map[string]any
				if err := nbt.UnmarshalEncoding(it.Value(), &data, nbt.LittleEndian); err != nil {
					it.Release()
					return fmt.Errorf("Cannot retain player actor identity: %w", err)
				}
				if id, ok := data["UniqueID"].(int64); ok && id <= nextEntityID {
					if id == math.MinInt64 {
						it.Release()
						return fmt.Errorf("Cannot allocate another entity ID")
					}
					nextEntityID = id - 1
				}
			}
			err := it.Error()
			it.Release()
			if err != nil {
				return err
			}
		}
		it := db.NewColumnIterator(nil)
		for it.Next() {
			if err := ctx.Err(); err != nil {
				it.Release()
				return err
			}
			for _, entity := range it.Column().Entities {
				if entity.ID <= nextEntityID {
					if entity.ID == math.MinInt64 {
						it.Release()
						return fmt.Errorf("Cannot allocate another entity ID")
					}
					nextEntityID = entity.ID - 1
				}
			}
		}
		err := it.Error()
		it.Release()
		if err != nil {
			return err
		}
	}
	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			cp := world.ChunkPos{x, z}
			col, err := db.LoadColumn(cp, dim)
			if errors.Is(err, leveldb.ErrNotFound) {
				col = &chunk.Column{Chunk: chunk.New(reg, r)}
			} else if err != nil {
				return err
			}
			changed := map[cube.Pos]bool{}
			b := d.Bounds()
			b.Min[0] = max(0, x*16-offset[0])
			b.Max[0] = min(d.Size[0]-1, x*16+15-offset[0])
			b.Min[2] = max(0, z*16-offset[2])
			b.Max[2] = min(d.Size[2]-1, z*16+15-offset[2])
			err = each(ctx, b, func(p Vec3) error {
				c := d.Cell(p)
				if !c.Known {
					return fmt.Errorf("Unrecorded cells must be resolved before placement")
				}
				if c.Layers[0] < 0 {
					return nil
				}
				if mode == "ignore-air" && Air(d.Palette[c.Layers[0]]) {
					return nil
				}
				q := cube.Pos{int(p[0] + offset[0]), int(p[1] + offset[1]), int(p[2] + offset[2])}
				if mode == "air-only" {
					rid := col.Chunk.Block(uint8(q[0]&15), int16(q[1]), uint8(q[2]&15), 0)
					name, _, ok := reg.RuntimeIDToState(rid)
					if !ok || !Air(State{Name: name}) {
						return nil
					}
				}
				for layer, idx := range c.Layers {
					rid := reg.AirRuntimeID()
					if idx >= 0 {
						rid = ids[idx]
					}
					col.Chunk.SetBlock(uint8(q[0]&15), int16(q[1]), uint8(q[2]&15), uint8(layer), rid)
				}
				changed[q] = true
				return nil
			})
			if err != nil {
				return err
			}
			kept := col.BlockEntities[:0]
			for _, be := range col.BlockEntities {
				if !changed[be.Pos] {
					kept = append(kept, be)
				}
			}
			col.BlockEntities = kept
			for p := range changed {
				local := Vec3{int32(p[0]) - offset[0], int32(p[1]) - offset[1], int32(p[2]) - offset[2]}
				if raw := d.BlockData[Key(local)]; raw != nil {
					if be, ok := raw["block_entity_data"].(map[string]any); ok {
						v := cloneMap(be)
						translateBlockData(v, Vec3{offset[0] - d.Origin[0], offset[1] - d.Origin[1], offset[2] - d.Origin[2]})
						v["x"], v["y"], v["z"] = int32(p[0]), int32(p[1]), int32(p[2])
						col.BlockEntities = append(col.BlockEntities, chunk.BlockEntity{Pos: p, Data: v})
					}
				}
			}
			if entities {
				for _, e := range d.Entities {
					if isPlayer(e) {
						continue
					}
					p, ok := entityPosition(e)
					if !ok {
						continue
					}
					for i := range p {
						p[i] += float64(offset[i])
					}
					if int32(math.Floor(p[0]/16)) != x || int32(math.Floor(p[2]/16)) != z {
						continue
					}
					v := cloneMap(e)
					setEntityPosition(v, p)
					if nextEntityID == math.MinInt64 {
						return fmt.Errorf("Entity ID space exhausted")
					}
					id := nextEntityID
					nextEntityID--
					v["UniqueID"] = id
					delete(v, "LinksTag")
					col.Entities = append(col.Entities, chunk.Entity{ID: id, Data: v})
				}
			}
			keptTicks := col.ScheduledBlocks[:0]
			for _, tick := range col.ScheduledBlocks {
				if !changed[tick.Pos] {
					keptTicks = append(keptTicks, tick)
				}
			}
			col.ScheduledBlocks = keptTicks
			if err = db.StoreColumn(cp, dim, col); err != nil {
				return err
			}
			done++
			if progress != nil {
				progress(done, total)
			}
		}
	}
	return ctx.Err()
}
