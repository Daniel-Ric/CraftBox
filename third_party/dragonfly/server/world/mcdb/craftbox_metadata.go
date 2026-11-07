package mcdb

import (
	"reflect"
	"strings"

	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func metadataFields(t reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := strings.SplitN(f.Tag.Get("nbt"), ",", 2)[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		fields[name] = f.Type
	}
	return fields
}
func projectMetadata(source map[string]any, t reflect.Type) map[string]any {
	out := map[string]any{}
	fields := metadataFields(t)
	for key, value := range source {
		kind, known := fields[key]
		if !known {
			continue
		}
		if nested, ok := value.(map[string]any); ok && kind.Kind() == reflect.Struct {
			value = projectMetadata(nested, kind)
		}
		out[key] = value
	}
	return out
}
func mergeMetadata(source, known map[string]any, t reflect.Type) map[string]any {
	out := make(map[string]any, len(source)+len(known))
	for key, value := range source {
		out[key] = value
	}
	fields := metadataFields(t)
	for key, value := range known {
		if kind, ok := fields[key]; ok && kind.Kind() == reflect.Struct {
			if nested, ok := value.(map[string]any); ok {
				original, _ := source[key].(map[string]any)
				value = mergeMetadata(original, nested, kind)
			}
		}
		out[key] = value
	}
	return out
}
func (db *DB) decodeMetadata(file *leveldat.LevelDat) error {
	if err := file.Unmarshal(&db.levelSource); err != nil {
		return err
	}
	known := projectMetadata(db.levelSource, reflect.TypeFor[leveldat.Data]())
	raw, err := nbt.MarshalEncoding(known, nbt.LittleEndian)
	if err != nil {
		return err
	}
	return nbt.UnmarshalEncoding(raw, db.ldat, nbt.LittleEndian)
}
func (db *DB) encodeMetadata() (map[string]any, error) {
	raw, err := nbt.MarshalEncoding(db.ldat, nbt.LittleEndian)
	if err != nil {
		return nil, err
	}
	var known map[string]any
	if err = nbt.UnmarshalEncoding(raw, &known, nbt.LittleEndian); err != nil {
		return nil, err
	}
	return mergeMetadata(db.levelSource, known, reflect.TypeFor[leveldat.Data]()), nil
}
