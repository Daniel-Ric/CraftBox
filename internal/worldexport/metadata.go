package worldexport

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func ReadMetadata(folder string) (map[string]any, int, error) {
	file, err := leveldat.ReadFile(filepath.Join(folder, "level.dat"))
	if err != nil {
		return nil, 0, err
	}
	var data map[string]any
	if err = file.Unmarshal(&data); err != nil {
		return nil, 0, err
	}
	return data, file.Ver(), nil
}

func UpdateMetadata(ctx context.Context, folder string, options Options) (result error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	data, version, err := ReadMetadata(folder)
	if err != nil {
		return err
	}
	if options.Profile != "" {
		baseline, err := profileBaseline(folder, data)
		if err != nil {
			return err
		}
		data = restoreFields(data, baseline, profileFields)
	}
	output, err := ApplySettings(data, options)
	if err != nil {
		return err
	}
	payload, err := nbt.MarshalEncoding(output, nbt.LittleEndian)
	if err != nil {
		return err
	}
	if len(payload) > 64<<20 {
		return fmt.Errorf("export level.dat metadata exceeds 64 MB")
	}
	encoded := make([]byte, 8, len(payload)+8)
	binary.LittleEndian.PutUint32(encoded, uint32(version))
	binary.LittleEndian.PutUint32(encoded[4:], uint32(len(payload)))
	encoded = append(encoded, payload...)
	preset, _ := profileSettings(options.Profile)
	gameMode := options.Settings.GameMode
	if gameMode == nil {
		gameMode = preset.GameMode
	}
	if options.Settings.Spawn != nil || gameMode != nil || options.Profile != "" {
		db, err := leveldb.OpenFile(filepath.Join(folder, "db"), &opt.Options{ErrorIfMissing: true})
		if err != nil {
			return err
		}
		updateErr := updatePlayer(db, folder, options.Profile, options.Settings.Spawn, gameMode)
		if err = errors.Join(updateErr, db.Close()); err != nil {
			return err
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = atomicWrite(filepath.Join(folder, "level.dat"), encoded); err != nil {
		return err
	}
	if options.Settings.Name != nil {
		if err = atomicWrite(filepath.Join(folder, "levelname.txt"), []byte(output["LevelName"].(string))); err != nil {
			return err
		}
	}
	if options.Profile != "" {
		return SaveOptions(folder, options)
	}
	return nil
}

func updatePlayer(db *leveldb.DB, folder, profile string, spawn *[3]int32, mode *int32) error {
	raw, err := db.Get([]byte("~local_player"), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var data map[string]any
	if err = nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); err != nil {
		return fmt.Errorf("read local player: %w", err)
	}
	if profile != "" {
		baseline, err := playerBaseline(folder, data)
		if err != nil {
			return err
		}
		data = restoreFields(data, baseline, playerProfileFields)
	}
	if spawn != nil {
		data["SpawnX"], data["SpawnY"], data["SpawnZ"] = spawn[0], spawn[1], spawn[2]
		data["Pos"] = []float32{float32(spawn[0]) + .5, float32(spawn[1]), float32(spawn[2]) + .5}
	}
	if mode != nil {
		data["PlayerGameMode"] = *mode
	}
	raw, err = nbt.MarshalEncoding(data, nbt.LittleEndian)
	if err != nil {
		return err
	}
	return db.Put([]byte("~local_player"), raw, &opt.WriteOptions{Sync: true})
}

func atomicWrite(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".craftbox-metadata-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	_, writeErr := file.Write(data)
	err = errors.Join(writeErr, file.Sync(), file.Close())
	if err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
