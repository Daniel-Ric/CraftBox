package modelexport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
)

type Options struct {
	Format   string  `json:"format"`
	Scale    float64 `json:"scale"`
	Origin   string  `json:"origin"`
	UpAxis   string  `json:"upAxis"`
	Textures *bool   `json:"textures,omitempty"`
}

func (o Options) WithTextures() bool { return o.Textures == nil || *o.Textures }
func (o Options) Validate() error {
	if _, err := Extension(o.Format); err != nil {
		return err
	}
	if math.IsNaN(o.Scale) || math.IsInf(o.Scale, 0) || o.Scale < 0.000001 || o.Scale > 1000000 {
		return fmt.Errorf("Model scale must be between 0.000001 and 1000000")
	}
	if o.Origin != "" && o.Origin != "world" && o.Origin != "center" {
		return fmt.Errorf("Choose world or centered origin")
	}
	if o.UpAxis != "" && o.UpAxis != "y" && o.UpAxis != "z" {
		return fmt.Errorf("Choose Y-up or Z-up")
	}
	if (o.Format == "glb" || o.Format == "gltf") && o.UpAxis == "z" {
		return fmt.Errorf("glTF uses Y-up coordinates")
	}
	return nil
}
func Extension(format string) (string, error) {
	switch format {
	case "glb", "stl":
		return "." + format, nil
	case "gltf", "obj", "fbx":
		return "." + format + ".zip", nil
	}
	return "", fmt.Errorf("Unsupported model format %q", format)
}

type accessor struct {
	BufferView    int    `json:"bufferView"`
	ByteOffset    int64  `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
	Min, Max      []float64
}
type bufferView struct {
	ByteOffset int64 `json:"byteOffset"`
	ByteLength int64 `json:"byteLength"`
	ByteStride int   `json:"byteStride"`
}
type primitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    int            `json:"indices"`
	Material   int            `json:"material"`
	Mode       *int           `json:"mode"`
}
type document struct {
	Nodes []struct {
		Mesh        *int      `json:"mesh"`
		Translation []float64 `json:"translation"`
	} `json:"nodes"`
	Meshes []struct {
		Name       string      `json:"name"`
		Primitives []primitive `json:"primitives"`
	} `json:"meshes"`
	Accessors   []accessor   `json:"accessors"`
	BufferViews []bufferView `json:"bufferViews"`
	Images      []struct {
		BufferView int `json:"bufferView"`
	} `json:"images"`
	Materials []struct {
		Name      string `json:"name"`
		AlphaMode string `json:"alphaMode"`
		PBR       struct {
			BaseColorFactor []float64 `json:"baseColorFactor"`
		} `json:"pbrMetallicRoughness"`
	} `json:"materials"`
	Scenes []struct {
		Nodes []int `json:"nodes"`
	} `json:"scenes"`
}
type model struct {
	ctx                  context.Context
	file                 *os.File
	doc                  document
	raw                  map[string]any
	binOffset, binLength int64
	options              Options
	center               [3]float64
}
type meshPart struct {
	Name                    string
	Positions, Normals, UVs []float32
	Indices                 []uint32
	Translation             [3]float64
	Material                int
}

func openModel(ctx context.Context, file *os.File, options Options) (*model, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := options.Validate(); err != nil {
		return nil, err
	}
	var header [20]byte
	if _, err := file.ReadAt(header[:], 0); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(header[:]) != 0x46546c67 || binary.LittleEndian.Uint32(header[4:]) != 2 || binary.LittleEndian.Uint32(header[16:]) != 0x4e4f534a {
		return nil, fmt.Errorf("Invalid intermediate GLB")
	}
	size := int64(binary.LittleEndian.Uint32(header[12:]))
	if size < 2 || size > 64<<20 {
		return nil, fmt.Errorf("Model metadata exceeds 64 MiB")
	}
	raw := make([]byte, size)
	if _, err := file.ReadAt(raw, 20); err != nil {
		return nil, err
	}
	m := &model{ctx: ctx, file: file, options: options}
	if err := json.Unmarshal(raw, &m.doc); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&m.raw); err != nil {
		return nil, err
	}
	var bin [8]byte
	if _, err := file.ReadAt(bin[:], 20+size); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(bin[4:]) != 0x004e4942 {
		return nil, fmt.Errorf("Model binary data is missing")
	}
	m.binOffset = 28 + size
	m.binLength = int64(binary.LittleEndian.Uint32(bin[:]))
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if m.binOffset+m.binLength > info.Size() {
		return nil, fmt.Errorf("Truncated model binary")
	}
	for _, view := range m.doc.BufferViews {
		if view.ByteOffset < 0 || view.ByteLength < 0 || view.ByteOffset > m.binLength || view.ByteLength > m.binLength-view.ByteOffset {
			return nil, fmt.Errorf("Invalid model buffer range")
		}
	}
	for _, node := range m.doc.Nodes {
		if node.Mesh != nil && (*node.Mesh < 0 || *node.Mesh >= len(m.doc.Meshes)) {
			return nil, fmt.Errorf("Invalid model mesh")
		}
	}
	if options.Origin == "center" {
		low, high := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}, [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
		for _, node := range m.doc.Nodes {
			if node.Mesh == nil {
				continue
			}
			if *node.Mesh < 0 || *node.Mesh >= len(m.doc.Meshes) {
				return nil, fmt.Errorf("Invalid model mesh")
			}
			for _, p := range m.doc.Meshes[*node.Mesh].Primitives {
				index, ok := p.Attributes["POSITION"]
				if !ok || index < 0 || index >= len(m.doc.Accessors) {
					return nil, fmt.Errorf("Model positions missing")
				}
				a := m.doc.Accessors[index]
				if len(a.Min) != 3 || len(a.Max) != 3 {
					return nil, fmt.Errorf("Model bounds missing")
				}
				for axis := 0; axis < 3; axis++ {
					offset := float64(0)
					if len(node.Translation) == 3 {
						offset = node.Translation[axis]
					}
					low[axis] = math.Min(low[axis], a.Min[axis]+offset)
					high[axis] = math.Max(high[axis], a.Max[axis]+offset)
				}
			}
		}
		for axis := range m.center {
			m.center[axis] = (low[axis] + high[axis]) / 2
			if math.IsNaN(m.center[axis]) || math.IsInf(m.center[axis], 0) {
				return nil, fmt.Errorf("Model has no finite bounds")
			}
		}
	}
	return m, nil
}
func (m *model) color(index int) [4]float64 {
	result := [4]float64{1, 1, 1, 1}
	factor := m.doc.Materials[index].PBR.BaseColorFactor
	if len(factor) == 4 {
		copy(result[:], factor)
	}
	if m.doc.Materials[index].AlphaMode == "OPAQUE" || m.doc.Materials[index].AlphaMode == "" {
		result[3] = 1
	}
	return result
}
func (m *model) writeOpacity(archive *zip.Writer) error {

	raw, err := m.atlas()
	if err != nil {
		return err
	}
	source, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	bounds := source.Bounds()
	opacity := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if y%32 == 0 {
			if err := m.ctx.Err(); err != nil {
				return err
			}
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := source.At(x, y).RGBA()
			opacity.SetGray(x, y, color.Gray{Y: uint8(a >> 8)})
		}
	}
	out, err := archive.Create("opacity.png")
	if err != nil {
		return err
	}
	return png.Encode(contextWriter{ctx: m.ctx, out: out}, opacity)
}

func (m *model) accessorBytes(index, width int) (accessor, []byte, error) {
	if index < 0 || index >= len(m.doc.Accessors) {
		return accessor{}, nil, fmt.Errorf("Invalid model accessor")
	}
	a := m.doc.Accessors[index]
	if a.BufferView < 0 || a.BufferView >= len(m.doc.BufferViews) || a.Count < 0 || a.Count > 16000000 {
		return a, nil, fmt.Errorf("Invalid model accessor range")
	}
	view := m.doc.BufferViews[a.BufferView]
	size := int64(a.Count) * int64(width)
	if size > 256<<20 || a.ByteOffset < 0 || a.ByteOffset > view.ByteLength || size > view.ByteLength-a.ByteOffset || (view.ByteStride != 0 && view.ByteStride != width) {
		return a, nil, fmt.Errorf("Unsupported or oversized model accessor")
	}
	raw := make([]byte, size)
	_, err := m.file.ReadAt(raw, m.binOffset+view.ByteOffset+a.ByteOffset)
	return a, raw, err
}
func (m *model) floats(index, components int) ([]float32, error) {
	a, raw, err := m.accessorBytes(index, components*4)
	if err != nil {
		return nil, err
	}
	if a.ComponentType != 5126 || a.Type != fmt.Sprintf("VEC%d", components) {
		return nil, fmt.Errorf("Unsupported model float component")
	}
	values := make([]float32, a.Count*components)
	for i := range values {
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		if math.IsNaN(float64(values[i])) || math.IsInf(float64(values[i]), 0) {
			return nil, fmt.Errorf("Non-finite model geometry")
		}
	}
	return values, nil
}
func (m *model) indices(index int) ([]uint32, error) {
	if index < 0 || index >= len(m.doc.Accessors) {
		return nil, fmt.Errorf("Invalid model index accessor")
	}
	kind := m.doc.Accessors[index].ComponentType
	width := map[int]int{5121: 1, 5123: 2, 5125: 4}[kind]
	if width == 0 {
		return nil, fmt.Errorf("Unsupported model indices")
	}
	a, raw, err := m.accessorBytes(index, width)
	if err != nil {
		return nil, err
	}
	if a.Type != "SCALAR" || a.Count%3 != 0 {
		return nil, fmt.Errorf("Model must contain triangles")
	}
	values := make([]uint32, a.Count)
	for i := range values {
		switch width {
		case 1:
			values[i] = uint32(raw[i])
		case 2:
			values[i] = uint32(binary.LittleEndian.Uint16(raw[i*2:]))
		case 4:
			values[i] = binary.LittleEndian.Uint32(raw[i*4:])
		}
	}
	return values, nil
}
func (m *model) parts(visit func(meshPart) error, progress func(int, int)) error {
	total := 0
	for _, mesh := range m.doc.Meshes {
		total += len(mesh.Primitives)
	}
	done := 0
	for _, node := range m.doc.Nodes {
		if err := m.ctx.Err(); err != nil {
			return err
		}
		if node.Mesh == nil {
			continue
		}
		if *node.Mesh < 0 || *node.Mesh >= len(m.doc.Meshes) {
			return fmt.Errorf("Invalid model mesh reference")
		}
		mesh := m.doc.Meshes[*node.Mesh]
		for partIndex, p := range mesh.Primitives {
			if p.Mode != nil && *p.Mode != 4 {
				return fmt.Errorf("Model primitive is not a triangle mesh")
			}
			for _, key := range []string{"POSITION", "NORMAL", "TEXCOORD_0"} {
				if _, ok := p.Attributes[key]; !ok {
					return fmt.Errorf("Model attribute %s is missing", key)
				}
			}
			positions, err := m.floats(p.Attributes["POSITION"], 3)
			if err != nil {
				return err
			}
			normals, err := m.floats(p.Attributes["NORMAL"], 3)
			if err != nil {
				return err
			}
			uvs, err := m.floats(p.Attributes["TEXCOORD_0"], 2)
			if err != nil {
				return err
			}
			indices, err := m.indices(p.Indices)
			if err != nil {
				return err
			}
			if len(normals) != len(positions) || len(uvs)*3 != len(positions)*2 {
				return fmt.Errorf("Inconsistent model attribute sizes")
			}
			for _, index := range indices {
				if uint64(index) >= uint64(len(positions)/3) {
					return fmt.Errorf("Model index exceeds vertices")
				}
			}
			part := meshPart{Name: fmt.Sprintf("%s_part_%d", mesh.Name, partIndex), Positions: positions, Normals: normals, UVs: uvs, Indices: indices, Material: p.Material}
			if len(node.Translation) == 3 {
				copy(part.Translation[:], node.Translation)
			}
			if err = visit(part); err != nil {
				return err
			}
			done++
			if progress != nil {
				progress(done, total)
			}
		}
	}
	return nil
}
func (m *model) point(p meshPart, i int) [3]float64 {
	var out [3]float64
	for axis := range out {
		out[axis] = (float64(p.Positions[i*3+axis]) + p.Translation[axis] - m.center[axis]) * m.options.Scale
	}
	if m.options.UpAxis == "z" {
		out = [3]float64{out[0], -out[2], out[1]}
	}
	return out
}
func (m *model) normal(p meshPart, i int) [3]float64 {
	out := [3]float64{float64(p.Normals[i*3]), float64(p.Normals[i*3+1]), float64(p.Normals[i*3+2])}
	if m.options.UpAxis == "z" {
		out = [3]float64{out[0], -out[2], out[1]}
	}
	return out
}
func (m *model) atlas() ([]byte, error) {
	if len(m.doc.Images) == 0 {
		return nil, fmt.Errorf("Model texture missing")
	}
	index := m.doc.Images[0].BufferView
	if index < 0 || index >= len(m.doc.BufferViews) {
		return nil, fmt.Errorf("Invalid texture view")
	}
	view := m.doc.BufferViews[index]
	if view.ByteLength > 64<<20 {
		return nil, fmt.Errorf("Model texture too large")
	}
	raw := make([]byte, view.ByteLength)
	_, err := m.file.ReadAt(raw, m.binOffset+view.ByteOffset)
	return raw, err
}

func Convert(ctx context.Context, source *os.File, out io.Writer, options Options, progress func(int, int)) error {
	m, err := openModel(ctx, source, options)
	if err != nil {
		return err
	}
	if options.Textures == nil && len(m.doc.Images) == 0 {
		disabled := false
		m.options.Textures = &disabled
		options = m.options
	}
	writer := contextWriter{ctx, out}
	switch options.Format {
	case "glb":
		return m.writeGLTF(writer, nil)
	case "stl":
		return m.writeSTL(writer, progress)
	}
	archive := zip.NewWriter(writer)
	switch options.Format {
	case "gltf":
		err = m.writeGLTF(nil, archive)
	case "obj":
		err = m.writeOBJ(archive, progress)
	case "fbx":
		var entry io.Writer
		entry, err = archive.Create("scene.fbx")
		if err == nil {
			err = m.writeFBX(entry, progress)
		}
		if err == nil && options.WithTextures() {
			var texture []byte
			texture, err = m.atlas()
			if err == nil {
				var target io.Writer
				target, err = archive.Create("atlas.png")
				if err == nil {
					_, err = target.Write(texture)
				}
			}
		}
	}
	if err == nil && options.Format == "fbx" && options.WithTextures() {
		err = m.writeOpacity(archive)
	}
	if err != nil {
		_ = archive.Close()
		return err
	}
	return archive.Close()
}

type contextWriter struct {
	ctx context.Context
	out io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.out.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}
