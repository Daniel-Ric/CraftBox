package scene

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/util"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type CompositionPart struct {
	Store  *Store
	Keep   func(int32, int32) bool
	Offset [3]int32
}
type CompositionResult struct {
	Conflicts     int64               `json:"conflicts"`
	Examples      [][3]int32          `json:"examples"`
	Blocks        int64               `json:"blocks"`
	Chunks        int                 `json:"chunks"`
	BlockEntities int                 `json:"blockEntities"`
	Entities      int                 `json:"entities"`
	Spawn         [3]int32            `json:"spawn"`
	Registry      world.BlockRegistry `json:"-"`
}

func entityNBTKey(id int64) []byte {
	key := append([]byte("entity-nbt:"), make([]byte, 8)...)
	binary.LittleEndian.PutUint64(key[len(key)-8:], uint64(id))
	return key
}

func (s *Store) PutEntityNBT(id int64, data map[string]any) error {
	raw, err := nbt.MarshalEncoding(data, nbt.LittleEndian)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("Scene closed")
	}
	return s.db.Put(entityNBTKey(id), raw, nil)
}
func fullBlockNBTKey(p world.ChunkPos) []byte { return append(chunkKey(p), 'b') }
func encodeBlockNBT(entities map[cube.Pos]map[string]any) ([]byte, error) {
	entries := make([]map[string]any, 0, len(entities))
	for pos, data := range entities {
		copy := map[string]any{}
		for k, v := range data {
			copy[k] = v
		}
		copy["x"], copy["y"], copy["z"] = int32(pos[0]), int32(pos[1]), int32(pos[2])
		entries = append(entries, copy)
	}
	return nbt.MarshalEncoding(map[string]any{"entries": entries}, nbt.LittleEndian)
}

type compositionSnapshot struct {
	part      CompositionPart
	db        *leveldb.Snapshot
	positions []world.ChunkPos
	reg       world.BlockRegistry
	states    map[uint32]world.BlockState
}

func airName(name string) bool {
	return name == "minecraft:air" || name == "minecraft:cave_air" || name == "minecraft:void_air"
}

func Compose(ctx context.Context, parts []CompositionPart, folder, name string, progress func(int, int)) (result CompositionResult, err error) {
	if len(parts) == 0 || len(parts) > 16 {
		return result, fmt.Errorf("Choose 1–16 source placements")
	}
	snapshots := []compositionSnapshot{}
	defer func() {
		for _, s := range snapshots {
			s.db.Release()
		}
	}()
	reg := world.NewBlockRegistry()
	seen := map[string]bool{}
	keyFor := func(name string, props map[string]any) string {

		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var key strings.Builder
		fmt.Fprintf(&key, "%q", name)
		for _, k := range keys {
			v := props[k]
			if b, ok := v.(bool); ok {
				v = uint8(0)
				if b {
					v = uint8(1)
				}
			}
			fmt.Fprintf(&key, "/%q:%T:%#v", k, v, v)
		}
		return key.String()
	}
	world.DefaultBlockRegistry.Finalize()
	for _, b := range world.DefaultBlockRegistry.Blocks() {
		n, p := b.EncodeBlock()
		seen[keyFor(n, p)] = true
	}
	total := 0
	for _, p := range parts {
		if p.Store == nil {
			return result, fmt.Errorf("Source map unavailable")
		}
		if err = ctx.Err(); err != nil {
			return result, err
		}
		s := p.Store
		s.mu.Lock()
		if s.closed || s.storageError != nil {
			s.mu.Unlock()
			return result, fmt.Errorf("Source map closed")
		}
		snap, e := s.db.GetSnapshot()
		if e != nil {
			s.mu.Unlock()
			return result, e
		}
		item := compositionSnapshot{part: p, db: snap, reg: s.registry}
		for pos := range s.captured {
			if p.Keep == nil || p.Keep(pos[0], pos[1]) {
				item.positions = append(item.positions, pos)
			}
		}
		s.mu.Unlock()
		snapshots = append(snapshots, item)
		total += len(item.positions)
		states := map[uint32]world.BlockState{}
		register := func(rid uint32) error {
			if _, ok := states[rid]; ok {
				return nil
			}
			n, props, ok := item.reg.RuntimeIDToState(rid)
			if !ok || n == "" {
				return fmt.Errorf("Unknown source block ID %d", rid)
			}
			state := world.BlockState{Name: n, Properties: props}
			states[rid] = state
			key := keyFor(n, props)
			if !seen[key] {
				if e := registerCompositionState(reg, state); e != nil {
					return e
				}
				seen[key] = true
			}
			return nil
		}
		if e = register(item.reg.AirRuntimeID()); e != nil {
			return result, e
		}
		for _, pos := range item.positions {
			if e = ctx.Err(); e != nil {
				return result, e
			}
			c, e := compositionChunk(item, pos)
			if e != nil {
				return result, e
			}
			for _, sub := range c.Sub() {
				for _, layer := range sub.Layers() {
					palette := layer.Palette()
					for i := 0; i < palette.Len(); i++ {
						if e = register(palette.Value(uint16(i))); e != nil {
							return result, e
						}
					}
				}
			}
		}
		snapshots[len(snapshots)-1].states = states
	}
	reg.Finalize()
	result.Registry = reg
	db, e := (mcdb.Config{Blocks: reg}).Open(folder)
	if e != nil {
		return result, e
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	cache := map[world.ChunkPos]*chunk.Column{}
	written := map[world.ChunkPos]bool{}
	flush := func() error {
		for pos, col := range cache {
			if e := db.StoreColumn(pos, world.Overworld, col); e != nil {
				return e
			}
			written[pos] = true
		}
		clear(cache)
		return nil
	}
	get := func(pos world.ChunkPos) (*chunk.Column, error) {
		if c := cache[pos]; c != nil {
			return c, nil
		}
		if len(cache) >= 48 {
			if e := flush(); e != nil {
				return nil, e
			}
		}
		var col *chunk.Column
		if written[pos] {
			var e error
			col, e = db.LoadColumn(pos, world.Overworld)
			if e != nil {
				return nil, e
			}
		} else {
			col = &chunk.Column{Chunk: chunk.New(reg, cube.Range{-64, 319})}
		}
		cache[pos] = col
		return col, nil
	}
	completed := 0
	entityID := int64(1)
	for _, source := range snapshots {
		ids := map[uint32]uint32{}
		isAir := map[uint32]bool{}
		for rid, state := range source.states {
			n, props := state.Name, state.Properties
			id, ok := reg.StateToRuntimeID(n, props)
			if !ok {
				return result, fmt.Errorf("Cannot translate block %s", n)
			}
			ids[rid] = id
			isAir[rid] = airName(n)
		}
		sort.Slice(source.positions, func(i, j int) bool {
			a, b := source.positions[i], source.positions[j]
			return a[0] < b[0] || a[0] == b[0] && a[1] < b[1]
		})
		for _, pos := range source.positions {
			if err = ctx.Err(); err != nil {
				return result, err
			}
			raw, e := source.db.Get(chunkKey(pos), nil)
			if e != nil {
				return result, e
			}
			if len(raw) < 8 {
				return result, fmt.Errorf("Invalid source chunk")
			}
			bounds := cube.Range{int(int32(binary.LittleEndian.Uint32(raw))), int(int32(binary.LittleEndian.Uint32(raw[4:])))}
			c, e := chunk.NetworkDecode(source.reg, raw[8:], (bounds.Height()>>4)+1, bounds)
			if e != nil {
				return result, e
			}
			changed := map[cube.Pos]bool{}
			for sy, sub := range c.Sub() {
				if sub.Empty() {
					continue
				}
				for dy := 0; dy < 16; dy++ {
					y := bounds[0] + sy*16 + dy
					ty := y + int(source.part.Offset[1])
					for x := 0; x < 16; x++ {
						for z := 0; z < 16; z++ {
							a, b := c.Block(uint8(x), int16(y), uint8(z), 0), c.Block(uint8(x), int16(y), uint8(z), 1)
							if isAir[a] && isAir[b] {
								continue
							}
							tx, tz := int64(pos[0])*16+int64(x)+int64(source.part.Offset[0]), int64(pos[1])*16+int64(z)+int64(source.part.Offset[2])
							if tx < -30_000_000 || tx > 30_000_000 || tz < -30_000_000 || tz > 30_000_000 {
								return result, fmt.Errorf("Placement exceeds the world border")
							}
							if ty < -64 || ty > 319 {
								return result, fmt.Errorf("Non-air blocks would leave Y −64…319; adjust the Y offset")
							}
							tp := world.ChunkPos{int32(tx >> 4), int32(tz >> 4)}
							target, e := get(tp)
							if e != nil {
								return result, e
							}
							xx, zz := uint8(tx&15), uint8(tz&15)
							occupied := target.Chunk.Block(xx, int16(ty), zz, 0) != reg.AirRuntimeID() || target.Chunk.Block(xx, int16(ty), zz, 1) != reg.AirRuntimeID()
							if occupied {
								kept := target.BlockEntities[:0]
								for _, be := range target.BlockEntities {
									if be.Pos != (cube.Pos{int(tx), ty, int(tz)}) {
										kept = append(kept, be)
									}
								}
								target.BlockEntities = kept
								result.Conflicts++
								if len(result.Examples) < 32 {
									result.Examples = append(result.Examples, [3]int32{int32(tx), int32(ty), int32(tz)})
								}
							}
							for layer, id := range []uint32{a, b} {
								target.Chunk.SetBlock(xx, int16(ty), zz, uint8(layer), ids[id])
							}
							target.Chunk.SetBiome(xx, int16(ty), zz, c.Biome(uint8(x), int16(y), uint8(z)))
							bp := cube.Pos{int(tx), ty, int(tz)}
							changed[bp] = true
							result.Blocks++
							if result.Blocks == 1 {
								result.Spawn = [3]int32{int32(tx), int32(min(319, ty+2)), int32(tz)}
							}
						}
					}
				}
			}

			for _, col := range cache {
				kept := col.BlockEntities[:0]
				for _, be := range col.BlockEntities {
					if !changed[be.Pos] {
						kept = append(kept, be)
					}
				}
				col.BlockEntities = kept
			}
			if raw, e = source.db.Get(fullBlockNBTKey(pos), nil); e == nil {
				var data struct {
					Entries []map[string]any `nbt:"entries"`
				}
				if e = nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); e != nil {
					return result, e
				}
				for _, be := range data.Entries {
					shiftBlockNBT(be, source.part.Offset)
					p := cube.Pos{number(be["x"]), number(be["y"]), number(be["z"])}
					if !changed[p] {
						continue
					}
					col, e := get(world.ChunkPos{int32(p[0] >> 4), int32(p[2] >> 4)})
					if e != nil {
						return result, e
					}
					col.BlockEntities = append(col.BlockEntities, chunk.BlockEntity{Pos: p, Data: be})
					result.BlockEntities++
				}
			} else if e != leveldb.ErrNotFound {
				return result, e
			}
			completed++
			if progress != nil {
				progress(completed, total)
			}
		}
		entityIDs := map[int64]int64{}
		index := source.db.NewIterator(util.BytesPrefix([]byte("entity-nbt:")), nil)
		for index.Next() {
			if err = ctx.Err(); err != nil {
				index.Release()
				return result, err
			}
			var data map[string]any
			if e = nbt.UnmarshalEncoding(index.Value(), &data, nbt.LittleEndian); e != nil {
				index.Release()
				return result, e
			}
			v := vectorValue(data["Pos"], [3]float32{})
			p := world.ChunkPos{int32(math.Floor(float64(v[0]) / 16)), int32(math.Floor(float64(v[2]) / 16))}
			if source.part.Keep != nil && !source.part.Keep(p[0], p[1]) {
				continue
			}
			if _, e = source.db.Get(chunkKey(p), nil); e != nil {
				continue
			}
			old := rawEntityID(index.Key(), data)
			entityIDs[old] = entityID
			entityID++
		}
		e = index.Error()
		index.Release()
		if e != nil {
			return result, e
		}
		it := source.db.NewIterator(util.BytesPrefix([]byte("entity-nbt:")), nil)
		for it.Next() {
			if err = ctx.Err(); err != nil {
				it.Release()
				return result, err
			}
			var data map[string]any
			if e = nbt.UnmarshalEncoding(it.Value(), &data, nbt.LittleEndian); e != nil {
				it.Release()
				return result, e
			}
			v := vectorValue(data["Pos"], [3]float32{})
			p := world.ChunkPos{int32(math.Floor(float64(v[0]) / 16)), int32(math.Floor(float64(v[2]) / 16))}
			if source.part.Keep != nil && !source.part.Keep(p[0], p[1]) {
				continue
			}
			if _, e = source.db.Get(chunkKey(p), nil); e != nil {
				continue
			}
			shifted := []float32{v[0] + float32(source.part.Offset[0]), v[1] + float32(source.part.Offset[1]), v[2] + float32(source.part.Offset[2])}
			data["Pos"] = shifted
			mappedID, ok := entityIDs[rawEntityID(it.Key(), data)]
			if !ok {
				continue
			}
			data["UniqueID"] = mappedID
			for _, key := range []string{"OwnerNew", "LeasherID", "TargetID"} {
				if old, ok := data[key].(int64); ok {
					if id, known := entityIDs[old]; known {
						data[key] = id
					} else {
						data[key] = int64(-1)
					}
				}
			}
			if links, ok := data["LinksTag"].([]any); ok {
				kept := []any{}
				for _, value := range links {
					if link, ok := value.(map[string]any); ok {
						if old, ok := link["entityID"].(int64); ok {
							if id, found := entityIDs[old]; found {
								link["entityID"] = id
								kept = append(kept, link)
							}
						}
					}
				}
				data["LinksTag"] = kept
			}
			col, e := get(world.ChunkPos{int32(math.Floor(float64(shifted[0]) / 16)), int32(math.Floor(float64(shifted[2]) / 16))})
			if e != nil {
				it.Release()
				return result, e
			}
			col.Entities = append(col.Entities, chunk.Entity{ID: mappedID, Data: data})
			result.Entities++
		}
		e = it.Error()
		it.Release()
		if e != nil {
			return result, e
		}
	}
	if err = flush(); err != nil {
		return result, err
	}
	result.Chunks = len(written)
	if result.Blocks == 0 {
		return result, fmt.Errorf("The selected areas contain only air")
	}

	spawnFound := false
	for pos := range written {
		col, e := db.LoadColumn(pos, world.Overworld)
		if e != nil {
			return result, e
		}
		for i := 0; i < 256; i++ {
			x, z := uint8(i%16), uint8(i/16)
			y := col.Chunk.HighestBlock(x, z)
			rid := col.Chunk.Block(x, y, z, 0)
			if rid == reg.AirRuntimeID() || reg.LiquidBlock(rid) || int(y)+2 > 319 {
				continue
			}
			result.Spawn = [3]int32{pos[0]*16 + int32(x), int32(y) + 2, pos[1]*16 + int32(z)}
			spawnFound = true
			break
		}
		if spawnFound {
			break
		}
	}
	settings := db.Settings()
	settings.Name = name
	settings.Spawn = cube.Pos{int(result.Spawn[0]), int(result.Spawn[1]), int(result.Spawn[2])}
	settings.DefaultGameMode = world.GameModeCreative
	settings.Time = 6000
	settings.TimeCycle = false
	db.SaveSettings(settings)
	ld := db.LevelDat()
	ld.Generator = 2
	ld.FlatWorldLayers = `{"biome_id":1,"block_layers":[{"block_name":"minecraft:air","count":1}],"encoding_version":6,"structure_options":null}`
	ld.RandomTickSpeed = 0
	err = db.SaveLocalPlayerData(map[string]any{"Pos": []float32{float32(result.Spawn[0]) + .5, float32(result.Spawn[1]), float32(result.Spawn[2]) + .5}, "SpawnX": result.Spawn[0], "SpawnY": result.Spawn[1], "SpawnZ": result.Spawn[2], "DimensionId": int32(0), "PlayerGameMode": int32(1), "Inventory": []any{}})
	return result, err
}

func registerCompositionState(reg world.BlockRegistry, state world.BlockState) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("Incompatible block states for %s: %v", state.Name, p)
		}
	}()
	reg.RegisterBlockState(state)
	return nil
}

func rawEntityID(key []byte, data map[string]any) int64 {
	if id, ok := data["UniqueID"].(int64); ok {
		return id
	}
	return int64(binary.LittleEndian.Uint64(key[len(key)-8:]))
}
func shiftBlockNBT(data map[string]any, offset [3]int32) {
	for i, k := range []string{"x", "y", "z"} {
		data[k] = int32(number(data[k])) + offset[i]
	}
	for k, v := range data {
		lower := strings.ToLower(k)
		if lower == "pairx" {
			data[k] = int32(number(v)) + offset[0]
		} else if lower == "pairz" {
			data[k] = int32(number(v)) + offset[2]
		}
	}
}

func compositionChunk(source compositionSnapshot, pos world.ChunkPos) (*chunk.Chunk, error) {
	raw, err := source.db.Get(chunkKey(pos), nil)
	if err != nil {
		return nil, err
	}
	if len(raw) < 8 {
		return nil, fmt.Errorf("Invalid source chunk")
	}
	bounds := cube.Range{int(int32(binary.LittleEndian.Uint32(raw))), int(int32(binary.LittleEndian.Uint32(raw[4:])))}
	return chunk.NetworkDecode(source.reg, raw[8:], (bounds.Height()>>4)+1, bounds)
}
