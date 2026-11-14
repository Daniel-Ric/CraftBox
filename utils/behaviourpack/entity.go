package behaviourpack

import (
	_ "embed"
	"encoding/json"
	"maps"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type EntityDescription struct {
	Identifier   string `json:"identifier"`
	Spawnable    bool   `json:"is_spawnable"`
	Summonable   bool   `json:"is_summonable"`
	Experimental bool   `json:"is_experimental"`

	Properties map[string]EntityPropertyJson `json:"properties,omitempty"`
}

type MinecraftEntity struct {
	Description     *EntityDescription `json:"description"`
	ComponentGroups map[string]any     `json:"component_groups,omitempty"`
	Components      map[string]any     `json:"components,omitempty"`
	Events          map[string]any     `json:"events,omitempty"`
}

type entityBehaviour struct {
	FormatVersion   string           `json:"format_version"`
	MinecraftEntity *MinecraftEntity `json:"minecraft:entity"`
}

type EntityPropertyJson struct {
	Type       string `json:"type"`
	Values     any    `json:"values,omitempty"`
	Range      []any  `json:"range,omitempty"`
	Default    any    `json:"default,omitempty"`
	ClientSync bool   `json:"client_sync"`
}

func makeEntityProperties(props []entity.EntityPropertyDef) map[string]EntityPropertyJson {
	if len(props) == 0 {
		return nil
	}
	properties := make(map[string]EntityPropertyJson)
	for _, v := range props {
		var prop EntityPropertyJson
		prop.ClientSync = true
		switch v.Type {
		case entity.PropertyTypeInt:
			prop.Type = "int"
			prop.Range = []any{int(v.Min), int(v.Max)}
			prop.Default = int(v.Min)
		case entity.PropertyTypeFloat:
			prop.Type = "float"
			prop.Range = []any{v.Min, v.Max}
			prop.Default = v.Min
		case entity.PropertyTypeBool:
			prop.Type = "bool"
			prop.Default = false
		case entity.PropertyTypeEnum:
			prop.Type = "enum"
			prop.Values = v.Enum
			prop.Default = v.Enum[0]
		}
		properties[v.Name] = prop
	}
	return properties
}

func (bp *Pack) AddEntity(EntityType string, attr []protocol.AttributeValue, meta protocol.EntityMetadata, props []entity.EntityPropertyDef) {
	ns, _ := splitNamespace(EntityType)
	if ns == "minecraft" {
		return
	}

	entry, ok := bp.entities[EntityType]
	if !ok {
		entry = &entityBehaviour{
			FormatVersion: "1.26.30",
			MinecraftEntity: &MinecraftEntity{
				Description: &EntityDescription{
					Identifier:   EntityType,
					Spawnable:    true,
					Summonable:   true,
					Experimental: true,
				},
				ComponentGroups: make(map[string]any),
				Components:      make(map[string]any),
				Events:          nil,
			},
		}
	}

	for _, av := range attr {
		m := map[string]float32{
			"value": av.Value,
			"min":   av.Min,
		}
		if av.Max > 0 && av.Max < 0xffffff {
			m["max"] = av.Max
		}
		entry.MinecraftEntity.Components[av.Name] = m
	}

	if scale, ok := meta[protocol.EntityDataKeyScale].(float32); ok {
		entry.MinecraftEntity.Components["minecraft:scale"] = map[string]any{
			"value": scale,
		}
	}

	width, widthOk := meta[protocol.EntityDataKeyWidth].(float32)
	height, heightOk := meta[protocol.EntityDataKeyHeight].(float32)
	if widthOk || heightOk {
		entry.MinecraftEntity.Components["minecraft:collision_box"] = map[string]any{
			"width":  width,
			"height": height,
		}
	}

	if ShowNameTag, ok := meta[protocol.EntityDataKeyAlwaysShowNameTag]; ok {
		if entity.MetadataNonZero(ShowNameTag) {
			entry.MinecraftEntity.Components["minecraft:nameable"] = map[string]any{
				"always_show": true,
			}
		}
	}

	if _, ok := meta[protocol.EntityDataKeyNameRawText].(string); ok {
		entry.MinecraftEntity.Components["minecraft:npc"] = map[string]any{}
	}

	if _, ok := meta[protocol.EntityDataKeyFlags]; ok {
		AlwaysShowName := meta.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAlwaysShowName)
		if AlwaysShowName {
			entry.MinecraftEntity.Components["minecraft:nameable"] = map[string]any{
				"always_show": true,
			}
		}
	}

	gravity, collision := true, true
	if _, ok := meta[protocol.EntityDataKeyFlags]; ok {
		gravity = meta.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasGravity)
		collision = meta.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasCollision)
	}
	entry.MinecraftEntity.Components["minecraft:physics"] = map[string]any{"has_collision": collision, "has_gravity": gravity}
	if !gravity {
		entry.MinecraftEntity.Components["minecraft:knockback_resistance"] = map[string]float32{"value": 1, "max": 1}
	}

	entry.MinecraftEntity.Components["minecraft:is_stackable"] = map[string]any{}
	entry.MinecraftEntity.Components["minecraft:push_through"] = 0

	entry.MinecraftEntity.Description.Properties = makeEntityProperties(props)
	bp.entities[EntityType] = entry
}

//go:embed player.json
var playerJson []byte

func (bp *Pack) AddComponents(identifier string, components map[string]any) {
	entry, ok := bp.entities[identifier]
	if !ok {
		return
	}
	copyEntry := *entry
	copyEntity := *entry.MinecraftEntity
	copyEntity.Components = maps.Clone(entry.MinecraftEntity.Components)
	if copyEntity.Components == nil {
		copyEntity.Components = map[string]any{}
	}
	maps.Copy(copyEntity.Components, components)
	copyEntry.MinecraftEntity = &copyEntity
	bp.entities[identifier] = &copyEntry
}

func (bp *Pack) AddEntityDefinition(identifier, formatVersion string, components, groups, events map[string]any) {
	bp.entities[identifier] = &entityBehaviour{FormatVersion: formatVersion, MinecraftEntity: &MinecraftEntity{
		Description: &EntityDescription{Identifier: identifier, Summonable: true}, Components: maps.Clone(components), ComponentGroups: maps.Clone(groups), Events: maps.Clone(events),
	}}
}

func (bp *Pack) SetEntityProperties(identifier string, props []entity.EntityPropertyDef) {
	ns, _ := splitNamespace(identifier)
	if ns == "minecraft" || identifier == "" {
		return
	}
	entry, ok := bp.entities[identifier]
	if !ok {
		bp.AddEntity(identifier, nil, nil, props)
		return
	}
	if props == nil {
		return
	}
	copyEntry := *entry
	copyEntity := *entry.MinecraftEntity
	copyDescription := *entry.MinecraftEntity.Description
	copyDescription.Properties = makeEntityProperties(props)
	copyEntity.Description = &copyDescription
	copyEntry.MinecraftEntity = &copyEntity
	bp.entities[identifier] = &copyEntry
}

func (bp *Pack) SetPlayerProperties(props []entity.EntityPropertyDef) {
	var basePlayer entityBehaviour
	err := json.Unmarshal(playerJson, &basePlayer)
	if err != nil {
		panic(err)
	}
	basePlayer.MinecraftEntity.Description.Properties = makeEntityProperties(props)
	bp.entities["minecraft:player"] = &basePlayer
}
