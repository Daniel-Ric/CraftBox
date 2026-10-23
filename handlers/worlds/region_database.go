package worlds

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func regionChunkIndex(key []byte) (x, z, dimension int32, ok bool) {
	if len(key) != 8 && len(key) != 12 {
		return
	}
	x = int32(binary.LittleEndian.Uint32(key))
	z = int32(binary.LittleEndian.Uint32(key[4:]))
	if len(key) == 12 {
		dimension = int32(binary.LittleEndian.Uint32(key[8:]))
	}
	ok = true
	return
}

func regionChunkRecord(key []byte) (x, z, dimension int32, ok bool) {
	length := len(key)
	if length != 9 && length != 10 && length != 13 && length != 14 {
		return
	}
	indexLength := 8
	if length >= 13 {
		indexLength = 12
	}
	tag := key[indexLength]
	if !(tag >= 0x2b && tag <= 0x40) && tag != 'v' && tag != '|' {
		return
	}
	if length == indexLength+2 && tag != '/' {
		return
	}
	return regionChunkIndex(key[:indexLength])
}

func filterRegionDatabase(folder string, keep func(x, z, dimension int32) bool) (int, error) {
	db, err := leveldb.OpenFile(folder, nil)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	ids := make(map[int64]bool)
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		if !bytes.HasPrefix(key, []byte("digp")) {
			continue
		}
		x, z, dim, ok := regionChunkIndex(key[4:])
		if !ok {
			continue
		}
		value := iter.Value()
		if len(value)%8 != 0 {
			iter.Release()
			return 0, fmt.Errorf("invalid actor index")
		}
		if keep(x, z, dim) {
			for offset := 0; offset < len(value); offset += 8 {
				ids[int64(binary.LittleEndian.Uint64(value[offset:]))] = true
			}
		}
	}
	err = iter.Error()
	iter.Release()
	if err != nil {
		return 0, err
	}
	batch := leveldb.MakeBatch(0)
	count := 0
	iter = db.NewIterator(nil, nil)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		if x, z, dim, ok := regionChunkRecord(key); ok {
			if !keep(x, z, dim) {
				batch.Delete(key)
			}
			continue
		}
		if bytes.HasPrefix(key, []byte("digp")) {
			if x, z, dim, ok := regionChunkIndex(key[4:]); ok && !keep(x, z, dim) {
				batch.Delete(key)
			}
			continue
		}
		if bytes.HasPrefix(key, []byte("actorprefix")) && len(key) == 19 {
			id := int64(binary.LittleEndian.Uint64(key[11:]))
			if !ids[id] {
				batch.Delete(key)
				continue
			}
			var data map[string]any
			if err := nbt.UnmarshalEncoding(value, &data, nbt.LittleEndian); err != nil {
				iter.Release()
				return 0, err
			}

			if links, ok := data["LinksTag"].([]any); ok {
				selected := make([]any, 0, len(links))
				for _, link := range links {
					if row, ok := link.(map[string]any); ok {
						if linked, ok := row["entityID"].(int64); ok && ids[linked] {
							selected = append(selected, row)
						}
					}
				}
				if len(selected) == 0 {
					delete(data, "LinksTag")
				} else {
					data["LinksTag"] = selected
				}
				encoded, err := nbt.MarshalEncoding(data, nbt.LittleEndian)
				if err != nil {
					iter.Release()
					return 0, err
				}
				batch.Put(key, encoded)
			}
			count++
		}
	}
	err = iter.Error()
	iter.Release()
	if err != nil {
		return 0, err
	}
	if err = db.Write(batch, nil); err != nil {
		return 0, err
	}
	return count, nil
}
