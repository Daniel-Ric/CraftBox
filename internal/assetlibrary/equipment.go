package assetlibrary

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type equipmentRecord struct {
	PlayerUUID string            `json:"playerUUID"`
	PlayerName string            `json:"playerName"`
	Slots      map[string]string `json:"slots"`
	Updated    string            `json:"updated"`
}

func (l *Library) equipmentRecords() ([]equipmentRecord, error) {
	raw, err := readLimit(filepath.Join(l.Root, "equipment.json"), 8<<20)
	if os.IsNotExist(err) {
		return []equipmentRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	var records []equipmentRecord
	err = json.Unmarshal(raw, &records)
	return records, err
}
func (l *Library) SetCapturedEquipment(playerUUID, playerName string, slots map[string]string) error {
	if playerUUID == "" && playerName == "" {
		return errors.New("Equipment has no player identity")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	records, err := l.equipmentRecords()
	if err != nil {
		return err
	}
	index := -1
	for i, record := range records {
		if record.PlayerUUID == playerUUID && record.PlayerName == playerName {
			index = i
			break
		}
	}
	if index < 0 {
		records = append(records, equipmentRecord{PlayerUUID: playerUUID, PlayerName: playerName, Slots: map[string]string{}})
		index = len(records) - 1
	}
	if records[index].Slots == nil {
		records[index].Slots = map[string]string{}
	}
	for slot, item := range slots {
		if len(item) > 256 {
			continue
		}
		switch slot {
		case "head", "chest", "legs", "feet", "body":
			records[index].Slots[slot] = item
		}
	}
	records[index].Updated = time.Now().UTC().Format(time.RFC3339Nano)
	sort.Slice(records, func(i, j int) bool { return records[i].Updated > records[j].Updated })
	if len(records) > 2000 {
		records = records[:2000]
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(l.Root, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(l.Root, "equipment-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(l.Root, "equipment.json"))
}
func (l *Library) capturedEquipment(metadata map[string]any) map[string]string {
	records, err := l.equipmentRecords()
	if err != nil {
		return nil
	}
	return equipmentForMetadata(records, metadata)
}
func equipmentForMetadata(records []equipmentRecord, metadata map[string]any) map[string]string {
	playerUUID, _ := metadata["playerUUID"].(string)
	name, _ := metadata["player"].(string)
	for _, record := range records {
		if playerUUID != "" && record.PlayerUUID == playerUUID || playerUUID == "" && name != "" && strings.EqualFold(record.PlayerName, name) {
			return record.Slots
		}
	}
	return nil
}
