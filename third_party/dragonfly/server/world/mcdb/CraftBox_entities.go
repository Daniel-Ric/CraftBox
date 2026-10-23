package mcdb

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func (db *DB) LDB() *leveldb.DB { return db.ldb }

func (db *DB) LevelDat() *leveldat.Data { return db.ldat }

func (db *DB) SaveLocalPlayerData(data map[string]any) error {
	encoded, err := nbt.MarshalEncoding(data, nbt.LittleEndian)
	if err != nil {
		return fmt.Errorf("encode local player: %w", err)
	}
	return db.ldb.Put([]byte(keyLocalPlayer), encoded, nil)
}

func (db *DB) StoreEntities(pos world.ChunkPos, dim world.Dimension, entities []chunk.Entity) error {
	batch := leveldb.MakeBatch(1 + len(entities))
	db.storeEntities(batch, dbKey{pos: pos, dim: dim}, entities)
	return db.ldb.Write(batch, nil)
}
