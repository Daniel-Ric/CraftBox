package structure

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxVolume int64 = 32 << 20
const MaxPalette = 262144

type Vec3 [3]int32
type Bounds struct {
	Min Vec3 `json:"min"`
	Max Vec3 `json:"max"`
}
type State struct {
	Name    string            `json:"name"`
	States  map[string]any    `json:"states"`
	Version int32             `json:"version"`
	Types   map[string]string `json:"types,omitempty"`
}
type Cell struct {
	Layers [2]int32 `json:"layers"`
	Known  bool     `json:"known"`
}
type Region struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Bounds Bounds `json:"bounds"`
}
type Dependency struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	UUID    string `json:"uuid"`
	Source  string `json:"-"`
	Missing bool   `json:"missing"`
}
type Section struct {
	Layers [2][]int32
	Known  []byte
}
type Document struct {
	Name           string
	Size           Vec3
	Origin         Vec3
	Anchor         Vec3
	Palette        []State
	Sections       map[string]*Section
	Default        Cell
	DefaultKnown   bool
	BlockData      map[string]map[string]any
	Entities       []map[string]any
	Regions        []Region
	Packs          []Dependency
	RootExtra      map[string]any
	StructureExtra map[string]any
	PaletteExtra   map[string]any
	owned          map[string]bool
	paletteKeys    map[string]int32
}

var identifier = regexp.MustCompile(`^[a-z0-9_.-]+:[a-z0-9_./-]+$`)

func Key(p Vec3) string { return fmt.Sprintf("%d,%d,%d", p[0], p[1], p[2]) }
func ParseKey(s string) (p Vec3, err error) {
	a := strings.Split(s, ",")
	if len(a) != 3 {
		return p, fmt.Errorf("Invalid block coordinate")
	}
	for i := range p {
		v, e := strconv.ParseInt(a[i], 10, 32)
		if e != nil {
			return p, e
		}
		p[i] = int32(v)
	}
	return
}
func (b Bounds) Normalized() Bounds {
	for i := 0; i < 3; i++ {
		if b.Min[i] > b.Max[i] {
			b.Min[i], b.Max[i] = b.Max[i], b.Min[i]
		}
	}
	return b
}
func (b Bounds) Contains(p Vec3) bool {
	for i := 0; i < 3; i++ {
		if p[i] < b.Min[i] || p[i] > b.Max[i] {
			return false
		}
	}
	return true
}
func (b Bounds) Size() Vec3 {
	return Vec3{b.Max[0] - b.Min[0] + 1, b.Max[1] - b.Min[1] + 1, b.Max[2] - b.Min[2] + 1}
}
func ValidateSize(s Vec3) (int64, error) {
	v := int64(1)
	for _, n := range s {
		if n < 1 || n > 4096 {
			return 0, fmt.Errorf("Each structure dimension must be between 1 and 4096 blocks")
		}
		v *= int64(n)
		if v > MaxVolume {
			return 0, fmt.Errorf("Structure exceeds %d million blocks; split it into named parts", MaxVolume/1_000_000)
		}
	}
	return v, nil
}
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 80 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("Use a name of 1–80 characters without control characters")
	}
	return nil
}
func New(name string, size Vec3) (*Document, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if _, err := ValidateSize(size); err != nil {
		return nil, err
	}
	return &Document{Name: name, Size: size, Palette: []State{{Name: "minecraft:air", States: map[string]any{}, Version: 18168865}}, Sections: map[string]*Section{}, Default: Cell{Layers: [2]int32{0, -1}, Known: true}, DefaultKnown: true, BlockData: map[string]map[string]any{}, owned: map[string]bool{}}, nil
}
func (d *Document) Bounds() Bounds {
	return Bounds{Max: Vec3{d.Size[0] - 1, d.Size[1] - 1, d.Size[2] - 1}}
}
func (d *Document) Volume() int64 { v, _ := ValidateSize(d.Size); return v }
func (d *Document) MarkMissingDependencies() {
	if len(d.Packs) > 0 {
		return
	}
	seen := map[string]bool{}
	for _, state := range d.Palette {
		namespace, _, _ := strings.Cut(state.Name, ":")
		if namespace != "minecraft" && namespace != "" && !seen[namespace] {
			seen[namespace] = true
			d.Packs = append(d.Packs, Dependency{ID: "unresolved:" + namespace, Name: namespace + " — original pack required", Kind: "resource_packs", Missing: true})
		}
	}
}
func sectionAddress(p Vec3) (string, int) {
	return Key(Vec3{p[0] >> 4, p[1] >> 4, p[2] >> 4}), int((p[0]&15)<<8 | (p[1]&15)<<4 | (p[2] & 15))
}
func (d *Document) Cell(p Vec3) Cell {
	if !d.Bounds().Contains(p) {
		return Cell{Layers: [2]int32{-1, -1}}
	}
	key, i := sectionAddress(p)
	s := d.Sections[key]
	if s == nil {
		c := d.Default
		c.Known = d.DefaultKnown
		return c
	}
	return Cell{Layers: [2]int32{s.Layers[0][i], s.Layers[1][i]}, Known: s.Known[i>>3]&(1<<uint(i&7)) != 0}
}
func (d *Document) Set(p Vec3, c Cell) {
	key, i := sectionAddress(p)
	s := d.Sections[key]
	if !d.owned[key] {
		next := &Section{Known: make([]byte, 512)}
		for layer := 0; layer < 2; layer++ {
			next.Layers[layer] = make([]int32, 4096)
			if s != nil {
				copy(next.Layers[layer], s.Layers[layer])
			} else {
				for j := range next.Layers[layer] {
					next.Layers[layer][j] = d.Default.Layers[layer]
				}
			}
		}
		if s != nil {
			copy(next.Known, s.Known)
		} else if d.DefaultKnown {
			for j := range next.Known {
				next.Known[j] = 255
			}
		}
		s = next
		d.Sections[key] = s
		d.owned[key] = true
	}
	for layer := 0; layer < 2; layer++ {
		s.Layers[layer][i] = c.Layers[layer]
	}
	mask := byte(1 << uint(i&7))
	if c.Known {
		s.Known[i>>3] |= mask
	} else {
		s.Known[i>>3] &= ^mask
	}
}
func cloneValue(v any) any {
	if v == nil {
		return nil
	}
	return cloneReflect(reflect.ValueOf(v)).Interface()
}
func cloneReflect(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		o := reflect.New(v.Type()).Elem()
		o.Set(cloneReflect(v.Elem()))
		return o
	case reflect.Map:
		o := reflect.MakeMapWithSize(v.Type(), v.Len())
		it := v.MapRange()
		for it.Next() {
			o.SetMapIndex(it.Key(), cloneReflect(it.Value()))
		}
		return o
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		o := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			o.Index(i).Set(cloneReflect(v.Index(i)))
		}
		return o
	default:
		return v
	}
}
func cloneMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return cloneValue(v).(map[string]any)
}
func (d *Document) Clone() *Document {
	out := *d
	out.Sections = maps.Clone(d.Sections)
	out.owned = map[string]bool{}
	out.Palette = append([]State(nil), d.Palette...)
	out.paletteKeys = maps.Clone(d.paletteKeys)
	out.BlockData = maps.Clone(d.BlockData)
	out.Entities = append([]map[string]any(nil), d.Entities...)
	out.Regions = append([]Region(nil), d.Regions...)
	out.Packs = append([]Dependency(nil), d.Packs...)
	return &out
}
func NormalizeState(s State) (State, error) {
	if !strings.Contains(s.Name, ":") {
		s.Name = "minecraft:" + s.Name
	}
	if !identifier.MatchString(s.Name) || len(s.Name) > 256 {
		return State{}, fmt.Errorf("Invalid Bedrock block identifier %q", s.Name)
	}
	if s.Version == 0 {
		s.Version = 18168865
	}
	s.States = cloneMap(s.States)
	for k, v := range s.States {
		if len(k) > 256 {
			return State{}, fmt.Errorf("Block state name is too long")
		}
		switch x := v.(type) {
		case bool:
			if x {
				s.States[k] = uint8(1)
			} else {
				s.States[k] = uint8(0)
			}
		case float64:
			if x != float64(int32(x)) {
				return State{}, fmt.Errorf("State %s requires an integer", k)
			}
			if s.Types[k] == "byte" {
				if x < 0 || x > 255 {
					return State{}, fmt.Errorf("Byte state out of range")
				}
				s.States[k] = uint8(x)
			} else {
				s.States[k] = int32(x)
			}
		case int:
			s.States[k] = int32(x)
		case int32, uint8, string:
		default:
			return State{}, fmt.Errorf("Unsupported state value %s (%T)", k, v)
		}
	}
	s.Types = map[string]string{}
	for k, v := range s.States {
		switch v.(type) {
		case uint8:
			s.Types[k] = "byte"
		case int32:
			s.Types[k] = "int"
		case string:
			s.Types[k] = "string"
		}
	}
	return s, nil
}
func StateKey(s State) string {
	keys := make([]string, 0, len(s.States))
	for k := range s.States {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(s.Name)
	for _, k := range keys {
		fmt.Fprintf(&b, "|%s=%T:%v", k, s.States[k], s.States[k])
	}
	return b.String()
}
func (d *Document) AddState(s State) (int32, error) {
	s, err := NormalizeState(s)
	if err != nil {
		return 0, err
	}
	if d.paletteKeys == nil {
		d.paletteKeys = map[string]int32{}
		for i, s := range d.Palette {
			d.paletteKeys[StateKey(s)] = int32(i)
		}
	}
	key := StateKey(s)
	if i, ok := d.paletteKeys[key]; ok {
		return i, nil
	}
	if len(d.Palette) >= MaxPalette {
		return 0, fmt.Errorf("Too many block states")
	}
	i := int32(len(d.Palette))
	d.Palette = append(d.Palette, s)
	d.paletteKeys[key] = i
	return i, nil
}
func (d *Document) StateAt(p Vec3) (State, bool) {
	c := d.Cell(p)
	if !c.Known || c.Layers[0] < 0 {
		return State{}, false
	}
	return d.Palette[c.Layers[0]], true
}
func Air(s State) bool {
	return s.Name == "minecraft:air" || s.Name == "minecraft:cave_air" || s.Name == "minecraft:void_air"
}
func each(ctx context.Context, b Bounds, fn func(Vec3) error) error {
	var n int
	for x := b.Min[0]; x <= b.Max[0]; x++ {
		for y := b.Min[1]; y <= b.Max[1]; y++ {
			for z := b.Min[2]; z <= b.Max[2]; z++ {
				if n&4095 == 0 {
					if err := ctx.Err(); err != nil {
						return err
					}
				}
				n++
				if err := fn(Vec3{x, y, z}); err != nil {
					return err
				}
			}
		}
	}
	return ctx.Err()
}
func (d *Document) Validate() error {
	if _, err := ValidateSize(d.Size); err != nil {
		return err
	}
	if err := ValidateName(d.Name); err != nil {
		return err
	}
	if len(d.Palette) < 1 || len(d.Palette) > MaxPalette {
		return fmt.Errorf("Invalid block palette")
	}
	for _, s := range d.Palette {
		if _, err := NormalizeState(s); err != nil {
			return err
		}
	}
	for key, s := range d.Sections {
		p, err := ParseKey(key)
		if err != nil {
			return err
		}
		for i := range p {
			if p[i] < 0 || p[i] >= (d.Size[i]+15)/16 {
				return fmt.Errorf("Section outside structure")
			}
		}
		if s == nil || len(s.Known) != 512 {
			return fmt.Errorf("Invalid known-cell mask")
		}
		for _, indices := range s.Layers {
			if len(indices) != 4096 {
				return fmt.Errorf("Invalid section length")
			}
			for _, i := range indices {
				if i < -1 || int(i) >= len(d.Palette) {
					return fmt.Errorf("Block palette index out of range")
				}
			}
		}
	}
	for _, i := range d.Default.Layers {
		if i < -1 || int(i) >= len(d.Palette) {
			return fmt.Errorf("Default palette index out of range")
		}
	}
	for key := range d.BlockData {
		p, err := ParseKey(key)
		if err != nil || !d.Bounds().Contains(p) {
			return fmt.Errorf("Block metadata lies outside structure")
		}
	}
	if !d.Bounds().Contains(d.Anchor) {
		return fmt.Errorf("Anchor lies outside structure")
	}
	for _, r := range d.Regions {
		if !d.Bounds().Contains(r.Bounds.Min) || !d.Bounds().Contains(r.Bounds.Max) {
			return fmt.Errorf("Named region lies outside structure")
		}
	}
	return nil
}
func PrettyState(s State) string { raw, _ := json.Marshal(s.States); return s.Name + " " + string(raw) }
