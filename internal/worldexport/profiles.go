package worldexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

var profileFields = []string{"GameType", "ForceGameType", "Difficulty", "Time", "rainLevel", "rainTime", "lightningLevel", "lightningTime", "dodaylightcycle", "doweathercycle", "domobspawning", "mobgriefing", "dofiretick", "keepinventory", "dotiledrops", "doentitydrops", "showcoordinates", "commandblocksenabled", "commandsEnabled", "cheatsEnabled", "randomtickspeed"}
var playerProfileFields = []string{"PlayerGameMode"}

func profileBaseline(folder string, current map[string]any) (map[string]any, error) {
	path := filepath.Join(folder, "craftbox-original-level.dat")
	file, err := leveldat.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw, readErr := os.ReadFile(filepath.Join(folder, "level.dat"))
		if readErr != nil {
			return nil, readErr
		}
		if err = atomicWrite(path, raw); err != nil {
			return nil, err
		}
		return current, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read original captured settings: %w", err)
	}
	var baseline map[string]any
	err = file.Unmarshal(&baseline)
	return baseline, err
}
func restoreFields(current, baseline map[string]any, fields []string) map[string]any {
	result := cloneNBT(current).(map[string]any)
	for _, key := range fields {
		if value, ok := baseline[key]; ok {
			result[key] = cloneNBT(value)
		} else {
			delete(result, key)
		}
	}
	return result
}
func playerBaseline(folder string, current map[string]any) (map[string]any, error) {
	path := filepath.Join(folder, "craftbox-original-player.nbt")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		baseline := map[string]any{}
		for _, key := range playerProfileFields {
			if v, ok := current[key]; ok {
				baseline[key] = v
			}
		}
		raw, err = nbt.MarshalEncoding(baseline, nbt.LittleEndian)
		if err != nil {
			return nil, err
		}
		if err = atomicWrite(path, raw); err != nil {
			return nil, err
		}
		return baseline, nil
	}
	if err != nil {
		return nil, err
	}
	var data map[string]any
	err = nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian)
	return data, err
}

func SaveOptions(folder string, options Options) error {
	if err := options.Validate(); err != nil {
		return err
	}

	options.Dimensions = nil
	options.Biomes = nil
	options.PlayerDimension = nil
	raw, err := json.MarshalIndent(struct {
		Version int     `json:"version"`
		Options Options `json:"options"`
	}{1, options}, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(folder, "craftbox-export-settings.json"), raw)
}
func ReadOptions(folder string) (Options, error) {
	raw, err := os.ReadFile(filepath.Join(folder, "craftbox-export-settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Options{Profile: "original"}, nil
	}
	if err != nil {
		return Options{}, err
	}
	var file struct {
		Version int     `json:"version"`
		Options Options `json:"options"`
	}
	if err = json.Unmarshal(raw, &file); err != nil {
		return Options{}, err
	}
	if file.Version != 1 {
		return Options{}, fmt.Errorf("unsupported world export settings version %d", file.Version)
	}
	return file.Options, file.Options.Validate()
}
