package utils

import (
	"archive/zip"
	"errors"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"os"
	"path/filepath"
)


func ApplyWorldMetadata(folder, name string, spawn *[3]int32) (result error) {
	db, err := mcdb.Config{}.Open(folder)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, db.Close()) }()
	settings := db.Settings()
	if name != "" {
		settings.Name = name
	}
	if spawn != nil {
		raw, err := db.LDB().Get([]byte("~local_player"), nil)
		if err != nil {
			return err
		}
		var local map[string]any
		if err = nbt.UnmarshalEncoding(raw, &local, nbt.LittleEndian); err != nil {
			return err
		}
		local["Pos"] = []float32{float32(spawn[0]) + .5, float32(spawn[1]), float32(spawn[2]) + .5}
		local["SpawnX"], local["SpawnY"], local["SpawnZ"] = spawn[0], spawn[1], spawn[2]
		if err = db.SaveLocalPlayerData(local); err != nil {
			return err
		}
		settings.Spawn = cube.Pos{int(spawn[0]), int(spawn[1]), int(spawn[2])}
	}
	db.SaveSettings(settings)
	return nil
}


func ReplaceWorldArchive(folder, archive string) error {
	file, err := os.CreateTemp(filepath.Dir(archive), ".craft-box-world-*.tmp")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	writer := zip.NewWriter(file)
	err = errors.Join(writer.AddFS(os.DirFS(folder)), writer.Close(), file.Close())
	if err != nil {
		return err
	}
	return os.Rename(tmp, archive)
}
