package structure

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
)

type Edit struct {
	Kind             string   `json:"kind"`
	Name             string   `json:"name,omitempty"`
	Bounds           []Bounds `json:"bounds,omitempty"`
	Block            State    `json:"block"`
	From             string   `json:"from,omitempty"`
	MatchStates      bool     `json:"matchStates,omitempty"`
	Destination      Vec3     `json:"destination"`
	Turns            int      `json:"turns"`
	Mirror           string   `json:"mirror,omitempty"`
	Anchor           *Vec3    `json:"anchor,omitempty"`
	Regions          []Region `json:"regions,omitempty"`
	AllowUnsupported bool     `json:"allowUnsupported,omitempty"`
}
type History struct {
	Doc        *Document
	Clipboard  *Document
	undo, redo []*Document
	limit      int
}

func NewHistory(d *Document) *History { return &History{Doc: d, limit: 64} }
func (h *History) Fork() *History {
	out := *h
	out.undo = append([]*Document{}, h.undo...)
	out.redo = append([]*Document{}, h.redo...)
	return &out
}
func (h *History) CanUndo() bool { return len(h.undo) > 0 }
func (h *History) CanRedo() bool { return len(h.redo) > 0 }
func (h *History) Undo() bool {
	if len(h.undo) == 0 {
		return false
	}
	h.redo = append(h.redo, h.Doc)
	h.Doc = h.undo[len(h.undo)-1]
	h.undo = h.undo[:len(h.undo)-1]
	return true
}
func (h *History) Redo() bool {
	if len(h.redo) == 0 {
		return false
	}
	h.undo = append(h.undo, h.Doc)
	h.Doc = h.redo[len(h.redo)-1]
	h.redo = h.redo[:len(h.redo)-1]
	return true
}
func (h *History) Commit(d *Document) {
	h.undo = append(h.undo, h.Doc)
	h.Doc = d
	h.redo = nil
	h.trim()
}
func (h *History) trim() {
	// COW sections are shared between versions. Bound history by the actual
	// unique section allocations, while retaining one undo for a large action.
	for len(h.undo) > 1 {
		seen := map[*Section]bool{}
		for _, d := range append(append([]*Document{}, h.undo...), h.Doc) {
			for _, s := range d.Sections {
				seen[s] = true
			}
		}
		if len(h.undo) <= h.limit && int64(len(seen))*33280 <= 256<<20 {
			break
		}
		h.undo = h.undo[1:]
	}
}
func checkedBounds(d *Document, boxes []Bounds) ([]Bounds, error) {
	if len(boxes) == 0 {
		return []Bounds{d.Bounds()}, nil
	}
	if len(boxes) > 128 {
		return nil, fmt.Errorf("At most 128 selection boxes are supported")
	}
	out := make([]Bounds, len(boxes))
	for i, b := range boxes {
		b = b.Normalized()
		if !d.Bounds().Contains(b.Min) || !d.Bounds().Contains(b.Max) {
			return nil, fmt.Errorf("Selection is outside the structure")
		}
		out[i] = b
	}
	return out, nil
}
func selected(boxes []Bounds, p Vec3) bool {
	for _, b := range boxes {
		if b.Contains(p) {
			return true
		}
	}
	return false
}
func enclosing(boxes []Bounds) Bounds {
	b := boxes[0]
	for _, v := range boxes[1:] {
		for i := 0; i < 3; i++ {
			b.Min[i] = min(b.Min[i], v.Min[i])
			b.Max[i] = max(b.Max[i], v.Max[i])
		}
	}
	return b
}
func (h *History) Apply(ctx context.Context, e Edit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d := h.Doc.Clone()
	var cutClipboard *Document
	boxes, err := checkedBounds(d, e.Bounds)
	if err != nil {
		return err
	}
	switch e.Kind {
	case "fill", "erase", "replace":
		state := e.Block
		if e.Kind == "erase" {
			state = State{Name: "minecraft:air"}
		}
		index, err := d.AddState(state)
		if err != nil {
			return err
		}
		for boxIndex, b := range boxes {
			if err := each(ctx, b, func(p Vec3) error {
				key := Key(p)
				if selected(boxes[:boxIndex], p) {
					return nil
				}
				c := d.Cell(p)
				if e.Kind == "replace" {
					if !c.Known || c.Layers[0] < 0 {
						return nil
					}
					s := d.Palette[c.Layers[0]]
					if e.MatchStates {
						if StateKey(s) != e.From {
							return nil
						}
					} else if s.Name != e.From {
						return nil
					}
				}
				d.Set(p, Cell{Layers: [2]int32{index, -1}, Known: true})
				delete(d.BlockData, key)
				return nil
			}); err != nil {
				return err
			}
		}
	case "copy", "cut":
		clip, err := Extract(ctx, d, boxes, "Clipboard")
		if err != nil {
			return err
		}
		if e.Kind == "copy" {
			h.Clipboard = clip
			return nil
		}
		cutClipboard = clip
		air, _ := d.AddState(State{Name: "minecraft:air"})
		for _, b := range boxes {
			if err := each(ctx, b, func(p Vec3) error {
				d.Set(p, Cell{Layers: [2]int32{air, -1}, Known: true})
				delete(d.BlockData, Key(p))
				return nil
			}); err != nil {
				return err
			}
		}
		keptEntities := make([]map[string]any, 0, len(d.Entities))
		for _, entity := range d.Entities {
			p, ok := entityPosition(entity)
			if !ok || !selected(boxes, Vec3{int32(math.Floor(p[0])), int32(math.Floor(p[1])), int32(math.Floor(p[2]))}) {
				keptEntities = append(keptEntities, entity)
			}
		}
		d.Entities = keptEntities
	case "paste":
		if h.Clipboard == nil {
			return fmt.Errorf("Copy a selection first")
		}
		next, _, err := Place(ctx, h.Clipboard, d, e.Destination, "replace", true)
		if err != nil {
			return err
		}
		d = next
	case "rotate", "mirror":
		turns := e.Turns
		if e.Kind == "mirror" {
			turns = 0
		}
		next, warnings, err := Transform(ctx, d, turns, e.Mirror)
		if err != nil {
			return err
		}
		if len(warnings) > 0 && !e.AllowUnsupported {
			return fmt.Errorf("Transformation needs review: %s", strings.Join(warnings, "; "))
		}
		d = next
	case "rename":
		if err := ValidateName(e.Name); err != nil {
			return err
		}
		d.Name = e.Name
	case "anchor":
		if e.Anchor == nil || !d.Bounds().Contains(*e.Anchor) {
			return fmt.Errorf("Choose an anchor inside the structure")
		}
		d.Anchor = *e.Anchor
	case "regions":
		if len(e.Regions) > 128 {
			return fmt.Errorf("At most 128 named regions")
		}
		d.Regions = append([]Region{}, e.Regions...)
	default:
		return fmt.Errorf("Unknown structure edit %q", e.Kind)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.Validate(); err != nil {
		return err
	}
	h.Commit(d)
	if cutClipboard != nil {
		h.Clipboard = cutClipboard
	}
	return nil
}
func Extract(ctx context.Context, d *Document, boxes []Bounds, name string) (*Document, error) {
	boxes, err := checkedBounds(d, boxes)
	if err != nil {
		return nil, err
	}
	bounds := enclosing(boxes)
	out, err := New(name, bounds.Size())
	if err != nil {
		return nil, err
	}
	out.Default = Cell{Layers: [2]int32{-1, -1}}
	out.DefaultKnown = true
	out.Palette = append([]State(nil), d.Palette...)
	out.Packs = append([]Dependency(nil), d.Packs...)
	for i := 0; i < 3; i++ {
		out.Origin[i] = d.Origin[i] + bounds.Min[i]
	}
	for _, b := range boxes {
		if err := each(ctx, b, func(p Vec3) error {
			q := Vec3{p[0] - bounds.Min[0], p[1] - bounds.Min[1], p[2] - bounds.Min[2]}
			out.Set(q, d.Cell(p))
			if data := d.BlockData[Key(p)]; data != nil {
				out.BlockData[Key(q)] = cloneMap(data)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	for _, entity := range d.Entities {
		p, ok := entityPosition(entity)
		if !ok {
			continue
		}
		if selected(boxes, Vec3{int32(math.Floor(p[0])), int32(math.Floor(p[1])), int32(math.Floor(p[2]))}) {
			copy := cloneMap(entity)
			for i := 0; i < 3; i++ {
				p[i] -= float64(bounds.Min[i])
			}
			setEntityPosition(copy, p)
			out.Entities = append(out.Entities, copy)
		}
	}
	return out, nil
}

type Difference struct {
	Position Vec3   `json:"position"`
	Kind     string `json:"kind"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}
type Comparison struct {
	Correct     int64        `json:"correct"`
	Missing     int64        `json:"missing"`
	Extra       int64        `json:"extra"`
	Wrong       int64        `json:"wrong"`
	Orientation int64        `json:"orientation"`
	Unknown     int64        `json:"unknown"`
	Samples     []Difference `json:"samples"`
	Truncated   bool         `json:"truncated"`
}

func Compare(ctx context.Context, source, target *Document, offset Vec3, boxes []Bounds) (Comparison, error) {
	var out Comparison
	boxes, err := checkedBounds(source, boxes)
	if err != nil {
		return out, err
	}
	bounds := enclosing(boxes)
	err = each(ctx, bounds, func(p Vec3) error {
		if !selected(boxes, p) {
			return nil
		}
		a := source.Cell(p)
		if a.Known && a.Layers[0] < 0 {
			return nil
		}
		q := Vec3{p[0] + offset[0], p[1] + offset[1], p[2] + offset[2]}
		b := target.Cell(q)
		kind := ""
		expected, actual := "unknown", "unknown"
		if !a.Known || !b.Known {
			out.Unknown++
			kind = "unknown"
		} else {
			sa := source.Palette[a.Layers[0]]
			expected = sa.Name
			sb := State{Name: "minecraft:air"}
			if b.Layers[0] >= 0 {
				sb = target.Palette[b.Layers[0]]
			}
			actual = sb.Name
			equal := StateKey(sa) == StateKey(sb)
			for layer := 1; layer < 2; layer++ {
				skey, tkey := "", ""
				if a.Layers[layer] >= 0 {
					skey = StateKey(source.Palette[a.Layers[layer]])
				}
				if b.Layers[layer] >= 0 {
					tkey = StateKey(target.Palette[b.Layers[layer]])
				}
				equal = equal && skey == tkey
			}
			switch {
			case equal:
				out.Correct++
			case Air(sb) && !Air(sa):
				out.Missing++
				kind = "missing"
			case Air(sa) && !Air(sb):
				out.Extra++
				kind = "extra"
			case sa.Name == sb.Name:
				out.Orientation++
				kind = "state"
			default:
				out.Wrong++
				kind = "wrong"
			}
		}
		if kind != "" {
			if len(out.Samples) < 10000 {
				out.Samples = append(out.Samples, Difference{p, kind, expected, actual})
			} else {
				out.Truncated = true
			}
		}
		return nil
	})
	return out, err
}
func Place(ctx context.Context, source, target *Document, anchor Vec3, mode string, entities bool) (*Document, int64, error) {
	if mode != "replace" && mode != "air-only" && mode != "ignore-air" {
		return nil, 0, fmt.Errorf("Choose replace, air-only or ignore-air")
	}
	out := target.Clone()
	offset := Vec3{anchor[0] - source.Anchor[0], anchor[1] - source.Anchor[1], anchor[2] - source.Anchor[2]}
	remap := make([]int32, len(source.Palette))
	for i, s := range source.Palette {
		idx, err := out.AddState(s)
		if err != nil {
			return nil, 0, err
		}
		remap[i] = idx
	}
	var count int64
	err := each(ctx, source.Bounds(), func(p Vec3) error {
		a := source.Cell(p)
		if !a.Known {
			return fmt.Errorf("Selection contains unrecorded cells; resolve them before placing")
		}
		if a.Layers[0] < 0 {
			return nil
		}
		if mode == "ignore-air" && Air(source.Palette[a.Layers[0]]) {
			return nil
		}
		q := Vec3{p[0] + offset[0], p[1] + offset[1], p[2] + offset[2]}
		if !out.Bounds().Contains(q) {
			return fmt.Errorf("Placement extends outside target bounds")
		}
		b := out.Cell(q)
		if mode == "air-only" && (!b.Known || b.Layers[0] >= 0 && !Air(out.Palette[b.Layers[0]])) {
			return nil
		}
		c := a
		for layer := 0; layer < 2; layer++ {
			if a.Layers[layer] >= 0 {
				c.Layers[layer] = remap[a.Layers[layer]]
			}
		}
		out.Set(q, c)
		delete(out.BlockData, Key(q))
		if data := source.BlockData[Key(p)]; data != nil {
			out.BlockData[Key(q)] = cloneMap(data)
			if be, ok := out.BlockData[Key(q)]["block_entity_data"].(map[string]any); ok {
				delta := Vec3{}
				for i := range delta {
					delta[i] = target.Origin[i] + offset[i] - source.Origin[i]
				}
				translateBlockData(be, delta)
			}
		}
		count++
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	if entities {
		for _, v := range source.Entities {
			copy := cloneMap(v)
			p, ok := entityPosition(copy)
			if !ok {
				continue
			}
			for i := 0; i < 3; i++ {
				p[i] += float64(offset[i])
			}
			setEntityPosition(copy, p)
			delete(copy, "UniqueID")
			out.Entities = append(out.Entities, copy)
		}
	}
	deps := map[string]bool{}
	for _, p := range out.Packs {
		deps[p.ID] = true
	}
	for _, p := range source.Packs {
		if !deps[p.ID] {
			out.Packs = append(out.Packs, p)
			deps[p.ID] = true
		}
	}
	return out, count, nil
}
func entityPosition(e map[string]any) (Vec3f, bool) {
	v := reflectSlice(e["Pos"])
	if len(v) != 3 {
		return Vec3f{}, false
	}
	var p Vec3f
	for i := 0; i < 3; i++ {
		n, ok := number(v[i])
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return p, false
		}
		p[i] = n
	}
	return p, true
}

type Vec3f [3]float64

func setEntityPosition(e map[string]any, p Vec3f) {
	e["Pos"] = []float32{float32(p[0]), float32(p[1]), float32(p[2])}
}
func reflectSlice(v any) []any {
	switch a := v.(type) {
	case []any:
		return a
	case []float32:
		o := make([]any, len(a))
		for i, v := range a {
			o[i] = v
		}
		return o
	case []float64:
		o := make([]any, len(a))
		for i, v := range a {
			o[i] = v
		}
		return o
	case []int32:
		o := make([]any, len(a))
		for i, v := range a {
			o[i] = v
		}
		return o
	}
	return nil
}
func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float32:
		return float64(x), true
	case float64:
		return x, true
	case int32:
		return float64(x), true
	case uint8:
		return float64(x), true
	}
	return 0, false
}
func uniqueWarnings(w map[string]bool) []string {
	out := make([]string, 0, len(w))
	for k := range w {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
