package structure

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

const MaxNBTBytes = 512 << 20

type ExportOptions struct {
	Entities    bool `json:"entities"`
	Inventories bool `json:"inventories"`
	BlockData   bool `json:"blockData"`
	OmitUnknown bool `json:"omitUnknown"`
	Packs       bool `json:"packs"`
}

func list(v any) []any {
	if v == nil {
		return nil
	}
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Slice && r.Kind() != reflect.Array {
		return nil
	}
	out := make([]any, r.Len())
	for i := range out {
		out[i] = r.Index(i).Interface()
	}
	return out
}
func compound(v any) map[string]any { m, _ := v.(map[string]any); return m }
func integer(v any) (int32, bool) {
	switch n := v.(type) {
	case int32:
		return n, true
	case uint8:
		return int32(n), true
	case int:
		return int32(n), true
	}
	return 0, false
}
func vector(v any) (Vec3, error) {
	a := list(v)
	if len(a) != 3 {
		return Vec3{}, fmt.Errorf("Expected three Bedrock integer coordinates")
	}
	var p Vec3
	for i := range p {
		n, ok := integer(a[i])
		if !ok {
			return p, fmt.Errorf("Expected Bedrock TAG_Int coordinate")
		}
		p[i] = n
	}
	return p, nil
}
func indices(v any) ([]int32, error) {
	if a, ok := v.([]int32); ok {
		return a, nil
	}
	a := list(v)
	out := make([]int32, len(a))
	for i, v := range a {
		n, ok := integer(v)
		if !ok {
			return nil, fmt.Errorf("Invalid block index")
		}
		out[i] = n
	}
	return out, nil
}

// The preflight checks lengths before the generic NBT decoder allocates slices.
// MCStructure uses uncompressed, fixed-width little-endian NBT (not network NBT).
func validateNBT(raw []byte) error {
	pos, nodes := 0, 0
	take := func(n int64) error {
		if n < 0 || n > int64(len(raw)-pos) {
			return io.ErrUnexpectedEOF
		}
		pos += int(n)
		return nil
	}
	readString := func() error {
		if pos+2 > len(raw) {
			return io.ErrUnexpectedEOF
		}
		n := binary.LittleEndian.Uint16(raw[pos:])
		pos += 2
		return take(int64(n))
	}
	var payload func(byte, int) error
	payload = func(tag byte, depth int) error {
		nodes++
		if depth > 64 || nodes > 2_000_000 {
			return fmt.Errorf("NBT nesting or object count exceeds import limits")
		}
		switch tag {
		case 1:
			return take(1)
		case 2:
			return take(2)
		case 3, 5:
			return take(4)
		case 4, 6:
			return take(8)
		case 8:
			return readString()
		case 7, 11, 12:
			if pos+4 > len(raw) {
				return io.ErrUnexpectedEOF
			}
			n := int64(int32(binary.LittleEndian.Uint32(raw[pos:])))
			pos += 4
			unit := int64(1)
			if tag == 11 {
				unit = 4
			}
			if tag == 12 {
				unit = 8
			}
			if n < 0 {
				return fmt.Errorf("Negative NBT array length")
			}
			return take(n * unit)
		case 9:
			if pos+5 > len(raw) {
				return io.ErrUnexpectedEOF
			}
			kind := raw[pos]
			n := int64(int32(binary.LittleEndian.Uint32(raw[pos+1:])))
			pos += 5
			if n < 0 || n > MaxVolume || kind == 0 && n > 0 {
				return fmt.Errorf("Invalid NBT list length")
			}
			width := map[byte]int64{1: 1, 2: 2, 3: 4, 4: 8, 5: 4, 6: 8}[kind]
			if width > 0 {
				return take(n * width)
			}
			if n > int64(len(raw)-pos) {
				return io.ErrUnexpectedEOF
			}
			for i := int64(0); i < n; i++ {
				if err := payload(kind, depth+1); err != nil {
					return err
				}
			}
			return nil
		case 10:
			for {
				if pos >= len(raw) {
					return io.ErrUnexpectedEOF
				}
				kind := raw[pos]
				pos++
				if kind == 0 {
					return nil
				}
				if err := readString(); err != nil {
					return err
				}
				if err := payload(kind, depth+1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("Unsupported NBT tag %d", tag)
		}
	}
	if len(raw) < 3 || raw[0] != 10 {
		return fmt.Errorf("A Bedrock structure must have an NBT compound root")
	}
	pos = 1
	if err := readString(); err != nil {
		return err
	}
	if err := payload(10, 0); err != nil {
		return err
	}
	if pos != len(raw) {
		return fmt.Errorf("Unexpected trailing structure data")
	}
	return nil
}
func readNBT(r io.Reader) (map[string]any, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxNBTBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxNBTBytes {
		return nil, fmt.Errorf("Structure data exceeds 512 MiB")
	}
	if err := validateNBT(raw); err != nil {
		return nil, err
	}
	var root map[string]any
	if err := nbt.UnmarshalEncoding(raw, &root, nbt.LittleEndian); err != nil {
		return nil, err
	}
	return root, nil
}
func ReadMCStructure(ctx context.Context, r io.Reader, name string) (*Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := readNBT(&contextReader{ctx, r})
	if err != nil {
		return nil, err
	}
	if version, ok := integer(root["format_version"]); !ok || version != 1 {
		return nil, fmt.Errorf("Unsupported Bedrock structure format")
	}
	size, err := vector(root["size"])
	if err != nil {
		return nil, err
	}
	d, err := New(name, size)
	if err != nil {
		return nil, err
	}
	volume := d.Volume()
	if value, ok := root["structure_world_origin"]; ok {
		d.Origin, err = vector(value)
		if err != nil {
			return nil, err
		}
	}
	data := compound(root["structure"])
	palette := compound(compound(data["palette"])["default"])
	states := list(palette["block_palette"])
	if len(states) == 0 || len(states) > MaxPalette {
		return nil, fmt.Errorf("Missing or oversized block palette")
	}
	d.Palette = nil
	for _, raw := range states {
		entry := compound(raw)
		s := State{States: compound(entry["states"])}
		s.Name, _ = entry["name"].(string)
		s.Version, _ = integer(entry["version"])
		s, err = NormalizeState(s)
		if err != nil {
			return nil, err
		}
		d.Palette = append(d.Palette, s)
	}
	layers := list(data["block_indices"])
	if len(layers) != 2 {
		return nil, fmt.Errorf("Bedrock structures require two block layers")
	}
	a, err := indices(layers[0])
	if err != nil {
		return nil, err
	}
	b, err := indices(layers[1])
	if err != nil {
		return nil, err
	}
	if int64(len(a)) != volume || int64(len(b)) != volume {
		return nil, fmt.Errorf("Block index count does not match structure size")
	}
	d.Default = Cell{Layers: [2]int32{-1, -1}, Known: true}
	var index int
	err = each(ctx, d.Bounds(), func(p Vec3) error {
		c := Cell{Layers: [2]int32{a[index], b[index]}, Known: true}
		index++
		for _, v := range c.Layers {
			if v < -1 || int(v) >= len(d.Palette) {
				return fmt.Errorf("Block index outside palette")
			}
		}
		if c.Layers != d.Default.Layers {
			d.Set(p, c)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for key, value := range compound(palette["block_position_data"]) {
		i, e := strconv.ParseInt(key, 10, 64)
		if e != nil || i < 0 || i >= volume {
			return nil, fmt.Errorf("Invalid block metadata index")
		}
		p := Vec3{int32(i / int64(size[2]) / int64(size[1])), int32(i / int64(size[2]) % int64(size[1])), int32(i % int64(size[2]))}
		d.BlockData[Key(p)] = cloneMap(compound(value))
	}
	for _, raw := range list(data["entities"]) {
		e := cloneMap(compound(raw))
		if isPlayer(e) {
			continue
		}
		p, ok := entityPosition(e)
		if !ok {
			return nil, fmt.Errorf("Entity is missing a valid Pos vector")
		}
		for i := 0; i < 3; i++ {
			p[i] -= float64(d.Origin[i])
		}
		setEntityPosition(e, p)
		d.Entities = append(d.Entities, e)
	}
	d.RootExtra = cloneMap(root)
	for _, k := range []string{"format_version", "size", "structure_world_origin", "structure"} {
		delete(d.RootExtra, k)
	}
	d.StructureExtra = cloneMap(data)
	for _, k := range []string{"block_indices", "entities", "palette"} {
		delete(d.StructureExtra, k)
	}
	d.PaletteExtra = cloneMap(palette)
	delete(d.PaletteExtra, "block_palette")
	delete(d.PaletteExtra, "block_position_data")
	d.MarkMissingDependencies()
	return d, d.Validate()
}
func isPlayer(e map[string]any) bool {
	name, _ := e["identifier"].(string)
	id, _ := e["id"].(string)
	if strings.EqualFold(name, "minecraft:player") || strings.EqualFold(name, "player") || strings.EqualFold(id, "player") || strings.EqualFold(id, "minecraft:player") {
		return true
	}
	_, ok := e["PlayerID"]
	return ok
}
func removeInventories(v any) {
	switch value := v.(type) {
	case map[string]any:
		for k, v := range value {
			if strings.EqualFold(k, "items") || strings.EqualFold(k, "inventory") || strings.EqualFold(k, "enderchestinventory") || k == "Armor" || k == "Mainhand" || k == "Offhand" {
				delete(value, k)
			} else {
				removeInventories(v)
			}
		}
	case []any:
		for _, v := range value {
			removeInventories(v)
		}
	case []map[string]any:
		for _, v := range value {
			removeInventories(v)
		}
	}
}
func WriteMCStructure(ctx context.Context, w io.Writer, d *Document, o ExportOptions) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if err := d.Validate(); err != nil {
		return err
	}
	layers := [2][]int32{make([]int32, int(d.Volume())), make([]int32, int(d.Volume()))}
	positionData := map[string]any{}
	var index int
	err := each(ctx, d.Bounds(), func(p Vec3) error {
		c := d.Cell(p)
		if !c.Known {
			if !o.OmitUnknown {
				return fmt.Errorf("Unrecorded cells remain; explicitly omit them or capture the missing area")
			}
			c.Layers = [2]int32{-1, -1}
		}
		layers[0][index], layers[1][index] = c.Layers[0], c.Layers[1]
		if o.BlockData {
			if data := d.BlockData[Key(p)]; data != nil {
				copy := cloneMap(data)
				if be := compound(copy["block_entity_data"]); len(be) > 0 {
					be["x"], be["y"], be["z"] = d.Origin[0]+p[0], d.Origin[1]+p[1], d.Origin[2]+p[2]
				}
				if !o.Inventories {
					removeInventories(copy)
				}
				positionData[strconv.Itoa(index)] = copy
			}
		}
		index++
		return nil
	})
	if err != nil {
		return err
	}
	palette := make([]map[string]any, len(d.Palette))
	for i, s := range d.Palette {
		palette[i] = map[string]any{"name": s.Name, "states": cloneMap(s.States), "version": s.Version}
	}
	entities := []map[string]any{}
	if o.Entities {
		for _, v := range d.Entities {
			if isPlayer(v) {
				continue
			}
			copy := cloneMap(v)
			p, ok := entityPosition(copy)
			if !ok {
				return fmt.Errorf("Invalid entity position")
			}
			for i := 0; i < 3; i++ {
				p[i] += float64(d.Origin[i])
			}
			setEntityPosition(copy, p)
			if !o.Inventories {
				removeInventories(copy)
			}
			entities = append(entities, copy)
		}
	}
	root := cloneMap(d.RootExtra)
	root["format_version"] = int32(1)
	root["size"] = []int32{d.Size[0], d.Size[1], d.Size[2]}
	root["structure_world_origin"] = []int32{d.Origin[0], d.Origin[1], d.Origin[2]}
	data := cloneMap(d.StructureExtra)
	data["block_indices"] = [][]int32{layers[0], layers[1]}
	data["entities"] = entities
	p := cloneMap(d.PaletteExtra)
	p["block_palette"] = palette
	p["block_position_data"] = positionData
	data["palette"] = map[string]any{"default": p}
	root["structure"] = data
	if err := ctx.Err(); err != nil {
		return err
	}
	return nbt.NewEncoderWithEncoding(&contextWriter{ctx, w}, nbt.LittleEndian).Encode(root)
}
