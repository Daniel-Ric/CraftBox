package nbt

import (
	"bytes"
	"fmt"
	"go/ast"
	"io"
	"reflect"
	"strings"
	"sync"
)

type Decoder struct {
	Encoding Encoding

	AllowZero bool

	r     *offsetReader
	depth int
}

var mapStringAnyType = reflect.TypeOf(map[string]any(nil))

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{Encoding: NetworkLittleEndian, r: newOffsetReader(r)}
}

func NewDecoderWithEncoding(r io.Reader, encoding Encoding) *Decoder {
	return &Decoder{Encoding: encoding, r: newOffsetReader(r)}
}

func (d *Decoder) Decode(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Pointer {
		return NonPointerTypeError{ActualType: val.Type()}
	}
	tagType, tagName, err := d.tag()
	if err != nil {
		return err
	}
	if tagType == tagEnd && d.AllowZero {

		if m, ok := v.(*map[string]any); ok {
			if *m == nil {
				*m = make(map[string]any)
			} else {
				clear(*m)
			}
		}
		return nil
	}
	return d.unmarshalTag(val.Elem(), tagType, tagName)
}

func Unmarshal(data []byte, v any) error {
	return UnmarshalEncoding(data, v, NetworkLittleEndian)
}

func UnmarshalEncoding(data []byte, v any, encoding Encoding) error {
	buf := bytes.NewBuffer(data)
	return (&Decoder{Encoding: encoding, r: &offsetReader{
		Reader:   buf,
		ReadByte: buf.ReadByte,
		Next:     buf.Next,
	}}).Decode(v)
}

var stringType = reflect.TypeOf("")
var byteType = reflect.TypeOf(byte(0))
var int32Type = reflect.TypeOf(int32(0))
var int64Type = reflect.TypeOf(int64(0))

var fieldMapPool = sync.Pool{
	New: func() any {
		return map[string]reflect.Value{}
	},
}

func (d *Decoder) unmarshalCompoundAny(m map[string]any) error {
	var v any
	vVal := reflect.ValueOf(&v).Elem()
	for {
		nestedTagType, nestedTagName, err := d.tag()
		if err != nil {
			return err
		}
		if !nestedTagType.IsValid() {
			return UnknownTagError{Off: d.r.off, Op: "Map", TagType: nestedTagType}
		}
		if nestedTagType == tagEnd {

			break
		}

		v = nil
		if err := d.unmarshalTag(vVal, nestedTagType, nestedTagName); err != nil {
			return err
		}
		m[nestedTagName] = v
	}
	return nil
}

func (d *Decoder) unmarshalTag(val reflect.Value, t tagType, tagName string) error {
	k := val.Kind()
	switch t {
	default:
		return UnknownTagError{Off: d.r.off, TagType: t, Op: "Match"}
	case tagEnd:
		return UnexpectedTagError{Off: d.r.off, TagType: tagEnd}
	case tagByte:
		value, err := d.r.ReadByte()
		if err != nil {
			return BufferOverrunError{Op: "Byte"}
		}
		switch {
		case k == reflect.Uint8:
			val.SetUint(uint64(value))
		case k == reflect.Bool:
			val.SetBool(value == 1)
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagInt16:
		value, err := d.Encoding.Int16(d.r)
		if err != nil {
			return err
		}
		switch {
		case k == reflect.Int16:
			val.SetInt(int64(value))
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagInt32:
		value, err := d.Encoding.Int32(d.r)
		if err != nil {
			return err
		}
		switch {
		case k == reflect.Int32:
			val.SetInt(int64(value))
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagInt64:
		value, err := d.Encoding.Int64(d.r)
		if err != nil {
			return err
		}
		switch {
		case k == reflect.Int64:
			val.SetInt(value)
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagFloat32:
		value, err := d.Encoding.Float32(d.r)
		if err != nil {
			return err
		}
		switch {
		case k == reflect.Float32:
			val.SetFloat(float64(value))
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagFloat64:
		value, err := d.Encoding.Float64(d.r)
		if err != nil {
			return err
		}
		switch {
		case k == reflect.Float64:
			val.SetFloat(value)
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagString:
		value, err := d.Encoding.String(d.r)
		if err != nil {
			return err
		}
		switch {
		case k == reflect.String:
			val.SetString(value)
		case isAny(val):
			val.Set(reflect.ValueOf(value))
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
	case tagByteArray:
		length, err := d.Encoding.Int32(d.r)
		if err != nil {
			return err
		}
		if err = d.checkRemaining(int(length), "ByteArray"); err != nil {
			return err
		}
		if length < 0 {
			return BufferOverrunError{Op: "ByteArray"}
		}
		b := make([]byte, length)
		if _, err := d.r.Read(b); err != nil {
			return BufferOverrunError{Op: "ByteArray"}
		}
		value := reflect.New(reflect.ArrayOf(int(length), byteType)).Elem()
		reflect.Copy(value, reflect.ValueOf(b))

		switch {
		case k == reflect.Array && val.Type().Elem().Kind() == reflect.Uint8:
			if val.Cap() != int(length) {
				return InvalidArraySizeError{Off: d.r.off, Op: "ByteArray", GoLength: val.Cap(), NBTLength: int(length)}
			}
		case isAny(val):
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
		val.Set(value)
	case tagInt32Array:
		s, err := d.Encoding.Int32Slice(d.r)
		if err != nil {
			return err
		}

		value := reflect.New(reflect.ArrayOf(len(s), int32Type)).Elem()
		reflect.Copy(value, reflect.ValueOf(s))

		switch {
		case k == reflect.Array && val.Type().Elem().Kind() == reflect.Int32:
			if val.Cap() != len(s) {
				return InvalidArraySizeError{Off: d.r.off, Op: "Int32Array", GoLength: val.Cap(), NBTLength: len(s)}
			}
		case isAny(val):
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
		val.Set(value)

	case tagInt64Array:
		s, err := d.Encoding.Int64Slice(d.r)
		if err != nil {
			return err
		}

		value := reflect.New(reflect.ArrayOf(len(s), int64Type)).Elem()
		reflect.Copy(value, reflect.ValueOf(s))

		switch {
		case k == reflect.Array && val.Type().Elem().Kind() == reflect.Int64:
			if val.Cap() != len(s) {
				return InvalidArraySizeError{Off: d.r.off, Op: "Int64Array", GoLength: val.Cap(), NBTLength: len(s)}
			}
		case isAny(val):
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
		val.Set(value)

	case tagSlice:
		d.depth++
		listTypeByte, err := d.r.ReadByte()
		if err != nil {
			return BufferOverrunError{Op: "Slice"}
		}
		listType := tagType(listTypeByte)
		if !listType.IsValid() {
			return UnknownTagError{Off: d.r.off, TagType: listType, Op: "Slice"}
		}
		sliceType := val.Type()
		if val.Kind() != reflect.Slice && !isAny(val) {
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		}
		if val.Kind() == reflect.Interface {
			sliceType = reflect.SliceOf(sliceType)
		}
		switch listType {
		case tagByte:
			length, err := d.Encoding.Int32(d.r)
			if err != nil {
				return BufferOverrunError{Op: "ByteSlice"}
			}
			if length < 0 {
				return BufferOverrunError{Op: "ByteSlice"}
			}
			if length == 0 {

				val.Set(reflect.MakeSlice(sliceType, int(length), int(length)))
				break
			}
			if err = d.checkRemaining(int(length), "ByteSlice"); err != nil {
				return err
			}
			b := make([]byte, length)
			if _, err := d.r.Read(b); err != nil {
				return BufferOverrunError{Op: "ByteSlice"}
			}
			switch {
			case k == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8, isAny(val):
				val.Set(reflect.ValueOf(b))
			default:
				return InvalidTypeError{Off: d.r.off, FieldType: val.Type().Elem(), Field: tagName, TagType: listType}
			}
		case tagInt32:
			b, err := d.Encoding.Int32Slice(d.r)
			if err != nil {
				return BufferOverrunError{Op: "Int32Slice"}
			}
			switch {
			case k == reflect.Slice && val.Type().Elem().Kind() == reflect.Int32, isAny(val):
				val.Set(reflect.ValueOf(b))
			default:
				return InvalidTypeError{Off: d.r.off, FieldType: val.Type().Elem(), Field: tagName, TagType: listType}
			}
		case tagInt64:
			b, err := d.Encoding.Int64Slice(d.r)
			if err != nil {
				return BufferOverrunError{Op: "Int64Slice"}
			}
			switch {
			case k == reflect.Slice && val.Type().Elem().Kind() == reflect.Int64, isAny(val):
				val.Set(reflect.ValueOf(b))
			default:
				return InvalidTypeError{Off: d.r.off, FieldType: val.Type().Elem(), Field: tagName, TagType: listType}
			}
		default:
			length, err := d.Encoding.Int32(d.r)
			if err != nil {
				return err
			}
			if length < 0 {
				return BufferOverrunError{Op: "Slice"}
			}
			v := reflect.MakeSlice(sliceType, int(length), int(length))
			for i := 0; i < int(length); i++ {
				if err := d.unmarshalTag(v.Index(i), listType, ""); err != nil {

					if e, ok := err.(InvalidTypeError); ok {
						return InvalidTypeError{Off: d.r.off, FieldType: sliceType.Elem(), Field: fmt.Sprintf("%v[%v].%v", tagName, i, e.Field), TagType: listType}
					}
					return err
				}
			}
			val.Set(v)
			d.depth--
		}

	case tagStruct:
		d.depth++
		switch val.Kind() {
		default:
			return InvalidTypeError{Off: d.r.off, FieldType: val.Type(), Field: tagName, TagType: t}
		case reflect.Struct:

			fields := fieldMapPool.Get().(map[string]reflect.Value)
			d.populateFields(val, fields)
			for {
				nestedTagType, nestedTagName, err := d.tag()
				if err != nil {
					return err
				}
				if nestedTagType == tagEnd {

					break
				}
				if !nestedTagType.IsValid() {
					return UnknownTagError{Off: d.r.off, Op: "Struct", TagType: nestedTagType}
				}
				field, ok := fields[nestedTagName]
				if ok {
					if err = d.unmarshalTag(field, nestedTagType, nestedTagName); err != nil {
						return err
					}
					continue
				}

				return UnexpectedNamedTagError{Off: d.r.off, TagName: tagName + "." + nestedTagName, TagType: nestedTagType}
			}

			for k := range fields {
				delete(fields, k)
			}
			fieldMapPool.Put(fields)
		case reflect.Interface, reflect.Map:
			valKind := val.Kind()
			valType := val.Type()
			if valKind == reflect.Interface && val.NumMethod() != 0 {
				return InvalidTypeError{Off: d.r.off, FieldType: valType, Field: tagName, TagType: t}
			}

			if (valKind == reflect.Map && valType == mapStringAnyType) || (valKind == reflect.Interface && isAny(val)) {
				m := make(map[string]any)
				if err := d.unmarshalCompoundAny(m); err != nil {
					return err
				}
				val.Set(reflect.ValueOf(m))
				break
			}

			if valKind == reflect.Map {
				valType = valType.Elem()
			}
			m := reflect.MakeMap(reflect.MapOf(stringType, valType))
			for {
				nestedTagType, nestedTagName, err := d.tag()
				if err != nil {
					return err
				}
				if !nestedTagType.IsValid() {
					return UnknownTagError{Off: d.r.off, Op: "Map", TagType: nestedTagType}
				}
				if nestedTagType == tagEnd {

					break
				}
				value := reflect.New(valType).Elem()
				if err := d.unmarshalTag(value, nestedTagType, nestedTagName); err != nil {
					return err
				}
				m.SetMapIndex(reflect.ValueOf(nestedTagName), value)
			}
			val.Set(m)
		}
		d.depth--
	}
	return nil
}

func (d *Decoder) populateFields(val reflect.Value, m map[string]reflect.Value) {
	for i := 0; i < val.NumField(); i++ {
		fieldType := val.Type().Field(i)
		if !ast.IsExported(fieldType.Name) {

			continue
		}
		field := val.Field(i)
		name := fieldType.Name
		if fieldType.Anonymous {

			d.populateFields(field, m)
			continue
		}
		if tag, ok := fieldType.Tag.Lookup("nbt"); ok {
			if tag == "-" {
				continue
			}
			tag = strings.TrimSuffix(tag, ",omitempty")
			if tag != "" {
				name = tag
			}
		}
		m[name] = field
	}
}

func (d *Decoder) tag() (t tagType, tagName string, err error) {
	if d.depth >= maximumNestingDepth {
		return 0, "", MaximumDepthReachedError{}
	}
	if d.r.off >= maximumNetworkOffset && d.Encoding == NetworkLittleEndian {
		return 0, "", MaximumBytesReadError{}
	}
	tagTypeByte, err := d.r.ReadByte()
	if err != nil {
		return 0, "", BufferOverrunError{Op: "ReadTag"}
	}
	t = tagType(tagTypeByte)
	if _, ok := d.Encoding.(networkBigEndian); ok && t == tagStruct && d.depth == 0 {

		return t, "", err
	}
	if t != tagEnd {

		tagName, err = d.Encoding.String(d.r)
	}
	return t, tagName, err
}

func (d *Decoder) checkRemaining(length int, op string) error {
	if remaining, ok := d.r.Reader.(interface{ Len() int }); ok && length > remaining.Len() {
		return BufferOverrunError{Op: op}
	}
	return nil
}

func isAny(v reflect.Value) bool {
	return v.Kind() == reflect.Interface && v.NumMethod() == 0
}
