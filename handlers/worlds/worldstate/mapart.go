package worldstate

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const mapSize = 128

type MapFrame struct {
	Pos      cube.Pos
	MapID    int64
	Facing   int32
	Rotation float64
}

type chunkMaps struct {
	frames []MapFrame
	loose  []int64
}

func NewMap(id int64) *Map {
	return &Map{
		MapID:         id,
		ParentMapId:   -1,
		Width:         mapSize,
		Height:        mapSize,
		Decorations:   []any{},
		FullyExplored: true,
		MapLocked:     true,
	}
}

func (m *Map) PixelsComplete() bool {
	return m != nil && m.receivedPixels == mapSize*mapSize
}

func ApplyMapPacket(m *Map, pk *packet.ClientBoundMapItemData) *Map {
	var out *Map
	if m == nil {
		out = NewMap(pk.MapID)
	} else {
		c := *m
		out = &c
	}
	out.Dimension = pk.Dimension
	out.XCenter = pk.Origin.X()
	out.ZCenter = pk.Origin.Z()
	if scale, ok := pk.Scale.Value(); ok {
		out.Scale = scale
	}
	pixels, ok := pk.Pixels.Value()
	if !ok {
		return out
	}
	width, _ := pk.Width.Value()
	height, _ := pk.Height.Value()
	xOff, _ := pk.XOffset.Value()
	yOff, _ := pk.YOffset.Value()
	if width <= 0 || height <= 0 {
		return out
	}
	for y := max(int64(0), -int64(yOff)); y < min(int64(height), int64(mapSize)-int64(yOff)); y++ {
		for x := max(int64(0), -int64(xOff)); x < min(int64(width), int64(mapSize)-int64(xOff)); x++ {
			i := y*int64(width) + x
			px, py := x+int64(xOff), y+int64(yOff)
			if i >= int64(len(pixels)) || px < 0 || py < 0 || px >= mapSize || py >= mapSize {
				continue
			}
			c := pixels[i]
			out.HasPixels = true
			pixel := int(py*mapSize + px)
			word, bit := pixel/64, uint(pixel%64)
			if out.pixels[word]&(uint64(1)<<bit) == 0 {
				out.pixels[word] |= uint64(1) << bit
				out.receivedPixels++
			}
			o := int(py*mapSize+px) * 4
			out.Colors[o], out.Colors[o+1], out.Colors[o+2], out.Colors[o+3] = c.R, c.G, c.B, c.A
		}
	}
	return out
}

func mapUUID(v any) (int64, bool) {
	item, ok := v.(map[string]any)
	if !ok {
		return 0, false
	}

	if name, ok := item["Name"].(string); ok {
		if name = strings.TrimPrefix(name, "minecraft:"); name != "filled_map" && name != "map" {
			return 0, false
		}
	}
	tag, ok := item["tag"].(map[string]any)
	if !ok {
		return 0, false
	}
	switch id := tag["map_uuid"].(type) {
	case int64:
		return id, true
	case int32:
		return int64(id), true
	}
	return 0, false
}

func nbtList(v any) []any {
	switch l := v.(type) {
	case []any:
		return l
	case []map[string]any:
		out := make([]any, len(l))
		for i, m := range l {
			out[i] = m
		}
		return out
	}
	return nil
}

func (w *World) scanMapsLocked(pos world.ChunkPos, ch *Chunk) {
	cm := &chunkMaps{}
	for p, be := range ch.BlockEntities {
		switch id, _ := be["id"].(string); id {
		case "ItemFrame", "GlowItemFrame":
			if mid, ok := mapUUID(be["Item"]); ok {
				cm.frames = append(cm.frames, MapFrame{Pos: p, MapID: mid, Facing: w.frameFacing(ch, p), Rotation: mapFrameRotation(be["ItemRotation"])})
			}
		default:
			for _, it := range nbtList(be["Items"]) {
				if mid, ok := mapUUID(it); ok {
					cm.loose = append(cm.loose, mid)
				}
			}
		}
	}
	if w.mapRefs == nil {
		w.mapRefs = map[world.ChunkPos]*chunkMaps{}
	}
	if w.seenMaps == nil {
		w.seenMaps = map[int64]struct{}{}
	}
	if len(cm.frames) == 0 && len(cm.loose) == 0 {
		delete(w.mapRefs, pos)
		return
	}
	w.mapRefs[pos] = cm
	note := func(id int64) {
		if _, seen := w.seenMaps[id]; seen {
			return
		}
		w.seenMaps[id] = struct{}{}
		if w.OnMapSeen != nil {
			w.OnMapSeen(id)
		}
	}
	for _, f := range cm.frames {
		note(f.MapID)
	}
	for _, id := range cm.loose {
		note(id)
	}
}

func (w *World) frameFacing(ch *Chunk, p cube.Pos) int32 {
	if w.BlockRegistry == nil || p.Y() < w.dimRange.Min() || p.Y() > w.dimRange.Max() {
		return -1
	}
	rid := ch.Block(uint8(p.X()&15), int16(p.Y()), uint8(p.Z()&15), 0)
	_, props, ok := w.BlockRegistry.RuntimeIDToState(rid)
	if !ok {
		return -1
	}
	switch f := props["facing_direction"].(type) {
	case int32:
		return f
	case uint8:
		return int32(f)
	}
	return -1
}

func (w *World) referencedMapsLocked() map[int64]struct{} {
	out := make(map[int64]struct{})
	for _, cm := range w.mapRefs {
		for _, f := range cm.frames {
			out[f.MapID] = struct{}{}
		}
		for _, id := range cm.loose {
			out[id] = struct{}{}
		}
	}
	return out
}

func (w *World) importMapsLocked(known map[int64]*Map) {
	for id := range w.referencedMapsLocked() {
		if m, ok := known[id]; ok {
			w.memState.maps[id] = m
		}
	}
}

func mapImage(m *Map) *image.NRGBA {
	if m == nil || !m.HasPixels {
		return nil
	}
	img := image.NewNRGBA(image.Rect(0, 0, mapSize, mapSize))
	copy(img.Pix, m.Colors[:])

	return img
}

func layout(f MapFrame) (col, row int) {
	x, y, z := f.Pos.X(), f.Pos.Y(), f.Pos.Z()
	switch f.Facing {
	case 2:
		return -x, -y
	case 3:
		return x, -y
	case 4:
		return z, -y
	case 5:
		return -z, -y
	case 1:
		return x, z
	case 0:
		return x, -z
	}
	return 0, 0
}

func (w *World) ExportMapArt(dir string) (grids, tiles int, err error) {
	return w.ExportMapArtWhere(dir, nil)
}

func (w *World) ExportMapArtWhere(dir string, keep func(int32, int32) bool) (grids, tiles int, err error) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	images := make(map[int64]*image.NRGBA)
	refs := map[int64]struct{}{}
	for pos, cm := range w.mapRefs {
		if keep != nil && !keep(pos[0], pos[1]) {
			continue
		}
		for _, frame := range cm.frames {
			refs[frame.MapID] = struct{}{}
		}
		for _, id := range cm.loose {
			refs[id] = struct{}{}
		}
	}
	for id := range refs {
		if img := mapImage(w.memState.maps[id]); img != nil {
			images[id] = img
		}
	}
	var frames []MapFrame
	for pos, cm := range w.mapRefs {
		if keep == nil || keep(pos[0], pos[1]) {
			frames = append(frames, cm.frames...)
		}
	}
	return WriteMapArt(dir, images, frames)
}

func WriteMapArt(dir string, images map[int64]*image.NRGBA, frames []MapFrame) (grids, tiles int, err error) {

	if len(images) == 0 {
		return 0, 0, nil
	}

	tileDir := filepath.Join(dir, "tiles")
	if err := os.MkdirAll(tileDir, 0o777); err != nil {
		return 0, 0, err
	}
	for id, img := range images {
		if err := writePNG(filepath.Join(tileDir, fmt.Sprintf("%d.png", id)), img); err != nil {
			return grids, tiles, err
		}
		tiles++
	}

	byPos := make(map[cube.Pos]MapFrame)
	for _, f := range frames {
		if images[f.MapID] != nil {
			byPos[f.Pos] = f
		}
	}

	seen := make(map[cube.Pos]bool)
	positions := make([]cube.Pos, 0, len(byPos))
	for p := range byPos {
		positions = append(positions, p)
	}
	slices.SortFunc(positions, func(a, b cube.Pos) int {
		for i := range 3 {
			if a[i] != b[i] {
				return a[i] - b[i]
			}
		}
		return 0
	})
	for _, start := range positions {
		if seen[start] {
			continue
		}
		facing := byPos[start].Facing
		var wall []MapFrame
		queue := []cube.Pos{start}
		seen[start] = true
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			wall = append(wall, byPos[p])
			for _, d := range []cube.Pos{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
				n := p.Add(d)
				if (facing == 0 || facing == 1) && d.Y() != 0 || (facing == 2 || facing == 3) && d.Z() != 0 || (facing == 4 || facing == 5) && d.X() != 0 {
					continue
				}
				if f, ok := byPos[n]; ok && !seen[n] && f.Facing == facing {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
		if len(wall) < 2 {
			continue
		}
		minC, minR, maxC, maxR := 1<<31-1, 1<<31-1, -(1 << 31), -(1 << 31)
		for _, f := range wall {
			c, r := wallLayout(f, wall)
			minC, maxC = min(minC, c), max(maxC, c)
			minR, maxR = min(minR, r), max(maxR, r)
		}
		width, height := (int64(maxC)-int64(minC)+1)*mapSize, (int64(maxR)-int64(minR)+1)*mapSize
		if width <= 0 || height <= 0 || width > 16384 || height > 16384 || width*height > 64*1024*1024 {
			return grids, tiles, fmt.Errorf("map wall exceeds image size limit")
		}
		canvas := image.NewNRGBA(image.Rect(0, 0, int(width), int(height)))
		for _, f := range wall {
			c, r := wallLayout(f, wall)
			at := image.Pt((c-minC)*mapSize, (r-minR)*mapSize)
			draw.Draw(canvas, image.Rectangle{Min: at, Max: at.Add(image.Pt(mapSize, mapSize))}, rotateMapImage(images[f.MapID], f.Rotation), image.Point{}, draw.Src)
		}
		name := fmt.Sprintf("grid_x%d_y%d_z%d.png", start.X(), start.Y(), start.Z())
		if err := writePNG(filepath.Join(dir, name), canvas); err != nil {
			return grids, tiles, err
		}
		grids++
	}
	return grids, tiles, nil
}

func wallLayout(f MapFrame, wall []MapFrame) (col, row int) {
	if f.Facing >= 0 && f.Facing <= 5 {
		return layout(f)
	}
	xs, zs := map[int]bool{}, map[int]bool{}
	for _, o := range wall {
		xs[o.Pos.X()] = true
		zs[o.Pos.Z()] = true
	}
	if len(xs) >= len(zs) {
		return f.Pos.X(), -f.Pos.Y()
	}
	return f.Pos.Z(), -f.Pos.Y()
}

func writePNG(name string, img image.Image) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func mapFrameRotation(value any) float64 {
	var angle float64
	degrees := false
	switch value := value.(type) {
	case byte:
		angle = float64(value)
	case int:
		angle = float64(value)
	case int8:
		angle = float64(value)
	case int16:
		angle = float64(value)
	case int32:
		angle = float64(value)
	case int64:
		angle = float64(value)
	case float32:
		angle = float64(value)
		degrees = true
	case float64:
		angle = value
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return 0
	}
	if !degrees && math.Abs(angle) <= 7 && math.Trunc(angle) == angle {
		angle *= 45
	}
	return math.Mod(angle, 360)
}

func rotateMapImage(source *image.NRGBA, degrees float64) *image.NRGBA {
	if degrees == 0 {
		return source
	}
	rotated := image.NewNRGBA(source.Bounds())
	sine, cosine := math.Sincos(degrees * math.Pi / 180)
	center := float64(mapSize-1) / 2
	for y := 0; y < mapSize; y++ {
		for x := 0; x < mapSize; x++ {
			dx, dy := float64(x)-center, float64(y)-center
			sx, sy := int(math.Round(cosine*dx+sine*dy+center)), int(math.Round(-sine*dx+cosine*dy+center))
			if sx >= 0 && sx < mapSize && sy >= 0 && sy < mapSize {
				rotated.SetNRGBA(x, y, source.NRGBAAt(sx, sy))
			}
		}
	}
	return rotated
}

func MapsIn(p cube.Pos, be map[string]any, facing func(cube.Pos) int32) (frames []MapFrame, loose []int64) {
	switch id, _ := be["id"].(string); id {
	case "ItemFrame", "GlowItemFrame":
		if mid, ok := mapUUID(be["Item"]); ok {
			frames = append(frames, MapFrame{Pos: p, MapID: mid, Facing: facing(p), Rotation: mapFrameRotation(be["ItemRotation"])})
		}
	default:
		for _, it := range nbtList(be["Items"]) {
			if mid, ok := mapUUID(it); ok {
				loose = append(loose, mid)
			}
		}
	}
	return frames, loose
}

func FacingDirection(props map[string]any) int32 {
	switch f := props["facing_direction"].(type) {
	case int32:
		return f
	case uint8:
		return int32(f)
	}
	return -1
}

func MapImage(m *Map) *image.NRGBA { return mapImage(m) }
