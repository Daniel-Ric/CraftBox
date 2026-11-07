package mcdb

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb/util"
)

// columnRange recognises captures that stored every vertical section of a
// server-defined dimension. Infer the origin only when all subchunk records
// form a contiguous range and agree with the complete biome section count.
// Sparse vanilla columns and truncated columns must not acquire a new origin.
func (db *DB) columnRange(k dbKey, biomes []byte) (cube.Range, error) {
	fallback := k.dim.Range()
	if len(biomes) == 0 {
		if strict, ok := k.dim.(interface{ RequireExplicitRange() bool }); ok && strict.RequireExplicitRange() {
			return fallback, fmt.Errorf("custom dimension height cannot be inferred: biome section data is missing")
		}
		return fallback, nil
	}
	sections, err := chunk.DiskBiomeSections(biomes)
	if err != nil {
		return fallback, err
	}
	prefix := k.Sum(keySubChunkData)
	it := db.ldb.NewIterator(util.BytesPrefix(prefix), nil)
	defer it.Release()
	count, minimum, maximum := 0, 128, -129
	complete := true
	for it.Next() {
		key, data := it.Key(), it.Value()
		if len(key) != len(prefix)+1 {
			continue
		}
		y := int(int8(key[len(prefix)]))
		if len(data) < 3 || data[0] != 9 || int(int8(data[2])) != y {
			complete = false
		}
		minimum = min(minimum, y)
		maximum = max(maximum, y)
		count++
	}
	if err := it.Error(); err != nil {
		return fallback, err
	}
	if complete && count != 0 && count == sections && maximum-minimum+1 == count {
		return cube.Range{minimum * 16, (maximum+1)*16 - 1}, nil
	}
	if strict, ok := k.dim.(interface{ RequireExplicitRange() bool }); ok && strict.RequireExplicitRange() {
		return fallback, fmt.Errorf("custom dimension height cannot be inferred from incomplete subchunk and biome data")
	}
	return fallback, nil
}
