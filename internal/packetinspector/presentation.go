package packetinspector

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"reflect"
	"strings"
	"sync"
)

// inspectionValue exposes protocol wrappers whose private fields otherwise
// disappear as {}. It is a view, never the editable packet or wire encoding.
func inspectionValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return inspectionValue(v.Elem())
	}
	if v.CanInterface() {
		switch flags := v.Interface().(type) {
		case protocol.InputFlags:
			ids := []int{}
			for i := 0; i < flags.Len(); i++ {
				if flags.Load(i) {
					ids = append(ids, i)
				}
			}
			return map[string]any{"present": flags.Present(), "size": flags.Len(), "set": ids}
		case protocol.Bitset:
			ids := []int{}
			for i := 0; i < flags.Len(); i++ {
				if flags.Load(i) {
					ids = append(ids, i)
				}
			}
			return map[string]any{"size": flags.Len(), "set": ids}
		}
		if strings.HasPrefix(v.Type().Name(), "Optional[") && v.Type().PkgPath() == reflect.TypeOf(protocol.Option(0)).PkgPath() {
			values := v.MethodByName("Value").Call(nil)
			if !values[1].Bool() {
				return nil
			}
			return inspectionValue(values[0])
		}
		if _, ok := v.Interface().(json.Marshaler); ok {
			return v.Interface()
		}
	}
	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return v.Bytes()
		}
		if v.IsNil() {
			return nil
		}
		fallthrough
	case reflect.Array:
		values := make([]any, v.Len())
		for i := range values {
			values[i] = inspectionValue(v.Index(i))
		}
		return values
	case reflect.Struct:
		fields := map[string]any{}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			value := inspectionValue(v.Field(i))
			if f.Name == "FormData" || f.Name == "ResponseData" {
				if raw, ok := value.([]byte); ok {
					value = readableBytes(raw)
				}
			}
			fields[name] = value
		}
		return fields
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		values := map[string]any{}
		iter := v.MapRange()
		for iter.Next() {
			if iter.Key().Kind() == reflect.String {
				values[iter.Key().String()] = inspectionValue(iter.Value())
			} else {
				return v.Interface()
			}
		}
		return values
	}
	return v.Interface()
}
func readableBytes(raw []byte) any {
	if json.Valid(raw) {
		return json.RawMessage(raw)
	}
	return raw
}
func inspectPayload(pk packet.Packet) string {
	if !inspectionRequired(reflect.ValueOf(pk)) {
		return ""
	}
	raw, e := json.Marshal(inspectionValue(reflect.ValueOf(pk)))
	if e != nil {
		return ""
	}
	return string(raw)
}

// readablePayload also upgrades old archives without rewriting their contents.
func readablePayload(p Packet) string {
	if p.Decoded != "" {
		return p.Decoded
	}
	// Old private wrappers can be recovered from retained wire data. Gate by
	// packet schema so large plain chunk/pack records aren't decoded again.
	if len(p.Raw) > 0 && !p.Partial && readablePacketType(p.PacketID) {
		if pk, e := Decode(p.PacketID, p.Raw, p.ShieldID); e == nil {
			canonical, _ := json.Marshal(pk)
			valid := string(canonical) == p.Payload
			if !valid {
				roundtrip, e := Encode(pk, p.ShieldID)
				valid = e == nil && bytes.Equal(roundtrip, p.Raw)
			}
			if valid {
				if view := inspectPayload(pk); view != "" {
					return view
				}
			}
		}
	}

	d := json.NewDecoder(strings.NewReader(p.Payload))
	d.UseNumber()
	var value any
	if d.Decode(&value) != nil {
		return p.Payload
	}
	changed := false
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				if s, ok := v.(string); ok && (k == "FormData" || k == "ResponseData") {
					if b, e := base64.StdEncoding.DecodeString(s); e == nil && json.Valid(b) {
						dec := json.NewDecoder(bytes.NewReader(b))
						dec.UseNumber()
						var val any
						if dec.Decode(&val) == nil {
							x[k] = val
							changed = true
						}
					}
				} else {
					walk(v)
				}
			}
		case []any:
			for _, v := range x {
				walk(v)
			}
		}
	}
	walk(value)
	if !changed {
		return p.Payload
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return p.Payload
	}
	return string(raw)
}
func packetSummary(p Packet) string {
	d := json.NewDecoder(strings.NewReader(readablePayload(p)))
	d.UseNumber()
	var m map[string]any
	if d.Decode(&m) != nil {
		return ""
	}
	parts := []string{}
	for _, key := range []string{"Message", "Text", "FormID", "Title", "EntityRuntimeID", "EntityUniqueID", "Position", "ChunkX", "ChunkZ", "WindowID", "Status", "CommandLine", "ActionType", "FormData"} {
		if v, ok := m[key]; ok && v != nil {
			b, _ := json.Marshal(v)
			s := string(b)
			if len(s) > 100 {
				s = s[:100] + "..."
			}
			parts = append(parts, key+"="+s)
			if len(parts) == 3 {
				break
			}
		}
	}
	return strings.Join(parts, " · ")
}

// Most packets already have complete JSON. Avoid duplicating large chunk/pack
// byte arrays merely to build an identical presentation.
func inspectionRequired(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		return inspectionRequired(v.Elem())
	}
	if v.Type() == reflect.TypeOf(protocol.InputFlags{}) || v.Type() == reflect.TypeOf(protocol.Bitset{}) {
		return true
	}
	if strings.HasPrefix(v.Type().Name(), "Optional[") && v.Type().PkgPath() == reflect.TypeOf(protocol.Option(0)).PkgPath() {
		return true
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.PkgPath != "" {
				continue
			}
			if f.Name == "FormData" || f.Name == "ResponseData" {
				return true
			}
			if inspectionRequired(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return false
		}
		for i := 0; i < v.Len(); i++ {
			if inspectionRequired(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if inspectionRequired(iter.Value()) {
				return true
			}
		}
	}
	return false
}

var readableTypes sync.Map

func readablePacketType(id uint32) bool {
	if value, ok := readableTypes.Load(id); ok {
		return value.(bool)
	}
	f := serverPackets[id]
	if f == nil {
		f = clientPackets[id]
	}
	if f == nil {
		return false
	}
	var walk func(reflect.Type, map[reflect.Type]bool) bool
	walk = func(t reflect.Type, seen map[reflect.Type]bool) bool {
		if seen[t] {
			return false
		}
		seen[t] = true
		if t == reflect.TypeOf(protocol.InputFlags{}) || t == reflect.TypeOf(protocol.Bitset{}) {
			return true
		}
		if strings.HasPrefix(t.Name(), "Optional[") && t.PkgPath() == reflect.TypeOf(protocol.Option(0)).PkgPath() {
			return true
		}
		switch t.Kind() {
		case reflect.Interface:
			return true
		case reflect.Pointer, reflect.Slice, reflect.Array:
			return walk(t.Elem(), seen)
		case reflect.Map:
			return walk(t.Elem(), seen)
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if f.PkgPath == "" && (f.Name == "FormData" || f.Name == "ResponseData" || walk(f.Type, seen)) {
					return true
				}
			}
		}
		return false
	}
	required := walk(reflect.TypeOf(f()), map[reflect.Type]bool{})
	readableTypes.Store(id, required)
	return required
}
