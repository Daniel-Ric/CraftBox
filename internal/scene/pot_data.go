package scene

import (
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/goleveldb/leveldb"
)

type potPlant struct {
	Position             cube.Pos
	Name                 string
	Sign                 map[string]any `json:",omitempty"`
	Frame                map[string]any `json:",omitempty"`
	Visual               map[string]any `json:",omitempty"`
	FrameRotationDegrees *float32       `json:",omitempty"`
}

func plantKey(p world.ChunkPos) []byte { return append(chunkKey(p), 'p') }

func encodePotPlants(entities map[cube.Pos]map[string]any) []byte {
	plants := []potPlant{}
	for pos, data := range entities {
		if data["id"] == "Bed" || data["id"] == "Banner" || data["id"] == "Chest" || data["id"] == "EnderChest" || data["id"] == "PistonArm" || data["id"] == "ShulkerBox" || data["id"] == "Lectern" {
			visual := map[string]any{}
			for _, key := range []string{"id", "color", "Base", "Patterns", "pairx", "pairz", "pairlead", "State", "NewState", "Progress", "LastProgress", "facing", "hasBook"} {
				if value, ok := data[key]; ok {
					visual[key] = value
				}
			}
			if data["id"] == "Lectern" {
				visual["hasBook"] = lecternHasBook(data)
			}
			plants = append(plants, potPlant{Position: pos, Visual: visual})
			continue
		}
		if data["id"] == "ItemFrame" || data["id"] == "GlowItemFrame" {
			rotation := frameRotationDegrees(data)
			plants = append(plants, potPlant{Position: pos, Frame: data, FrameRotationDegrees: &rotation})
			continue
		}
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
	s.decorativeEntities = map[cube.Pos]map[string]any{}
	s.decorativeModels = map[string]*blockModel{}
	s.frames = map[cube.Pos]map[string]any{}
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
		if plant.Visual != nil {
			s.decorativeEntities[plant.Position] = plant.Visual
		}
		if plant.Frame != nil {
			if plant.FrameRotationDegrees != nil {
				plant.Frame[frameRotationKey] = *plant.FrameRotationDegrees
			}
			s.frames[plant.Position] = plant.Frame
		}
		s.potPlants[plant.Position] = plant.Name
		if plant.Sign != nil {
			s.signs[plant.Position] = plant.Sign
		}
	}
	return nil
}
