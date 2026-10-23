package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
)

type potPlant struct {
	Position cube.Pos
	Name     string
	Sign     map[string]any `json:",omitempty"`
}

func plantKey(p world.ChunkPos) []byte { return append(chunkKey(p), 'p') }

func encodePotPlants(entities map[cube.Pos]map[string]any) []byte {
	plants := []potPlant{}
	for pos, data := range entities {
		if data["id"] == "Sign" || data["id"] == "HangingSign" {
			plants = append(plants, potPlant{Position: pos, Sign: data})
			continue
		}
		if data["id"] != "FlowerPot" {
			continue
		}
		plant, _ := data["PlantBlock"].(map[string]any)
		name, _ := plant["name"].(string)
		if name != "" {
			plants = append(plants, potPlant{Position: pos, Name: name})
		}
	}
	raw, _ := json.Marshal(plants)
	return raw
}

func (s *Store) loadPotPlants(snapshot *leveldb.Snapshot, p world.ChunkPos) error {
	if raw, e := snapshot.Get(actorKey(p), nil); e == nil {
		if e = json.Unmarshal(raw, &s.visualActors); e != nil {
			return e
		}
	} else if e != leveldb.ErrNotFound {
		return e
	}
	raw, err := snapshot.Get(plantKey(p), nil)
	if err == leveldb.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	var plants []potPlant
	if err = json.Unmarshal(raw, &plants); err != nil {
		return err
	}
	s.potPlants = map[cube.Pos]string{}
	s.signs = map[cube.Pos]map[string]any{}
	for _, plant := range plants {
		s.potPlants[plant.Position] = plant.Name
		if plant.Sign != nil {
			s.signs[plant.Position] = plant.Sign
		}
	}
	return nil
}
