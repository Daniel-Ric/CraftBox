package worldstate

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

const FirstCustomDimensionID = 1000

const keyDimensionNameIDTable = "DimensionNameIdTable"

type MergedDimension struct {
	ID int

	Identifier string

	Title string

	Spawn mgl32.Vec3

	Range cube.Range
}

func (d MergedDimension) ShortName() string {
	_, name, _ := strings.Cut(d.Identifier, ":")
	return name
}

var notIdentifier = regexp.MustCompile(`[^a-z0-9_]+`)

func DimensionIdentifier(worldName string, taken map[string]bool) string {
	name := notIdentifier.ReplaceAllString(strings.ToLower(text.Clean(worldName)), "_")
	name = strings.Trim(name, "_")
	if name == "" || name == "overworld" || name == "nether" || name == "the_end" {
		name = "world_" + name
		name = strings.TrimSuffix(name, "_")
	}
	id := "craftbox:" + name
	for i := 2; taken[id]; i++ {
		id = fmt.Sprintf("craftbox:%s_%d", name, i)
	}
	return id
}

var DefaultDimensionRange = cube.Range{-64, 319}

func TargetRange(src cube.Range) cube.Range {
	if src.Min() >= DefaultDimensionRange.Min() && src.Max() <= DefaultDimensionRange.Max() {
		return DefaultDimensionRange
	}
	return src
}

type MergeStats struct {
	Chunks       int
	Entities     int
	Maps         int
	RenamedIDs   int
	OtherSkipped int
}

var chunkTags = func() [256]bool {
	var t [256]bool
	for _, c := range ",/+-12345670689:;<=>?@ABv|" {
		t[c] = true
	}
	return t
}()

const (
	keyEntityIDs = "digp"
	keyActor     = "actorprefix"
)

func openLevelDB(folder string, readOnly bool) (*leveldb.DB, error) {
	return leveldb.OpenFile(filepath.Join(folder, "db"), &opt.Options{
		Compression: opt.DefaultCompression,
		ReadOnly:    readOnly,
	})
}

func chunkKeyDimension(k []byte) (pos []byte, dim int32, rest []byte, ok bool) {
	switch len(k) {
	case 9, 10:
		if chunkTags[k[8]] {
			return k[:8], 0, k[8:], true
		}
	case 13, 14:
		if chunkTags[k[12]] {
			return k[:8], int32(binary.LittleEndian.Uint32(k[8:12])), k[12:], true
		}
	}
	return nil, 0, nil, false
}

func dimensionKey(pos []byte, dim int32, rest []byte) []byte {
	k := make([]byte, 0, 12+len(rest))
	k = append(k, pos...)
	k = binary.LittleEndian.AppendUint32(k, uint32(dim))
	return append(k, rest...)
}

func MergeWorld(dstFolder, srcFolder string, srcDim int, srcRange cube.Range, dim MergedDimension) (MergeStats, error) {
	var stats MergeStats
	src, err := openLevelDB(srcFolder, true)
	if err != nil {
		return stats, fmt.Errorf("open %s: %w", srcFolder, err)
	}
	defer src.Close()
	dst, err := openLevelDB(dstFolder, false)
	if err != nil {
		return stats, fmt.Errorf("open %s: %w", dstFolder, err)
	}
	defer dst.Close()

	usedIDs := make(map[int64]bool)
	it := dst.NewIterator(nil, nil)
	for ok := it.Seek([]byte(keyActor)); ok && bytes.HasPrefix(it.Key(), []byte(keyActor)); ok = it.Next() {
		if k := it.Key(); len(k) == len(keyActor)+8 {
			usedIDs[int64(binary.LittleEndian.Uint64(k[len(keyActor):]))] = true
		}
	}
	it.Release()
	if err := it.Error(); err != nil {
		return stats, err
	}

	newIDs := make(map[int64]int64)
	nextID := int64(dim.ID) << 40
	it = src.NewIterator(nil, nil)
	var srcIDs []int64
	for ok := it.Seek([]byte(keyActor)); ok && bytes.HasPrefix(it.Key(), []byte(keyActor)); ok = it.Next() {
		if k := it.Key(); len(k) == len(keyActor)+8 {
			srcIDs = append(srcIDs, int64(binary.LittleEndian.Uint64(k[len(keyActor):])))
		}
	}
	it.Release()
	srcSet := make(map[int64]bool, len(srcIDs))
	for _, id := range srcIDs {
		srcSet[id] = true
	}
	for _, id := range srcIDs {
		if !usedIDs[id] {
			continue
		}
		for usedIDs[nextID] || srcSet[nextID] {
			nextID++
		}
		newIDs[id] = nextID
		usedIDs[nextID] = true
		nextID++
	}
	stats.RenamedIDs = len(newIDs)
	remap := func(id int64) int64 {
		if n, ok := newIDs[id]; ok {
			return n
		}
		return id
	}

	batch := new(leveldb.Batch)
	flush := func() error {
		if batch.Len() == 0 {
			return nil
		}
		err := dst.Write(batch, nil)
		batch.Reset()
		return err
	}

	it = src.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		k, v := it.Key(), it.Value()
		switch {
		case bytes.HasPrefix(k, []byte(keyEntityIDs)) && (len(k) == len(keyEntityIDs)+8 || len(k) == len(keyEntityIDs)+12):
			pos := k[len(keyEntityIDs) : len(keyEntityIDs)+8]
			var kd int32
			if len(k) == len(keyEntityIDs)+12 {
				kd = int32(binary.LittleEndian.Uint32(k[len(keyEntityIDs)+8:]))
			}
			if int(kd) != srcDim {
				stats.OtherSkipped++
				continue
			}
			ids := make([]byte, 0, len(v))
			for i := 0; i+8 <= len(v); i += 8 {
				ids = binary.LittleEndian.AppendUint64(ids, uint64(remap(int64(binary.LittleEndian.Uint64(v[i:])))))
			}
			batch.Put(append([]byte(keyEntityIDs), dimensionKey(pos, int32(dim.ID), nil)...), ids)

		case bytes.HasPrefix(k, []byte(keyActor)) && len(k) == len(keyActor)+8:
			id := int64(binary.LittleEndian.Uint64(k[len(keyActor):]))
			if newID := remap(id); newID != id || len(newIDs) > 0 {
				var m map[string]any
				if err := nbt.UnmarshalEncoding(v, &m, nbt.LittleEndian); err != nil {
					return stats, fmt.Errorf("entity %d: %w", id, err)
				}
				m["UniqueID"] = newID
				remapLinks(m, remap)
				nv, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
				if err != nil {
					return stats, err
				}
				v = nv
				id = newID
			}
			batch.Put(binary.LittleEndian.AppendUint64([]byte(keyActor), uint64(id)), v)
			stats.Entities++

		case bytes.HasPrefix(k, []byte("map_")):
			if has, _ := dst.Has(k, nil); !has {
				batch.Put(bytes.Clone(k), bytes.Clone(v))
				stats.Maps++
			}

		default:
			pos, kd, rest, ok := chunkKeyDimension(k)
			if !ok || int(kd) != srcDim {
				stats.OtherSkipped++
				continue
			}
			if rest[0] == ',' {
				stats.Chunks++
			}
			if rest[0] == '+' && srcRange != dim.Range {
				nv, err := moveBiomes(v, srcRange, dim.Range)
				if err != nil {
					return stats, fmt.Errorf("biomes of chunk %v: %w", pos, err)
				}
				v = nv
			}
			batch.Put(dimensionKey(pos, int32(dim.ID), rest), bytes.Clone(v))
		}
		if batch.Len() >= 512 {
			if err := flush(); err != nil {
				return stats, err
			}
		}
	}
	if err := it.Error(); err != nil {
		return stats, err
	}
	if err := flush(); err != nil {
		return stats, err
	}

	if err := addDimensionToTable(dst, dim); err != nil {
		return stats, err
	}
	return stats, nil
}

func moveBiomes(data []byte, from, to cube.Range) ([]byte, error) {
	if len(data) <= 512 {
		return data, nil
	}
	br := world.DefaultBlockRegistry
	src, err := chunk.DiskDecode(br, chunk.SerialisedData{Biomes: data[512:]}, from)
	if err != nil {
		return nil, err
	}
	dst := chunk.New(br, to)
	for y := to.Min(); y <= to.Max(); y++ {
		sy := int16(min(max(y, from.Min()), from.Max()))
		for x := uint8(0); x < 16; x++ {
			for z := uint8(0); z < 16; z++ {
				dst.SetBiome(x, int16(y), z, src.Biome(x, sy, z))
			}
		}
	}
	return append(make([]byte, 512), chunk.EncodeBiomes(dst, chunk.DiskEncoding)...), nil
}

func remapLinks(m map[string]any, remap func(int64) int64) {
	for _, key := range []string{"OwnerNew", "TargetID", "LeasherID", "LoveCause", "LastHurtBy"} {
		if id, ok := m[key].(int64); ok && id != -1 {
			m[key] = remap(id)
		}
	}

	links, ok := m["LinksTag"].([]any)
	if !ok {
		return
	}
	for _, l := range links {
		if lm, ok := l.(map[string]any); ok {
			if id, ok := lm["entityID"].(int64); ok {
				lm["entityID"] = remap(id)
			}
		}
	}
}

func addDimensionToTable(db *leveldb.DB, dim MergedDimension) error {
	table := map[string]any{}
	if v, err := db.Get([]byte(keyDimensionNameIDTable), nil); err == nil {
		if err := nbt.UnmarshalEncoding(v, &table, nbt.LittleEndian); err != nil {
			return fmt.Errorf("%s: %w", keyDimensionNameIDTable, err)
		}
	} else if !errors.Is(err, leveldb.ErrNotFound) {
		return err
	}
	entries, _ := table["entries"].(map[string]any)
	if entries == nil {
		entries = map[string]any{}
	}
	entries[dim.Identifier] = int32(dim.ID)
	table["entries"] = entries
	v, err := nbt.MarshalEncoding(table, nbt.LittleEndian)
	if err != nil {
		return err
	}
	if err := db.Put([]byte(keyDimensionNameIDTable), v, nil); err != nil {
		return err
	}

	limbo, err := nbt.MarshalEncoding(map[string]any{
		"data": map[string]any{"LimboEntities": []any{}},
	}, nbt.LittleEndian)
	if err != nil {
		return err
	}
	return db.Put([]byte(dim.Identifier), limbo, nil)
}

func ReadDimensionTable(folder string) (map[string]int, error) {
	db, err := openLevelDB(folder, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	v, err := db.Get([]byte(keyDimensionNameIDTable), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return map[string]int{}, nil
	}
	if err != nil {
		return nil, err
	}
	var table struct {
		Entries map[string]int32 `nbt:"entries"`
	}
	if err := nbt.UnmarshalEncoding(v, &table, nbt.LittleEndian); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(table.Entries))
	for k, id := range table.Entries {
		out[k] = int(id)
	}
	return out, nil
}

func MergePacks(dstFolder, srcFolder string) error {
	for _, dir := range []string{"behavior_packs", "resource_packs"} {
		entries, err := os.ReadDir(filepath.Join(srcFolder, dir))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dst := filepath.Join(dstFolder, dir, e.Name())
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
			err := utils.CopyFS(os.DirFS(filepath.Join(srcFolder, dir, e.Name())), utils.OSWriter{Base: dst})
			if err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"world_behavior_packs.json", "world_resource_packs.json"} {
		add, err := readPacksJSON(filepath.Join(srcFolder, name))
		if err != nil {
			return err
		}
		if err := AddWorldPacks(dstFolder, name, add); err != nil {
			return err
		}
	}
	return nil
}

func readPacksJSON(file string) ([]resourcePackDependency, error) {
	b, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var deps []resourcePackDependency
	if err := json.Unmarshal(b, &deps); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return deps, nil
}

func AddWorldPacks(folder, name string, add []resourcePackDependency) error {
	if len(add) == 0 {
		return nil
	}
	file := filepath.Join(folder, name)
	deps, err := readPacksJSON(file)
	if err != nil {
		return err
	}
	for _, a := range add {
		found := false
		for i := range deps {
			if strings.EqualFold(deps[i].UUID, a.UUID) {
				deps[i].Version = a.Version
				found = true
			}
		}
		if !found {
			deps = append(deps, a)
		}
	}
	b, err := json.Marshal(deps)
	if err != nil {
		return err
	}
	return os.WriteFile(file, b, 0o644)
}
