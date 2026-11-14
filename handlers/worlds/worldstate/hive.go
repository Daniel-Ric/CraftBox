package worldstate

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"regexp"
	"strconv"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const HiveTextEntity = "hive:text"

const HiveTextFormatVersion = "1.18.20"

var HiveTextComponents = map[string]any{
	"minecraft:collision_box": map[string]any{"height": 0.0, "width": 0.0},
	"minecraft:damage_sensor": map[string]any{"triggers": map[string]any{"deals_damage": false}},
	"minecraft:health":        map[string]any{"max": 1, "min": 0, "value": 1},
	"minecraft:is_stackable":  map[string]any{},
	"minecraft:nameable":      map[string]any{"always_show": true},
	"minecraft:push_through":  1,
	"minecraft:pushable":      map[string]any{"is_pushable": false, "is_pushable_by_piston": false},
	"minecraft:variant":       map[string]any{"value": 0},
	"minecraft:interact": map[string]any{"interactions": map[string]any{
		"interact_text": "",
		"on_interact":   map[string]any{"event": "hive:interact", "target": "self"},
	}},
}

var HiveTextGroups, HiveTextEvents = hiveTextGroupsEvents()

func hiveTextGroupsEvents() (groups, events map[string]any) {
	hitbox := func(w, h float64, pivot ...float64) map[string]any {
		return map[string]any{"hitboxes": []any{map[string]any{"width": w, "height": h, "pivot": pivot}}}
	}
	hitboxes := map[int]map[string]any{
		2: hitbox(2, 3.5, 0, -1.5, 0),
		3: hitbox(1.5, 2.5, 0, -1, 0),
		4: hitbox(1.5, 2.5, 0, -1, 0),
		8: hitbox(1.5, 2.5, 0, -1, 0.7),
	}
	groups = map[string]any{"hive:despawn": map[string]any{"minecraft:instant_despawn": map[string]any{}}}
	events = map[string]any{
		"hive:interact": map[string]any{},
		"hive:despawn":  map[string]any{"add": map[string]any{"component_groups": []string{"hive:despawn"}}},
	}
	for v := 1; v <= 50; v++ {
		g := map[string]any{"minecraft:variant": map[string]any{"value": v}}
		if hb, ok := hitboxes[v]; ok {
			g["minecraft:custom_hit_test"] = hb
		}
		if v == 19 || v == 20 {
			g["minecraft:type_family"] = map[string]any{"family": []string{"murder"}}
		}
		name := "hive:" + strconv.Itoa(v)
		groups[name] = g
		events["hive_"+strconv.Itoa(v)] = map[string]any{"add": map[string]any{"component_groups": []string{name}}}
	}
	return groups, events
}

var HiveTextClientEntity = map[string]any{
	"format_version": "1.10.0",
	"minecraft:client_entity": map[string]any{
		"description": map[string]any{
			"identifier":         HiveTextEntity,
			"materials":          map[string]any{"default": "player_spectator"},
			"textures":           map[string]any{"default": "textures/entity/cape_invisible"},
			"geometry":           map[string]any{"default": "geometry.item_sprite"},
			"render_controllers": []string{"controller.render.default"},
		},
	},
}

var hiveFormatting = regexp.MustCompile("§.")

func (w *World) skipEntity(kind string) bool {
	return w.SkipCustomEntities && entity.IsCustomType(kind) && !(w.Hive && kind == HiveTextEntity)
}

func (w *World) convertHiveSheep() (sheep, tags int) {
	for rid, ent := range w.memState.entities {
		switch ent.EntityType {
		case "minecraft:sheep":
			name, _ := ent.Metadata[protocol.EntityDataKeyName].(string)
			if strings.TrimSpace(hiveFormatting.ReplaceAllString(name, "")) == "" {
				delete(w.memState.entities, rid)
				continue
			}
			sheep++
		case HiveTextEntity:
		default:
			continue
		}
		if ent.Metadata == nil {
			ent.Metadata = protocol.NewEntityMetadata()
		}
		ent.EntityType = HiveTextEntity
		for _, k := range []uint32{
			protocol.EntityDataKeyColorIndex,
			protocol.EntityDataKeyColorTwoIndex,
			protocol.EntityDataKeyVariant,
			protocol.EntityDataKeyMarkVariant,
		} {
			delete(ent.Metadata, k)
		}
		if _, ok := ent.Metadata[protocol.EntityDataKeyFlags]; ok {
			ent.Metadata.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSheared)
			ent.Metadata.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby)
		}

		ent.Metadata[protocol.EntityDataKeyAlwaysShowNameTag] = byte(1)
		ent.Metadata.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAlwaysShowName)
		ent.Metadata.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible)
		delete(ent.Metadata, protocol.EntityDataKeyScale)
		ent.HasMoved = false
		ent.DeletedDistance = -1
		tags++
	}
	return sheep, tags
}
