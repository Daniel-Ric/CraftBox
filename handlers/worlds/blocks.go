package worlds

import (
	"errors"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/utils/blocktraits"
	"github.com/Daniel-Ric/CraftBox/utils/nbtconv"
	"github.com/sirupsen/logrus"
	"strings"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/zaataylor/cartesian/cartesian"
)

var traitLookup = map[string][]any{
	"minecraft:cardinal_direction": {
		"north", "east", "south", "west",
	},
	"minecraft:facing_direction": {
		"north", "east", "south", "west", "down", "up",
	},
	"minecraft:corner_and_cardinal_direction": {
		"inner_left", "inner_right", "outer_left", "outer_right", "none",
	},
	"minecraft:sixteen_way_rotation": {
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16",
	},
	"minecraft:vertical_half": {
		"top", "bottom",
	},
	"minecraft:block_face": {
		"north", "east", "south", "west", "down", "up",
	},
}

func splitNamespace(identifier string) (ns, name string) {
	sp := strings.SplitN(identifier, ":", 2)
	if len(sp) == 1 {
		return "", identifier
	}
	return sp[0], sp[1]
}

var errMalformedBlock = errors.New("invalid block definition")

func AddCustomBlocks(reg world.BlockRegistry, entries []protocol.BlockEntry) error {
	registered := map[string]bool{}
	for _, entry := range entries {
		if err := addCustomBlock(reg, entry, registered); err != nil {
			if !errors.Is(err, errMalformedBlock) {
				return err
			}
			logrus.Warnf("Skipping block states %s: %v", entry.Name, err)
		}
	}
	return nil
}
func addCustomBlock(reg world.BlockRegistry, entry protocol.BlockEntry, registered map[string]bool) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("%w: %v", errMalformedBlock, value)
		}
	}()

	ns, _ := splitNamespace(entry.Name)
	if ns == "minecraft" {
		return nil
	}

	var propertyNames []string
	var propertyValues []any
	seenProperties := make(map[string]struct{})
	addProperty := func(name string, values any) {

		if _, exists := seenProperties[name]; exists {
			return
		}
		seenProperties[name] = struct{}{}
		propertyNames = append(propertyNames, name)
		propertyValues = append(propertyValues, values)
	}

	props, ok := entry.Properties["properties"].([]any)
	if ok {
		for _, v := range props {
			v := v.(map[string]any)
			name := v["name"].(string)
			enum := v["enum"]
			addProperty(name, enum)
		}
	}

	traits, ok := entry.Properties["traits"].([]any)
	if ok {
		for _, trait := range traits {
			trait := trait.(map[string]any)
			name := trait["name"].(string)
			if name == "minecraft:multi_block" {
				if blocktraits.MultiBlockEnabled(trait) {
					values := make([]int32, blocktraits.MultiBlockParts(trait))
					for i := range values {
						values[i] = int32(i)
					}
					addProperty("minecraft:multi_block_part", values)
				}
				continue
			}
			switch enabled_states := trait["enabled_states"].(type) {
			case map[string]any:
				for k, enabled := range enabled_states {
					if !strings.ContainsRune(k, ':') {
						k = "minecraft:" + k
					}
					if !nbtconv.Flag(enabled) {
						continue
					}
					v, ok := traitLookup[k]
					if !ok {
						return fmt.Errorf("unresolved trait %s", k)
					}

					addProperty(k, v)
				}
			case int32:
				if name == "minecraft:connection" {
					var b = []bool{false, true}
					for _, direction := range []string{"north", "south", "west", "east"} {
						addProperty("minecraft:connection_"+direction, b)
					}
				} else {
					return fmt.Errorf("unresolved trait %s", name)
				}
			}
		}
	}

	permutations := cartesian.NewCartesianProduct(propertyValues).Values()

	for _, values := range permutations {
		m := make(map[string]any)
		for i, value := range values {
			name := propertyNames[i]
			m[name] = value
		}
		key := entry.Name + fmt.Sprint(m)
		if registered[key] {
			continue
		}
		registered[key] = true
		reg.RegisterBlockState(world.BlockState{
			Name:       entry.Name,
			Properties: m,
		})
	}
	return nil
}
