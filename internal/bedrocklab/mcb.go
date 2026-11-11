package bedrocklab

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type schema map[string]any
type schemaDocument struct {
	ID, File, Title, Version string
	Schema                   schema
}
type Schemas struct {
	root      string
	documents []*schemaDocument
	byID      map[string]*schemaDocument
	byFile    map[string]*schemaDocument
}

func text(m schema, key string) string { s, _ := m[key].(string); return s }
func object(v any) schema              { m, _ := v.(map[string]any); return schema(m) }
func array(v any) []any                { a, _ := v.([]any); return a }
func norm(v string) string {
	decoded, e := url.PathUnescape(v)
	if e == nil {
		v = decoded
	}
	return strings.ToLower(path.Clean("/" + strings.ReplaceAll(v, "\\", "/")))
}
func title(v string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(v, "%20", " ")))
}
func schemaType(m schema) string {
	if s, ok := m["type"].(string); ok {
		return s
	}
	for _, v := range array(m["type"]) {
		if s, ok := v.(string); ok && s != "null" {
			return s
		}
	}
	if m["properties"] != nil || m["additionalProperties"] != nil {
		return "object"
	}
	return ""
}
func versionCompare(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(as), len(bs)); i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			return av - bv
		}
	}
	return 0
}
func LoadSchemas(root string) (*Schemas, error) {
	r := &Schemas{byID: map[string]*schemaDocument{}, byFile: map[string]*schemaDocument{}}
	if st, e := os.Stat(filepath.Join(root, "metadata", "json_schemas")); e == nil && st.IsDir() {
		root = filepath.Join(root, "metadata", "json_schemas")
	}
	root, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	r.root = root
	total := int64(0)
	e = filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".json") {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		total += info.Size()
		if info.Size() > 16<<20 || total > 512<<20 || len(r.documents) > 100000 {
			return errors.New("schema collection exceeds limits")
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		m := schema{}
		if e = json.Unmarshal(bytes.TrimPrefix(b, []byte{239, 187, 191}), &m); e != nil {
			return fmt.Errorf("schema %s: %w", p, e)
		}
		if text(m, "$id") == "" {
			return nil
		}
		doc := &schemaDocument{norm(text(m, "$id")), p, title(text(m, "title")), text(m, "x-format-version"), m}
		if r.byID[doc.ID] != nil {
			return fmt.Errorf("duplicate schema ID %s", doc.ID)
		}
		r.documents = append(r.documents, doc)
		r.byID[doc.ID] = doc
		r.byFile[strings.ToLower(doc.File)] = doc
		return nil
	})
	if e != nil {
		return nil, e
	}
	if len(r.documents) == 0 {
		return nil, errors.New("no schemas with $id found")
	}
	return r, nil
}
func (r *Schemas) resolve(m schema, d *schemaDocument) (schema, *schemaDocument, error) {
	for depth := 0; text(m, "$ref") != ""; depth++ {
		if depth > 64 {
			return nil, d, errors.New("cyclic schema reference")
		}
		ref := text(m, "$ref")
		base, fragment, _ := strings.Cut(ref, "#")
		target := d
		if base != "" {
			id := norm(path.Join(path.Dir(d.ID), base))
			if strings.HasPrefix(base, "/") {
				id = norm(base)
			}
			target = r.byID[id]
			if target == nil {
				decoded, _ := url.PathUnescape(base)
				target = r.byFile[strings.ToLower(filepath.Clean(filepath.Join(filepath.Dir(d.File), filepath.FromSlash(decoded))))]
			}
			if target == nil {
				return nil, d, fmt.Errorf("missing schema reference %s", ref)
			}
		}
		var v any = map[string]any(target.Schema)
		if fragment != "" {
			if !strings.HasPrefix(fragment, "/") {
				return nil, d, fmt.Errorf("unsupported schema fragment %s", ref)
			}
			for _, part := range strings.Split(fragment[1:], "/") {
				part, e := url.PathUnescape(part)
				if e != nil {
					return nil, d, e
				}
				part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
				switch cur := v.(type) {
				case map[string]any:
					v = cur[part]
				case []any:
					i, e := strconv.Atoi(part)
					if e != nil || i < 0 || i >= len(cur) {
						return nil, d, errors.New("invalid schema array pointer")
					}
					v = cur[i]
				default:
					return nil, d, errors.New("missing schema pointer")
				}
			}
		}
		resolved := object(v)
		if resolved == nil {
			return nil, d, fmt.Errorf("missing schema pointer %s", ref)
		}
		merged := schema{}
		for k, v := range resolved {
			merged[k] = v
		}
		for k, v := range m {
			if k != "$ref" {
				merged[k] = v
			}
		}
		m, d = merged, target
	}
	return m, d, nil
}

var rootAliases = map[string][]string{
	"minecraft:camera_entity":       {"camera document"},
	"minecraft:block_culling_rules": {"block culling"},
	"minecraft:processor_list":      {"rules used by jigsaw structures to determine which blocks to modify or replace when placing a structure template in the world."},
	"minecraft:template_pool":       {"used to pair block rules with structure templates and to randomly place structure templates using a weighted list."},
	"minecraft:structure_set":       {"a jigsaw structure set is a collection of jigsaw structures that are placed according to a set of rules."},
	"minecraft:entity":              {"actor document"}, "minecraft:client_entity": {"atomic client entity document"}, "minecraft:block": {"blockdefinitiondocument"}, "minecraft:item": {"item document", "item document (beta)"}, "minecraft:client_biome": {"client biome definition"}, "minecraft:biome": {"biome definition"}, "minecraft:spawn_rules": {"spawn rules"}, "minecraft:feature_rules": {"feature rule definition"}, "minecraft:camera_preset": {"camera preset", "camera preset v1.21.80", "camera preset v1.26.50"}, "minecraft:dimension": {"dimension"}, "minecraft:voxel_shape": {"voxelshapefile"}, "minecraft:crafting_items_catalog": {"crafting catalog document"}, "tiers": {"trade table"}, "sound_definitions": {"server sound definition document"}, "minecraft:aim_assist_categories": {"cameraaimassistcategoriesdefinition"}, "minecraft:aim_assist_preset": {"cameraaimassistpresetdefinition"}, "minecraft:camera_custom_splines": {"camerasplinefile"}, "minecraft:jigsaw": {"jigsawstructure"}, "minecraft:jigsaw_structure_metadata": {"jigsaw structure metadata registry"},
}

func (r *Schemas) selectRoot(key, version string) (schema, *schemaDocument, error) {
	type candidate struct {
		m       schema
		d       *schemaDocument
		version string
	}
	candidates := []candidate{}
	for _, d := range r.documents {
		props := object(d.Schema["properties"])
		if props["format_version"] != nil && props[key] != nil {
			m, doc, e := r.resolve(object(props[key]), d)
			if e == nil {
				candidates = append(candidates, candidate{m, doc, d.Version})
			}
		}
	}
	for _, d := range r.documents {
		match := d.Title == title(key) || d.Title == title(strings.TrimPrefix(key, "minecraft:"))
		for _, alias := range rootAliases[strings.ToLower(key)] {
			if d.Title == alias {
				match = true
			}
		}
		if match {
			candidates = append(candidates, candidate{d.Schema, d, d.Version})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return versionCompare(candidates[i].version, candidates[j].version) > 0 })
	for _, c := range candidates {
		if c.version == version || numericFormatVersion.MatchString(version) && (c.version == "" || numericFormatVersion.MatchString(c.version) && versionCompare(c.version, version) <= 0) {
			return c.m, c.d, nil
		}
	}
	return nil, nil, fmt.Errorf("no compatible schema for %s (%s)", key, version)
}

type decodeFailure struct{ err error }

var numericFormatVersion = regexp.MustCompile(`^\d+(\.\d+)*$`)

type mcbReader struct {
	b          []byte
	pos, nodes int
	location   string
	r          *Schemas
}

func (m *mcbReader) fail(s string) {
	panic(decodeFailure{fmt.Errorf("%s at %s, byte %d (0x%x)", s, m.location, m.pos, m.pos)})
}
func (m *mcbReader) take(n int) []byte {
	if n < 0 || n > len(m.b)-m.pos {
		m.fail("truncated MCB")
	}
	b := m.b[m.pos : m.pos+n]
	m.pos += n
	return b
}
func (m *mcbReader) u8() byte    { return m.take(1)[0] }
func (m *mcbReader) u16() uint16 { return binary.LittleEndian.Uint16(m.take(2)) }
func (m *mcbReader) u32() uint32 { return binary.LittleEndian.Uint32(m.take(4)) }
func (m *mcbReader) boolean() bool {
	b := m.u8()
	if b > 1 {
		m.fail("invalid boolean")
	}
	return b == 1
}
func (m *mcbReader) varuint() uint32 {
	var result uint32
	for i := 0; i < 5; i++ {
		b := m.u8()
		if i == 4 && b > 15 {
			m.fail("varuint overflow")
		}
		result |= uint32(b&127) << uint(i*7)
		if b < 128 {
			return result
		}
	}
	m.fail("unterminated varuint")
	return 0
}
func (m *mcbReader) str() string {
	n := m.varuint()
	if n > 16<<20 {
		m.fail("string exceeds limit")
	}
	b := m.take(int(n))
	if !utf8.Valid(b) {
		m.fail("invalid UTF-8")
	}
	return string(b)
}
func (m *mcbReader) count() int {
	n := m.varuint()
	if n > 1000000 {
		m.fail("container exceeds limit")
	}
	return int(n)
}
func (m *mcbReader) number(underlying string) any {
	switch underlying {
	case "uint8":
		return m.u8()
	case "int8":
		return int8(m.u8())
	case "uint16":
		return m.u16()
	case "int16":
		return int16(m.u16())
	case "uint32":
		return m.u32()
	case "int32":
		return int32(m.u32())
	case "uint64":
		n := binary.LittleEndian.Uint64(m.take(8))
		return n
	case "int64":
		n := int64(binary.LittleEndian.Uint64(m.take(8)))
		return n
	case "float":
		n := float64(math.Float32frombits(m.u32()))
		if math.IsNaN(n) || math.IsInf(n, 0) {
			m.fail("non-finite number")
		}
		v, _ := strconv.ParseFloat(strconv.FormatFloat(n, 'g', 7, 64), 64)
		return v
	case "double":
		n := math.Float64frombits(binary.LittleEndian.Uint64(m.take(8)))
		if math.IsNaN(n) || math.IsInf(n, 0) {
			m.fail("non-finite number")
		}
		return n
	}
	m.fail("unsupported numeric encoding " + underlying)
	return nil
}

var preferredVariants = map[string]string{"molang string": "string", "crafting catalog item": "string", "vectorevents": "array", "color_expr": "array", "particle_motion_collision_event_vector": "array", "vec3": "array", "item descriptor": "object", "minecraft:icon v1.21.80": "object", "trade quantity": "object", "minecraft:hand_equipped": "boolean", "minecraft:max_stack_size": "integer"}

func (m *mcbReader) node(s schema, d *schemaDocument, depth int) any {
	return m.nodeAt(s, d, depth, m.location)
}
func (m *mcbReader) nodeAt(s schema, d *schemaDocument, depth int, location string) any {
	previous := m.location
	m.location = location
	defer func() { m.location = previous }()
	m.nodes++
	if depth > 128 || m.nodes > 3000000 {
		m.fail("schema recursion or node limit")
	}
	var e error
	s, d, e = m.r.resolve(s, d)
	if e != nil {
		m.fail(e.Error())
	}
	if variants := array(s["oneOf"]); len(variants) > 0 {
		t := title(text(s, "title"))
		if t == "tradeitemlist" {
			n := m.count()
			values := []any{}
			for i := 0; i < n; i++ {
				values = append(values, m.node(object(variants[0]), d, depth+1))
			}
			if n == 1 {
				return values[0]
			}
			return map[string]any{"choice": values}
		}
		tagged := t == "particle_appearance_tinting color_data" || t == "particle_curve"
		for _, v := range variants {
			if _, ok := object(v)["x-ordinal-index"].(float64); ok {
				tagged = true
			}
		}
		if tagged {
			tag := int(m.u8())
			for i, v := range variants {
				index := i
				if ordinal, ok := object(v)["x-ordinal-index"].(float64); ok {
					index = int(ordinal)
				}
				if index == tag {
					return m.node(object(v), d, depth+1)
				}
			}
			m.fail("unknown variant tag")
		}
		if preferred := preferredVariants[t]; preferred != "" {
			for _, v := range variants {
				resolved, _, e := m.r.resolve(object(v), d)
				if e == nil && schemaType(resolved) == preferred {
					return m.node(object(v), d, depth+1)
				}
			}
		}
		if len(variants) == 1 {
			return m.node(object(variants[0]), d, depth+1)
		}
		m.fail("ambiguous oneOf: " + t)
	}
	switch schemaType(s) {
	case "string":
		v := m.str()
		if enums := array(s["enum"]); len(enums) > 0 {
			valid := false
			for _, allowed := range enums {
				if allowed == v {
					valid = true
				}
			}
			if !valid {
				m.fail("unknown enum value")
			}
		}
		return v
	case "boolean":
		return m.boolean()
	case "integer", "number":
		return m.number(text(s, "x-underlying-type"))
	case "array":
		values := []any{}
		if tuple, ok := s["items"].([]any); ok {
			for _, item := range tuple {
				values = append(values, m.nodeAt(object(item), d, depth+1, fmt.Sprintf("%s[%d]", m.location, len(values))))
			}
			return values
		}
		item := object(s["items"])
		if item == nil {
			m.fail("missing array items")
		}
		n := 0
		if min, ok := s["minItems"].(float64); ok && min == s["maxItems"] {
			if min < 0 || min > 1000000 || min != math.Trunc(min) {
				m.fail("invalid fixed array length")
			}
			n = int(min)
		} else {
			n = m.count()
		}
		for i := 0; i < n; i++ {
			values = append(values, m.nodeAt(item, d, depth+1, fmt.Sprintf("%s[%d]", m.location, i)))
		}
		return values
	case "object":
		props := object(s["properties"])
		out := map[string]any{}
		component := len(props) > 0
		for name, v := range props {
			if !strings.Contains(name, ":") || object(v)["x-ordinal-index"] != nil {
				component = false
			}
		}
		if component {
			ids := map[uint32]string{}
			for name := range props {
				h := fnv.New32a()
				h.Write([]byte(name))
				id := h.Sum32()
				if ids[id] != "" {
					m.fail("component hash collision")
				}
				ids[id] = name
			}
			n := m.u32()
			if n > 1000000 {
				m.fail("component count exceeds limit")
			}
			for i := uint32(0); i < n; i++ {
				id := m.u32()
				name := ids[id]
				if name == "" {
					m.fail(fmt.Sprintf("missing component schema %08x", id))
				}
				if _, ok := out[name]; ok {
					m.fail("duplicate component")
				}
				out[name] = m.nodeAt(object(props[name]), d, depth+1, m.location+"."+name)
			}
			return out
		}
		if len(props) == 0 && s["additionalProperties"] != nil {
			value := object(s["additionalProperties"])
			if value == nil {
				m.fail("unsupported map schema")
			}
			n := m.count()
			keyType := text(s, "x-key-underlying-type")
			for i := 0; i < n; i++ {
				key := ""
				switch keyType {
				case "", "string":
					key = m.str()
				case "float", "int32":
					key = fmt.Sprint(m.number(keyType))
				default:
					m.fail("unsupported map key type")
				}
				if _, ok := out[key]; ok {
					m.fail("duplicate map key")
				}
				out[key] = m.nodeAt(value, d, depth+1, m.location+"["+strconv.Quote(key)+"]")
			}
			return out
		}
		names := make([]string, 0, len(props))
		ordinals := map[float64]bool{}
		for name, v := range props {
			ordinal, ok := object(v)["x-ordinal-index"].(float64)
			if !ok || ordinal < 0 || ordinal != math.Trunc(ordinal) || ordinals[ordinal] {
				m.fail("missing or duplicate field ordinal: " + name)
			}
			ordinals[ordinal] = true
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return object(props[names[i]])["x-ordinal-index"].(float64) < object(props[names[j]])["x-ordinal-index"].(float64)
		})
		required := map[string]bool{}
		for _, v := range array(s["required"]) {
			if name, ok := v.(string); ok {
				required[name] = true
			}
		}
		for _, name := range names {
			property := object(props[name])
			_, hasDefault := property["default"]
			m.location = location + "." + name
			if !required[name] && !hasDefault && !m.boolean() {
				continue
			}
			out[name] = m.nodeAt(property, d, depth+1, m.location)
		}
		return out
	}
	m.fail("unsupported schema type " + schemaType(s))
	return nil
}
func (r *Schemas) Decode(b []byte) (result []byte, err error) {
	defer func() {
		if v := recover(); v != nil {
			if failure, ok := v.(decodeFailure); ok {
				err = failure.err
			} else {
				panic(v)
			}
		}
	}()
	m := &mcbReader{b: b, r: r, location: "$header"}
	if m.u32() != 0x42434d7f {
		return nil, errors.New("invalid MCB magic")
	}
	major, minor, patch := m.u16(), m.u16(), m.u16()
	pre, build := m.str(), m.str()
	version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	if major == 9999 && minor == 9999 && patch == 9999 && pre == "beta" && build == "" {
		version = "beta"
	} else {
		if pre != "" {
			version += "-" + pre
		}
		if build != "" {
			version += "+" + build
		}
	}
	key := m.str()
	s, d, e := r.selectRoot(key, version)
	if e != nil {
		return nil, e
	}
	v := m.nodeAt(s, d, 0, "$."+key)
	if m.pos != len(b) {
		return nil, fmt.Errorf("%d trailing bytes", len(b)-m.pos)
	}
	value := map[string]any{key: v}
	if key != "tiers" {
		value["format_version"] = version
	}
	return json.MarshalIndent(value, "", "  ")
}
