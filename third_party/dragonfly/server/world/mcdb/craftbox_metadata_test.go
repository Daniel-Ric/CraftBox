package mcdb

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

func TestReadOnlyOpenCloseDoesNotRewriteWorldMetadata(t *testing.T) {
	folder := t.TempDir()
	db, err := (Config{}).Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, "level.dat")
	before, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	db, err = (Config{LDBOptions: &opt.Options{ReadOnly: true}}).Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	db.ldat.LevelName = "must not be written"
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	next, _ := os.Stat(path)
	if !bytes.Equal(before, after) || !info.ModTime().Equal(next.ModTime()) {
		t.Fatal("read-only import changed source metadata")
	}
}
func TestUnknownLevelTagsSurviveReadModifyWrite(t *testing.T) {
	folder := t.TempDir()
	db, err := (Config{}).Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	path := filepath.Join(folder, "level.dat")
	file, _ := leveldat.ReadFile(path)
	var metadata map[string]any
	file.Unmarshal(&metadata)
	metadata["FutureRoot"] = map[string]any{"preserve": "root"}
	metadata["abilities"].(map[string]any)["FutureAbility"] = map[string]any{"value": int32(9)}
	file.Marshal(metadata)
	file.WriteFile(path)
	db, err = (Config{}).Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	db.ldat.LevelName = "Changed"
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	file, _ = leveldat.ReadFile(path)
	file.Unmarshal(&metadata)
	if metadata["LevelName"] != "Changed" || metadata["FutureRoot"].(map[string]any)["preserve"] != "root" || metadata["abilities"].(map[string]any)["FutureAbility"].(map[string]any)["value"] != int32(9) {
		t.Fatal("unknown NBT data lost")
	}
}
