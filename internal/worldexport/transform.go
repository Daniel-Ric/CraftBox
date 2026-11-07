package worldexport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	_ "github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/df-mc/goleveldb/leveldb/util"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type Bounds struct {
	MinX int32 `json:"minX"`
	MinZ int32 `json:"minZ"`
	MaxX int32 `json:"maxX"`
	MaxZ int32 `json:"maxZ"`
}

func (b Bounds) Contains(x, z int32) bool {
	return x >= b.MinX && x <= b.MaxX && z >= b.MinZ && z <= b.MaxZ
}

type DimensionRule struct {
	Source int32   `json:"source"`
	Target *int32  `json:"target"`
	Bounds *Bounds `json:"bounds,omitempty"`
}
type BiomeRule struct {
	Source uint32 `json:"source"`
	Target uint32 `json:"target"`
}
type ChunkSelection struct {
	X         int32 `json:"x"`
	Z         int32 `json:"z"`
	Dimension int32 `json:"dimension"`
	Included  bool  `json:"included"`
}
type DimensionInfo struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	Chunks int    `json:"chunks"`
	Kept   int    `json:"kept"`
}
type BiomeChoice struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}
type Plan struct {
	Dimensions       []DimensionInfo  `json:"dimensions"`
	Chunks           []ChunkSelection `json:"chunks"`
	KeptChunks       int              `json:"keptChunks"`
	ExcludedChunks   int              `json:"excludedChunks"`
	PreviewTruncated bool             `json:"previewTruncated"`
	Records          int              `json:"records"`
	Biomes           []BiomeChoice    `json:"biomes"`
	positions        map[position]bool
	spawnDefault     int32
}
type Progress struct {
	Stage     string `json:"stage"`
	Completed int    `json:"completed"`
	Total     int    `json:"total"`
}
type position struct{ x, z, dim int32 }

func validateTransforms(o Options) error {
	if o.PlayerDimension != nil && o.Settings.Spawn == nil {
		return fmt.Errorf("moving the player to another dimension requires an explicit spawn point")
	}
	dims := map[int32]bool{}
	for _, r := range o.Dimensions {
		if dims[r.Source] {
			return fmt.Errorf("duplicate dimension rule %d", r.Source)
		}
		dims[r.Source] = true
		if b := r.Bounds; b != nil && (b.MinX > b.MaxX || b.MinZ > b.MaxZ) {
			return fmt.Errorf("invalid bounds for dimension %d", r.Source)
		}
	}
	biomes := map[uint32]bool{}
	for _, r := range o.Biomes {
		if biomes[r.Source] {
			return fmt.Errorf("duplicate biome rule %d", r.Source)
		}
		biomes[r.Source] = true
		if _, ok := world.BiomeByID(int(r.Target)); !ok {
			return fmt.Errorf("unknown target biome %d", r.Target)
		}
	}
	return nil
}
func destination(o Options, p position) (position, bool) {
	for _, r := range o.Dimensions {
		if r.Source == p.dim {
			if r.Target == nil || r.Bounds != nil && !r.Bounds.Contains(p.x, p.z) {
				return position{}, false
			}
			p.dim = *r.Target
			break
		}
	}
	return p, true
}
func chunkIndex(x, z, dim int32) []byte {
	raw := binary.LittleEndian.AppendUint32(nil, uint32(x))
	raw = binary.LittleEndian.AppendUint32(raw, uint32(z))
	if dim != 0 {
		raw = binary.LittleEndian.AppendUint32(raw, uint32(dim))
	}
	return raw
}
func parseIndex(key []byte) (position, bool) {
	if len(key) != 8 && len(key) != 12 {
		return position{}, false
	}
	p := position{x: int32(binary.LittleEndian.Uint32(key)), z: int32(binary.LittleEndian.Uint32(key[4:]))}
	if len(key) == 12 {
		p.dim = int32(binary.LittleEndian.Uint32(key[8:]))
	}
	return p, true
}
func chunkRecord(key []byte) (position, int, bool) {
	n := 8
	if len(key) == 13 || len(key) == 14 {
		n = 12
	}
	if len(key) != n+1 && len(key) != n+2 {
		return position{}, 0, false
	}
	tag := key[n]
	if !(tag >= 0x2b && tag <= 0x40) && tag != 'v' && tag != '|' {
		return position{}, 0, false
	}
	if len(key) == n+2 && tag != '/' {
		return position{}, 0, false
	}
	p, ok := parseIndex(key[:n])
	return p, n, ok
}
func dimensionName(id int32) string {
	switch id {
	case 0:
		return "Overworld"
	case 1:
		return "Nether"
	case 2:
		return "End"
	}
	return fmt.Sprintf("Custom dimension %d", id)
}
func Preview(ctx context.Context, folder string, o Options) (Plan, error) {
	if err := o.Validate(); err != nil {
		return Plan{}, err
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	if _, _, err := ReadMetadata(folder); err != nil {
		return Plan{}, err
	}
	db, err := leveldb.OpenFile(filepath.Join(folder, "db"), &opt.Options{ReadOnly: true, ErrorIfMissing: true})
	if err != nil {
		return Plan{}, err
	}
	defer db.Close()
	plan, err := inspect(ctx, db, o, true)
	if err != nil {
		return Plan{}, err
	}
	data, _, err := ReadMetadata(folder)
	if err != nil {
		return Plan{}, err
	}
	data, err = ApplySettings(data, o)
	if err != nil {
		return Plan{}, err
	}
	if err = validateWorldSpawn(data, o, plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func validateWorldSpawn(data map[string]any, o Options, plan Plan) error {
	if len(o.Dimensions) == 0 && o.PlayerDimension == nil && o.Settings.Spawn == nil {
		return nil
	}
	id, present := data["SpawnDimension"].(int32)
	if !present {
		id = plan.spawnDefault
	}
	if o.PlayerDimension != nil {
		id = *o.PlayerDimension
	}
	x, _ := data["SpawnX"].(int32)
	z, _ := data["SpawnZ"].(int32)
	p := position{x >> 4, z >> 4, id}
	target, keep := destination(o, p)
	if !keep || !plan.positions[p] {
		return fmt.Errorf("world spawn is outside retained terrain; set a spawn in a retained source dimension")
	}
	y, _ := data["SpawnY"].(int32)
	lo, hi := int32(-2048), int32(2047)
	switch target.dim {
	case 0:
		lo, hi = -64, 319
	case 1:
		lo, hi = 0, 127
	case 2:
		lo, hi = 0, 255
	}
	if y < lo || y+1 > hi {
		return fmt.Errorf("world spawn height %d is outside target dimension %d", y, target.dim)
	}
	return nil
}
func Describe(ctx context.Context, folder string) (Plan, error) {
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	if _, _, err := ReadMetadata(folder); err != nil {
		return Plan{}, err
	}
	db, err := leveldb.OpenFile(filepath.Join(folder, "db"), &opt.Options{ReadOnly: true, ErrorIfMissing: true})
	if err != nil {
		return Plan{}, err
	}
	defer db.Close()
	return inspect(ctx, db, Options{}, false)
}
func inspect(ctx context.Context, db *leveldb.DB, o Options, checkPlayer bool) (Plan, error) {
	plan := Plan{Dimensions: []DimensionInfo{}, Chunks: []ChunkSelection{}, Biomes: []BiomeChoice{}}
	positions := map[position]bool{}
	targets := map[position]position{}
	counts := map[int32]*DimensionInfo{}
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		if err := ctx.Err(); err != nil {
			return Plan{}, err
		}
		plan.Records++
		key := iter.Key()
		p, n, ok := chunkRecord(key)
		if !ok {
			continue
		}
		target, keep := destination(o, p)
		if keep && target.dim != p.dim && key[n] == '/' && len(key) == n+2 {
			y := int(int8(key[n+1])) * 16
			minY, maxY := -2048, 2047
			switch target.dim {
			case 0:
				minY, maxY = -64, 319
			case 1:
				minY, maxY = 0, 127
			case 2:
				minY, maxY = 0, 255
			}
			if y < minY || y+15 > maxY {
				return Plan{}, fmt.Errorf("chunk (%d, %d) in dimension %d has terrain at Y=%d outside target dimension %d (%d to %d)", p.x, p.z, p.dim, y, target.dim, minY, maxY)
			}
		}
		if _, seen := positions[p]; seen {
			continue
		}
		positions[p] = keep
		info := counts[p.dim]
		if info == nil {
			info = &DimensionInfo{ID: p.dim, Name: dimensionName(p.dim)}
			counts[p.dim] = info
		}
		info.Chunks++
		if keep {
			if previous, collision := targets[target]; collision && previous != p {
				return Plan{}, fmt.Errorf("dimensions %d and %d overlap at target chunk (%d, %d) in dimension %d", previous.dim, p.dim, p.x, p.z, target.dim)
			}
			targets[target] = p
			info.Kept++
			plan.KeptChunks++
		} else {
			plan.ExcludedChunks++
		}
	}
	if err := iter.Error(); err != nil {
		return Plan{}, err
	}
	if plan.KeptChunks == 0 {
		return Plan{}, fmt.Errorf("the export selection contains no chunks")
	}
	plan.positions = positions
	if len(counts) == 1 && counts[0] == nil {
		for dimension := range counts {
			plan.spawnDefault = dimension
		}
	}
	if raw, err := db.Get([]byte("~local_player"), nil); checkPlayer && err == nil {
		if _, err = remapPlayer(raw, o, plan); err != nil {
			return Plan{}, err
		}
	} else if checkPlayer && err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return Plan{}, err
	}
	for source := range ruleSources(o) {
		if counts[source] == nil {
			return Plan{}, fmt.Errorf("dimension %d is not present in this world", source)
		}
	}
	for _, rule := range o.Dimensions {
		if rule.Target != nil && (*rule.Target < 0 || *rule.Target > 2) && counts[*rule.Target] == nil {
			return Plan{}, fmt.Errorf("target custom dimension %d is not defined by this source world; choose an existing dimension", *rule.Target)
		}
	}
	for _, info := range counts {
		plan.Dimensions = append(plan.Dimensions, *info)
	}
	slices.SortFunc(plan.Dimensions, func(a, b DimensionInfo) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	sorted := make([]position, 0, len(positions))
	for p := range positions {
		sorted = append(sorted, p)
	}
	slices.SortFunc(sorted, func(a, b position) int {
		for _, v := range [][2]int32{{a.dim, b.dim}, {a.z, b.z}, {a.x, b.x}} {
			if v[0] < v[1] {
				return -1
			}
			if v[0] > v[1] {
				return 1
			}
		}
		return 0
	})
	if len(sorted) > 50000 {
		plan.PreviewTruncated = true
		sorted = sorted[:50000]
	}
	for _, p := range sorted {
		plan.Chunks = append(plan.Chunks, ChunkSelection{p.x, p.z, p.dim, positions[p]})
	}
	for _, b := range world.Biomes() {
		plan.Biomes = append(plan.Biomes, BiomeChoice{uint32(b.EncodeBiome()), b.String()})
	}
	slices.SortFunc(plan.Biomes, func(a, b BiomeChoice) int { return strings.Compare(a.Name, b.Name) })
	return plan, nil
}
func ruleSources(o Options) map[int32]bool {
	out := map[int32]bool{}
	for _, r := range o.Dimensions {
		out[r.Source] = true
	}
	return out
}

func Copy(ctx context.Context, source, dest string, o Options, progress func(Progress)) (result error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := o.Validate(); err != nil {
		return err
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.VolumeName(source), filepath.VolumeName(dest)) {
		relative, err := filepath.Rel(source, dest)
		if err != nil {
			return err
		}
		if relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("destination must be outside the source world")
		}
	}
	src, err := leveldb.OpenFile(filepath.Join(source, "db"), &opt.Options{ReadOnly: true, ErrorIfMissing: true})
	if err != nil {
		return err
	}
	defer src.Close()
	plan, err := inspect(ctx, src, o, true)
	if err != nil {
		return err
	}
	metadata, _, err := ReadMetadata(source)
	if err != nil {
		return err
	}
	metadata, err = ApplySettings(metadata, o)
	if err != nil {
		return err
	}
	if err = validateWorldSpawn(metadata, o, plan); err != nil {
		return err
	}
	if err = os.Mkdir(dest, 0755); err != nil {
		return err
	}
	defer func() {
		if result != nil {
			result = errors.Join(result, os.RemoveAll(dest))
		}
	}()
	report := func(stage string, done, total int) {
		if progress != nil {
			progress(Progress{stage, done, total})
		}
	}
	report("copy", 0, 0)
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == "db" {
			return filepath.SkipDir
		}
		target := filepath.Join(dest, rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("world contains a symbolic link: %s", rel)
		}
		if entry.IsDir() {
			return os.Mkdir(target, 0755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported world file %s", rel)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.Create(target)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, contextReader{ctx, input})
		return errors.Join(copyErr, output.Close())
	})
	if err != nil {
		return err
	}
	dst, err := leveldb.OpenFile(filepath.Join(dest, "db"), nil)
	if err != nil {
		return err
	}
	err = copyRecords(ctx, src, dst, o, plan, report)
	err = errors.Join(err, dst.Close())
	if err != nil {
		return err
	}
	report("metadata", 0, 1)
	if err = UpdateMetadata(ctx, dest, o); err != nil {
		return err
	}
	if err = remapSpawnDimension(ctx, dest, o, plan); err != nil {
		return err
	}
	report("metadata", 1, 1)
	if o.Compact {
		report("compaction", 0, 1)
		if err = Compact(ctx, dest); err != nil {
			return err
		}
		report("compaction", 1, 1)
	}
	return ctx.Err()
}

type contextReader struct {
	context.Context
	io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}

func copyRecords(ctx context.Context, src, dst *leveldb.DB, o Options, plan Plan, report func(string, int, int)) error {
	actors := map[int64]bool{}
	actorDimensions := map[int64]int32{}
	iter := src.NewIterator(nil, nil)
	for iter.Next() {
		if err := ctx.Err(); err != nil {
			iter.Release()
			return err
		}
		key := iter.Key()
		if !bytes.HasPrefix(key, []byte("digp")) {
			continue
		}
		p, ok := parseIndex(key[4:])
		if !ok {
			continue
		}
		target, keep := destination(o, p)
		if !keep {
			continue
		}
		value := iter.Value()
		if len(value)%8 != 0 {
			iter.Release()
			return fmt.Errorf("invalid actor index in dimension %d", p.dim)
		}
		for i := 0; i < len(value); i += 8 {
			id := int64(binary.LittleEndian.Uint64(value[i:]))
			if previous, exists := actorDimensions[id]; exists && previous != target.dim {
				iter.Release()
				return fmt.Errorf("actor %d is indexed in multiple target dimensions", id)
			}
			actors[id] = true
			actorDimensions[id] = target.dim
		}
	}
	err := iter.Error()
	iter.Release()
	if err != nil {
		return err
	}
	biomes := map[uint32]uint32{}
	for _, r := range o.Biomes {
		biomes[r.Source] = r.Target
	}
	batch := leveldb.MakeBatch(0)
	count := 0
	iter = src.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		key, value := bytes.Clone(iter.Key()), bytes.Clone(iter.Value())
		count++
		if p, n, ok := chunkRecord(key); ok {
			target, keep := destination(o, p)
			if !keep {
				continue
			}
			tag := key[n]
			key = append(chunkIndex(target.x, target.z, target.dim), key[n:]...)
			if len(biomes) > 0 && (tag == '+' || tag == '-') {
				value, err = remapBiomes(value, tag, biomes)
				if err != nil {
					return fmt.Errorf("biomes at (%d, %d) dimension %d: %w", p.x, p.z, p.dim, err)
				}
			}
		} else if bytes.HasPrefix(key, []byte("digp")) {
			if p, ok := parseIndex(key[4:]); ok {
				target, keep := destination(o, p)
				if !keep {
					continue
				}
				key = append([]byte("digp"), chunkIndex(target.x, target.z, target.dim)...)
			}
		} else if bytes.HasPrefix(key, []byte("actorprefix")) && len(key) == 19 {
			id := int64(binary.LittleEndian.Uint64(key[11:]))
			if !actors[id] {
				continue
			}
			value, err = filterActorLinks(value, actors, actorDimensions[id])
			if err != nil {
				return err
			}
		} else if string(key) == "~local_player" {
			value, err = remapPlayer(value, o, plan)
			if err != nil {
				return err
			}
		}
		batch.Put(key, value)
		if batch.Len() >= 1024 {
			if err = dst.Write(batch, nil); err != nil {
				return err
			}
			batch.Reset()
			report("database", count, plan.Records)
		}
	}
	if err = iter.Error(); err != nil {
		return err
	}
	if err = dst.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
		return err
	}
	report("database", plan.Records, plan.Records)
	return nil
}
func filterActorLinks(raw []byte, actors map[int64]bool, targetDimension int32) ([]byte, error) {
	var data map[string]any
	if err := nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); err != nil {
		return nil, fmt.Errorf("actor NBT: %w", err)
	}
	changed := false
	if dimension, ok := data["DimensionId"].(int32); ok && dimension != targetDimension {
		data["DimensionId"] = targetDimension
		changed = true
	}
	links, ok := data["LinksTag"].([]any)
	if !ok {
		if changed {
			return nbt.MarshalEncoding(data, nbt.LittleEndian)
		}
		return raw, nil
	}
	kept := []any{}
	for _, link := range links {
		row, ok := link.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid actor link")
		}
		id, ok := row["entityID"].(int64)
		if !ok {
			return nil, fmt.Errorf("invalid linked actor ID")
		}
		if actors[id] {
			kept = append(kept, link)
		}
	}
	if len(kept) == len(links) {
		if changed {
			return nbt.MarshalEncoding(data, nbt.LittleEndian)
		}
		return raw, nil
	}
	if len(kept) == 0 {
		delete(data, "LinksTag")
	} else {
		data["LinksTag"] = kept
	}
	return nbt.MarshalEncoding(data, nbt.LittleEndian)
}
func remapPlayer(raw []byte, o Options, plan Plan) ([]byte, error) {
	var data map[string]any
	if err := nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); err != nil {
		return nil, err
	}
	id, _ := data["DimensionId"].(int32)
	p := position{dim: id}
	if o.PlayerDimension != nil {
		p.dim = *o.PlayerDimension
	}
	y := float64(0)
	switch pos := data["Pos"].(type) {
	case []float32:
		if len(pos) >= 3 {
			p.x = int32(floorChunk(float64(pos[0])))
			p.z = int32(floorChunk(float64(pos[2])))
			y = float64(pos[1])
		}
	case []any:
		if len(pos) >= 3 {
			x, _ := pos[0].(float32)
			z, _ := pos[2].(float32)
			p.x = int32(floorChunk(float64(x)))
			p.z = int32(floorChunk(float64(z)))
			v, _ := pos[1].(float32)
			y = float64(v)
		}
	}
	if o.Settings.Spawn != nil {
		p.x, p.z = o.Settings.Spawn[0]>>4, o.Settings.Spawn[2]>>4
		y = float64(o.Settings.Spawn[1])
	}
	target, keep := destination(o, p)
	if !keep {
		return nil, fmt.Errorf("player location is excluded; set a spawn point in a retained dimension before exporting")
	}
	if !plan.positions[p] {
		return nil, fmt.Errorf("player/spawn location (%d, %d) in dimension %d is outside retained terrain", p.x, p.z, p.dim)
	}
	if o.Settings.Spawn != nil || target.dim != id {
		lo, hi := int32(-2048), int32(2047)
		switch target.dim {
		case 0:
			lo, hi = -64, 319
		case 1:
			lo, hi = 0, 127
		case 2:
			lo, hi = 0, 255
		}
		if math.IsNaN(y) || math.IsInf(y, 0) || y < float64(lo) || y+1 > float64(hi) {
			return nil, fmt.Errorf("player height %g is outside target dimension %d", y, target.dim)
		}
	}
	changed := id != target.dim
	if changed {
		data["DimensionId"] = target.dim
	}
	if o.Settings.Spawn != nil {
		data["SpawnDimension"] = target.dim
		changed = true
	} else if spawnDim, ok := data["SpawnDimension"].(int32); ok {
		x, _ := data["SpawnX"].(int32)
		z, _ := data["SpawnZ"].(int32)
		p := position{x >> 4, z >> 4, spawnDim}
		mapped, kept := destination(o, p)
		if !kept || !plan.positions[p] {
			return nil, fmt.Errorf("saved player spawn is excluded; set a spawn point in retained terrain")
		}
		if spawnDim != mapped.dim {
			data["SpawnDimension"] = mapped.dim
			changed = true
		}
	}
	if !changed {
		return raw, nil
	}
	return nbt.MarshalEncoding(data, nbt.LittleEndian)
}
func remapSpawnDimension(ctx context.Context, folder string, o Options, plan Plan) error {
	if len(o.Dimensions) == 0 && o.PlayerDimension == nil {
		return nil
	}
	data, version, err := ReadMetadata(folder)
	if err != nil {
		return err
	}
	id, present := data["SpawnDimension"].(int32)
	if !present {
		id = plan.spawnDefault
	}
	if o.PlayerDimension != nil {
		id = *o.PlayerDimension
	}
	x, _ := data["SpawnX"].(int32)
	z, _ := data["SpawnZ"].(int32)
	target, keep := destination(o, position{x >> 4, z >> 4, id})
	if !keep {
		return fmt.Errorf("world spawn is excluded; set a spawn in retained terrain")
	}
	data["SpawnDimension"] = target.dim
	payload, err := nbt.MarshalEncoding(data, nbt.LittleEndian)
	if err != nil {
		return err
	}
	encoded := binary.LittleEndian.AppendUint32(nil, uint32(version))
	encoded = binary.LittleEndian.AppendUint32(encoded, uint32(len(payload)))
	encoded = append(encoded, payload...)
	if err = ctx.Err(); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(folder, "level.dat"), encoded)
}

func floorChunk(v float64) int64 {
	n := int64(v)
	if v < float64(n) {
		n--
	}
	return n >> 4
}
func Compact(ctx context.Context, folder string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db, err := leveldb.OpenFile(filepath.Join(folder, "db"), &opt.Options{ErrorIfMissing: true})
	if err != nil {
		return err
	}
	defer db.Close()
	done := make(chan error, 1)
	go func() { done <- db.CompactRange(util.Range{}) }()
	select {
	case err = <-done:
		return err
	case <-ctx.Done():
		db.Close()
		<-done
		return ctx.Err()
	}
}

func remapBiomes(raw []byte, tag byte, mapping map[uint32]uint32) ([]byte, error) {
	out := bytes.Clone(raw)
	if len(out) < 512 {
		return nil, fmt.Errorf("missing height map")
	}
	if tag == '-' {
		if len(out) != 768 {
			return nil, fmt.Errorf("unsupported legacy 2D biome record length %d", len(out))
		}
		for i := 512; i < len(out); i++ {
			if v, ok := mapping[uint32(out[i])]; ok {
				if v > 255 {
					return nil, fmt.Errorf("biome %d cannot be stored in legacy 2D format", v)
				}
				out[i] = byte(v)
			}
		}
		return out, nil
	}
	offset, storages := 512, 0
	for offset < len(out) {
		bits := int(out[offset] >> 1)
		network := out[offset] & 1
		offset++
		if network != 0 {
			return nil, fmt.Errorf("network biome palette in disk record")
		}
		if bits == 127 {
			if storages == 0 {
				return nil, fmt.Errorf("first biome storage cannot reference previous")
			}
			storages++
			continue
		}
		if bits != 0 && bits != 1 && bits != 2 && bits != 3 && bits != 4 && bits != 5 && bits != 6 && bits != 8 && bits != 16 && bits != 32 {
			return nil, fmt.Errorf("invalid biome palette size %d", bits)
		}
		words := 0
		if bits > 0 {
			per := 32 / bits
			words = (4096 + per - 1) / per
		}
		offset += words * 4
		if offset > len(out) {
			return nil, fmt.Errorf("truncated biome indices")
		}
		count := uint32(1)
		if bits > 0 {
			if offset+4 > len(out) {
				return nil, fmt.Errorf("missing biome palette length")
			}
			count = binary.LittleEndian.Uint32(out[offset:])
			offset += 4
		}
		if count == 0 || count > 65536 || uint64(offset)+uint64(count)*4 > uint64(len(out)) {
			return nil, fmt.Errorf("invalid biome palette length %d", count)
		}
		for j := uint32(0); j < count; j++ {
			v := binary.LittleEndian.Uint32(out[offset:])
			if target, ok := mapping[v]; ok {
				binary.LittleEndian.PutUint32(out[offset:], target)
			}
			offset += 4
		}
		storages++
		if storages > 256 {
			return nil, fmt.Errorf("too many biome storages")
		}
	}
	if storages == 0 {
		return nil, fmt.Errorf("missing biome storages")
	}
	return out, nil
}
