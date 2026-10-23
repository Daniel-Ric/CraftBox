package worlds

import (
	"encoding/binary"
	"errors"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"path/filepath"
	"testing"
)

func testChunkKey(x, z, dim int32, suffix ...byte) []byte {
	key := binary.LittleEndian.AppendUint32(nil, uint32(x))
	key = binary.LittleEndian.AppendUint32(key, uint32(z))
	if dim != 0 {
		key = binary.LittleEndian.AppendUint32(key, uint32(dim))
	}
	return append(key, suffix...)
}

func TestRegionFilterKeepsColumnsInventoriesAndReferencedActors(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "db")
	db, err := leveldb.OpenFile(folder, nil)
	if err != nil {
		t.Fatal(err)
	}
	put := func(k, v []byte) {
		t.Helper()
		if err := db.Put(k, v, nil); err != nil {
			t.Fatal(err)
		}
	}
	inventory := []byte("chest NBT with stored items")
	for _, tag := range []byte{',', '/', '1', '+'} {
		put(testChunkKey(-1, 2, 0, tag), inventory)
		put(testChunkKey(9, 2, 0, tag), inventory)
		put(testChunkKey(-1, 2, 1, tag), inventory)
	}
	for _, tc := range []struct {
		x  int32
		id int64
	}{{-1, 11}, {9, 22}} {
		put(append([]byte("digp"), testChunkKey(tc.x, 2, 0)...), binary.LittleEndian.AppendUint64(nil, uint64(tc.id)))
		value, err := nbt.MarshalEncoding(map[string]any{"UniqueID": tc.id, "Inventory": []map[string]any{{"Name": "minecraft:diamond", "Count": uint8(2)}}, "LinksTag": []map[string]any{{"entityID": int64(22), "linkID": int32(0)}}}, nbt.LittleEndian)
		if err != nil {
			t.Fatal(err)
		}
		put(binary.LittleEndian.AppendUint64([]byte("actorprefix"), uint64(tc.id)), value)
	}
	put([]byte("~local_player"), []byte("player inventory"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	count, err := filterRegionDatabase(folder, func(x, z, dim int32) bool { return x == -1 && z == 2 && dim == 0 })
	if err != nil {
		t.Fatal(err)
	}
	db, err = leveldb.OpenFile(folder, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Get(testChunkKey(9, 2, 0, ','), nil); !errors.Is(err, leveldb.ErrNotFound) {
		t.Fatal("outside terrain retained")
	}
	if _, err := db.Get(testChunkKey(-1, 2, 1, ','), nil); !errors.Is(err, leveldb.ErrNotFound) {
		t.Fatal("other dimension retained")
	}
	if value, err := db.Get(testChunkKey(-1, 2, 0, '1'), nil); err != nil || string(value) != string(inventory) {
		t.Fatal("block inventory changed")
	}
	if _, err := db.Get(binary.LittleEndian.AppendUint64([]byte("actorprefix"), 22), nil); !errors.Is(err, leveldb.ErrNotFound) {
		t.Fatal("outside actor retained")
	}
	if _, err := db.Get(binary.LittleEndian.AppendUint64([]byte("actorprefix"), 11), nil); err != nil {
		t.Fatal("selected actor lost")
	}
	retained, err := db.Get(binary.LittleEndian.AppendUint64([]byte("actorprefix"), 11), nil)
	if err != nil {
		t.Fatal(err)
	}
	var actor map[string]any
	if err := nbt.UnmarshalEncoding(retained, &actor, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if _, exists := actor["LinksTag"]; exists {
		t.Fatal("actor links point into another world")
	}
	if _, exists := actor["Inventory"]; !exists {
		t.Fatal("actor inventory lost during link filtering")
	}
	if _, err := db.Get([]byte("~local_player"), nil); err != nil {
		t.Fatal("player inventory lost")
	}
	if count != 1 {
		t.Fatalf("actor count %d", count)
	}
}
