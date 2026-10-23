package nbtconv

import (
	"bytes"
	"encoding/gob"
	"sort"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)


func WriteItem(s item.Stack, disk bool) map[string]any {
	tag := make(map[string]any)
	if nbt, ok := s.Item().(world.NBTer); ok {
		for k, v := range nbt.EncodeNBT() {
			tag[k] = v
		}
	}
	writeAnvilCost(tag, s)
	writeDamage(tag, s, disk)
	writeDisplay(tag, s)
	writeDragonflyData(tag, s)
	writeEnchantments(tag, s)

	data := make(map[string]any)
	if disk {
		writeItemStack(data, tag, s)
	} else {
		for k, v := range tag {
			data[k] = v
		}
	}
	return data
}


func WriteBlock(b world.Block) map[string]any {
	name, properties := b.EncodeBlock()
	return map[string]any{
		"name":    name,
		"states":  properties,
		"version": chunk.CurrentBlockVersion,
	}
}


func writeItemStack(m, t map[string]any, s item.Stack) {
	m["Name"], m["Damage"] = s.Item().EncodeItem()
	if b, ok := s.Item().(world.Block); ok {
		v := map[string]any{}
		writeBlock(v, b)
		m["Block"] = v
	}
	m["Count"] = byte(s.Count())
	if len(t) > 0 {
		m["tag"] = t
	}
}


func writeBlock(m map[string]any, b world.Block) {
	m["name"], m["states"] = b.EncodeBlock()
	m["version"] = chunk.CurrentBlockVersion
}


func writeDragonflyData(m map[string]any, s item.Stack) {
	if v := s.Values(); len(v) != 0 {
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(mapToSlice(v)); err != nil {
			panic("error encoding item user data: " + err.Error())
		}
		m["dragonflyData"] = buf.Bytes()
	}
}



func mapToSlice(m map[string]any) []mapValue {
	values := make([]mapValue, 0, len(m))
	for k, v := range m {
		values = append(values, mapValue{K: k, V: v})
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].K < values[j].K
	})
	return values
}



type mapValue struct {
	K string
	V any
}


func writeEnchantments(m map[string]any, s item.Stack) {
	if len(s.Enchantments()) != 0 {
		var enchantments []map[string]any
		for _, e := range s.Enchantments() {
			if eType, ok := item.EnchantmentID(e.Type()); ok {
				enchantments = append(enchantments, map[string]any{
					"id":  int16(eType),
					"lvl": int16(e.Level()),
				})
			}
		}
		m["ench"] = enchantments
	}
}

func writeTrim(m map[string]any, t item.ArmourTrim) {
	if !t.Zero() {
		m["Trim"] = map[string]any{
			"Material": t.Material.TrimMaterial(),
			"Pattern":  t.Template.String(),
		}
	}
}


func writeDisplay(m map[string]any, s item.Stack) {
	name, lore := s.CustomName(), s.Lore()
	v := map[string]any{}
	if name != "" {
		v["Name"] = name
	}
	if len(lore) != 0 {
		v["Lore"] = lore
	}
	if len(v) != 0 {
		m["display"] = v
	}
}



func writeDamage(m map[string]any, s item.Stack, disk bool) {
	if v, ok := m["Damage"]; !ok || v.(int16) == 0 {
		if _, ok := s.Item().(item.Durable); ok {
			if disk {
				m["Damage"] = int16(s.MaxDurability() - s.Durability())
			} else {
				m["Damage"] = int32(s.MaxDurability() - s.Durability())
			}
		}
	}
}


func writeAnvilCost(m map[string]any, s item.Stack) {
	if cost := s.AnvilCost(); cost > 0 {
		m["RepairCost"] = int32(cost)
	}
}
