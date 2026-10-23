package mcdb

import (
	"errors"
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/goleveldb/leveldb"
)

func TestCapturedEntitiesRoundTrip(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pos := world.ChunkPos{2, -3}
	entities := []chunk.Entity{{ID: 1234, Data: map[string]any{"identifier": "minecraft:pig", "UniqueID": int64(1234)}}}
	if err := db.StoreEntities(pos, world.Overworld, entities); err != nil {
		t.Fatal(err)
	}
	got, err := db.entities(dbKey{pos: pos, dim: world.Overworld})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 1234 || got[0].Data["identifier"] != "minecraft:pig" {
		t.Fatalf("entities = %#v", got)
	}
	if err := db.StoreEntities(pos, world.Overworld, nil); err != nil {
		t.Fatal(err)
	}
	got, err = db.entities(dbKey{pos: pos, dim: world.Overworld})
	if (err != nil && !errors.Is(err, leveldb.ErrNotFound)) || len(got) != 0 {
		t.Fatalf("deleted entities = %#v, %v", got, err)
	}
}
