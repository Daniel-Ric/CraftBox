package utils

import (
	"archive/zip"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWorldArchiveMetadataEditRetainsPackFiles(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "world")
	db, err := mcdb.Config{}.Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	db.SaveLocalPlayerData(map[string]any{"Pos": []float32{0, 70, 0}})
	db.Close()
	pack := filepath.Join(folder, "resource_packs", "custom")
	if err = os.MkdirAll(pack, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(pack, "texture.bin"), []byte("unchanged pixels"), 0600); err != nil {
		t.Fatal(err)
	}
	archive := folder + ".mcworld"
	if err = ReplaceWorldArchive(folder, archive); err != nil {
		t.Fatal(err)
	}
	spawn := [3]int32{1, 71, 1}
	if err = ApplyWorldMetadata(folder, "Renamed island", &spawn); err != nil {
		t.Fatal(err)
	}
	if err = ReplaceWorldArchive(folder, archive); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	data, err := fs.ReadFile(z, "resource_packs/custom/texture.bin")
	if err != nil || string(data) != "unchanged pixels" {
		t.Fatalf("pack lost: %q %v", data, err)
	}
	name, err := fs.ReadFile(z, "levelname.txt")
	if err != nil || string(name) != "Renamed island" {
		t.Fatalf("name not archived: %q %v", name, err)
	}
}

func TestWorldMetadataPreservesInventoryAndSetsSpawn(t *testing.T) {
	folder := t.TempDir()
	db, err := mcdb.Config{}.Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SaveLocalPlayerData(map[string]any{"Inventory": []map[string]any{{"Name": "minecraft:diamond"}}, "Pos": []float32{0, 70, 0}}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	spawn := [3]int32{-8, 80, 40}
	if err = ApplyWorldMetadata(folder, "Island", &spawn); err != nil {
		t.Fatal(err)
	}
	db, err = mcdb.Config{}.Open(folder)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.Settings().Name != "Island" || db.Settings().Spawn != (cube.Pos{-8, 80, 40}) {
		t.Fatal(db.Settings())
	}
	raw, err := db.LDB().Get([]byte("~local_player"), nil)
	if err != nil {
		t.Fatal(err)
	}
	var local map[string]any
	if err = nbt.UnmarshalEncoding(raw, &local, nbt.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if local["SpawnX"] != int32(-8) || len(local["Inventory"].([]any)) != 1 || local["Pos"].([]any)[0] != float32(-7.5) {
		t.Fatal(local)
	}
}
