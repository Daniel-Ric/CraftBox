package structure

import (
	"context"
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"math"
)

func Registry() *merge.BlockRegistry {
	world.DefaultBlockRegistry.Finalize()
	return &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]merge.Block{}}
}

type Importer struct {
	Doc    *Document
	bounds Bounds
	states map[uint32]int32
}

func NewImporter(name string, bounds Bounds) (*Importer, error) {
	bounds = bounds.Normalized()
	size := Vec3{}
	for i := range size {
		n := int64(bounds.Max[i]) - int64(bounds.Min[i]) + 1
		if n > 4096 {
			return nil, fmt.Errorf("Select a region no wider than 4096 blocks")
		}
		size[i] = int32(n)
	}
	d, err := New(name, size)
	if err != nil {
		return nil, err
	}
	d.Origin = bounds.Min
	d.DefaultKnown = false
	d.Default = Cell{Layers: [2]int32{-1, -1}}
	return &Importer{d, bounds, map[uint32]int32{}}, nil
}
func (i *Importer) Column(ctx context.Context, p world.ChunkPos, col *chunk.Column, reg world.BlockRegistry) error {
	b := i.bounds
	b.Min[0] = max(b.Min[0], p[0]*16)
	b.Max[0] = min(b.Max[0], p[0]*16+15)
	b.Min[2] = max(b.Min[2], p[1]*16)
	b.Max[2] = min(b.Max[2], p[1]*16+15)
	b.Min[1] = max(b.Min[1], int32(col.Chunk.Range()[0]))
	b.Max[1] = min(b.Max[1], int32(col.Chunk.Range()[1]))
	if b.Min[0] > b.Max[0] || b.Min[1] > b.Max[1] || b.Min[2] > b.Max[2] {
		return nil
	}
	err := each(ctx, b, func(pos Vec3) error {
		c := Cell{Layers: [2]int32{-1, -1}, Known: true}
		for layer := 0; layer < 2; layer++ {
			rid := col.Chunk.Block(uint8(pos[0]&15), int16(pos[1]), uint8(pos[2]&15), uint8(layer))
			idx, ok := i.states[rid]
			if !ok {
				n, s, found := reg.RuntimeIDToState(rid)
				if !found {
					return fmt.Errorf("Unknown block runtime ID %d", rid)
				}
				var err error
				idx, err = i.Doc.AddState(State{Name: n, States: s, Version: chunk.CurrentBlockVersion})
				if err != nil {
					return err
				}
				i.states[rid] = idx
			}
			if layer == 0 || !Air(i.Doc.Palette[idx]) {
				c.Layers[layer] = idx
			}
		}
		local := Vec3{pos[0] - i.bounds.Min[0], pos[1] - i.bounds.Min[1], pos[2] - i.bounds.Min[2]}
		i.Doc.Set(local, c)
		return nil
	})
	if err != nil {
		return err
	}
	for _, be := range col.BlockEntities {
		p := Vec3{int32(be.Pos[0]), int32(be.Pos[1]), int32(be.Pos[2])}
		if i.bounds.Contains(p) {
			q := Vec3{p[0] - i.bounds.Min[0], p[1] - i.bounds.Min[1], p[2] - i.bounds.Min[2]}
			i.Doc.BlockData[Key(q)] = map[string]any{"block_entity_data": cloneMap(be.Data)}
		}
	}
	for _, e := range col.Entities {
		if isPlayer(e.Data) {
			continue
		}
		p, ok := entityPosition(e.Data)
		if !ok || !i.bounds.Contains(Vec3{int32(math.Floor(p[0])), int32(math.Floor(p[1])), int32(math.Floor(p[2]))}) {
			continue
		}
		v := cloneMap(e.Data)
		for axis := range p {
			p[axis] -= float64(i.bounds.Min[axis])
		}
		setEntityPosition(v, p)
		i.Doc.Entities = append(i.Doc.Entities, v)
	}
	return nil
}
func FromScene(ctx context.Context, s *scene.Store, name string, b Bounds) (*Document, error) {
	i, err := NewImporter(name, b)
	if err != nil {
		return nil, err
	}
	err = s.VisitColumns(ctx, func(p world.ChunkPos) bool {
		return p[0] >= b.Min[0]>>4 && p[0] <= b.Max[0]>>4 && p[1] >= b.Min[2]>>4 && p[1] <= b.Max[2]>>4
	}, func(p world.ChunkPos, c *chunk.Column, r world.BlockRegistry) error { return i.Column(ctx, p, c, r) })
	return i.Doc, err
}
func FromWorld(ctx context.Context, root, name string, b Bounds, dimID int32, progress func(int)) (*Document, error) {
	i, err := NewImporter(name, b)
	if err != nil {
		return nil, err
	}
	reg := Registry()
	db, err := (mcdb.Config{Blocks: reg, LDBOptions: &opt.Options{ReadOnly: true}}).Open(root)
	if err != nil {
		return nil, fmt.Errorf("Close Minecraft or the active capture before opening this world: %w", err)
	}
	defer db.Close()
	dim, ok := world.DimensionByID(int(dimID))
	if !ok {
		return nil, fmt.Errorf("Unsupported dimension")
	}
	count := 0
	for x := i.bounds.Min[0] >> 4; x <= i.bounds.Max[0]>>4; x++ {
		for z := i.bounds.Min[2] >> 4; z <= i.bounds.Max[2]>>4; z++ {
			if err = ctx.Err(); err != nil {
				return nil, err
			}
			p := world.ChunkPos{x, z}
			col, e := db.LoadColumn(p, dim)
			if errors.Is(e, leveldb.ErrNotFound) {
				continue
			}
			if e != nil {
				return nil, e
			}
			if e = i.Column(ctx, p, col, reg); e != nil {
				return nil, e
			}
			count++
			if progress != nil {
				progress(count)
			}
		}
	}
	return i.Doc, nil
}

// Columns emits an independent native column at a time, bounding render/export memory.
func (d *Document) Columns(ctx context.Context, reg world.BlockRegistry, offset Vec3, visit func(world.ChunkPos, *chunk.Column) error) error {
	rids := make([]uint32, len(d.Palette))
	for i, s := range d.Palette {
		rid, ok := reg.StateToRuntimeID(s.Name, s.States)
		if !ok {
			return fmt.Errorf("Cannot retain block state %s", s.Name)
		}
		rids[i] = rid
	}
	minX, maxX := offset[0]>>4, (offset[0]+d.Size[0]-1)>>4
	minZ, maxZ := offset[2]>>4, (offset[2]+d.Size[2]-1)>>4
	lo, hi := int(offset[1])&^15, (int(offset[1]+d.Size[1]-1) | 15)
	if lo < -2048 || hi > 2047 {
		return fmt.Errorf("Y range is outside supported Bedrock chunk heights")
	}
	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			col := &chunk.Column{Chunk: chunk.New(reg, cube.Range{lo, hi})}
			b := d.Bounds()
			b.Min[0] = max(0, x*16-offset[0])
			b.Max[0] = min(d.Size[0]-1, x*16+15-offset[0])
			b.Min[2] = max(0, z*16-offset[2])
			b.Max[2] = min(d.Size[2]-1, z*16+15-offset[2])
			err := each(ctx, b, func(p Vec3) error {
				c := d.Cell(p)
				if !c.Known {
					return nil
				}
				q := Vec3{p[0] + offset[0], p[1] + offset[1], p[2] + offset[2]}
				for layer, idx := range c.Layers {
					if idx >= 0 {
						col.Chunk.SetBlock(uint8(q[0]&15), int16(q[1]), uint8(q[2]&15), uint8(layer), rids[idx])
					}
				}
				if v := d.BlockData[Key(p)]; v != nil {
					if be, ok := v["block_entity_data"].(map[string]any); ok {
						be = cloneMap(be)
						translateBlockData(be, Vec3{offset[0] - d.Origin[0], offset[1] - d.Origin[1], offset[2] - d.Origin[2]})
						be["x"], be["y"], be["z"] = q[0], q[1], q[2]
						col.BlockEntities = append(col.BlockEntities, chunk.BlockEntity{Pos: cube.Pos{int(q[0]), int(q[1]), int(q[2])}, Data: be})
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			for _, e := range d.Entities {
				p, ok := entityPosition(e)
				if !ok || isPlayer(e) {
					continue
				}
				p[0] += float64(offset[0])
				p[1] += float64(offset[1])
				p[2] += float64(offset[2])
				if int32(math.Floor(p[0]/16)) == x && int32(math.Floor(p[2]/16)) == z {
					v := cloneMap(e)
					setEntityPosition(v, p)
					id := int64(len(col.Entities)+1) + (int64(x) << 32) + (int64(z) << 16)
					v["UniqueID"] = id
					col.Entities = append(col.Entities, chunk.Entity{ID: id, Data: v})
				}
			}
			if err = visit(world.ChunkPos{x, z}, col); err != nil {
				return err
			}
		}
	}
	return nil
}
