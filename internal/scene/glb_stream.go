package scene

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type GLBWriter struct {
	ctx                             context.Context
	bin                             *os.File
	size                            uint64
	views, accessors, meshes, nodes []any
	roots                           []int
}

func NewGLBWriter(ctx context.Context, directory string) (*GLBWriter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(directory, ".craftbox-glb-bin-*")
	if err != nil {
		return nil, err
	}
	return &GLBWriter{ctx: ctx, bin: f}, nil
}
func (g *GLBWriter) Close() error {
	if g.bin == nil {
		return nil
	}
	name := g.bin.Name()
	err := g.bin.Close()
	g.bin = nil
	removeErr := os.Remove(name)
	if err != nil {
		return err
	}
	return removeErr
}
func (g *GLBWriter) Bytes() uint64 { return g.size }
func (g *GLBWriter) write(data []byte) error {
	if g.size+uint64(len(data)) > math.MaxUint32-28 {
		return fmt.Errorf("GLB format exceeds 4 GiB; export a smaller region")
	}
	for len(data) > 0 {
		if err := g.ctx.Err(); err != nil {
			return err
		}
		n := min(len(data), 64<<10)
		written, err := g.bin.Write(data[:n])
		g.size += uint64(written)
		if err != nil {
			return err
		}
		if written != n {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
func (g *GLBWriter) align() error {
	if n := int((4 - g.size%4) % 4); n != 0 {
		return g.write(make([]byte, n))
	}
	return nil
}
func (g *GLBWriter) view(start uint64, target int) int {
	v := map[string]any{"buffer": 0, "byteOffset": start, "byteLength": g.size - start}
	if target != 0 {
		v["target"] = target
	}
	g.views = append(g.views, v)
	return len(g.views) - 1
}
func (g *GLBWriter) floats(values []float32, kind string, bounds bool) (int, error) {
	if err := g.align(); err != nil {
		return 0, err
	}
	start := g.size
	var buf [64 << 10]byte
	for offset := 0; offset < len(values); {
		n := min(len(values)-offset, len(buf)/4)
		for i, v := range values[offset : offset+n] {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return 0, fmt.Errorf("Non-finite GLB geometry")
			}
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
		}
		if err := g.write(buf[:n*4]); err != nil {
			return 0, err
		}
		offset += n
	}
	width := 3
	if kind == "VEC2" {
		width = 2
	}
	a := map[string]any{"bufferView": g.view(start, 34962), "componentType": 5126, "count": len(values) / width, "type": kind}
	if bounds {
		lo, hi := []float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}, []float32{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
		for i, v := range values {
			lo[i%3] = min(lo[i%3], v)
			hi[i%3] = max(hi[i%3], v)
		}
		a["min"] = lo
		a["max"] = hi
	}
	g.accessors = append(g.accessors, a)
	return len(g.accessors) - 1, nil
}
func (g *GLBWriter) AddChunk(c ChunkMesh) error {
	if err := g.ctx.Err(); err != nil {
		return err
	}
	primitives := []any{}
	for _, p := range c.Parts {
		if len(p.Positions) == 0 {
			continue
		}
		vertices := len(p.Positions) / 3
		if len(p.Positions)%3 != 0 || len(p.Normals) != len(p.Positions) || len(p.UVs) != vertices*2 || len(p.Indices)%3 != 0 {
			return fmt.Errorf("Invalid GLB geometry buffer sizes")
		}
		for _, i := range p.Indices {
			if uint64(i) >= uint64(vertices) {
				return fmt.Errorf("Invalid GLB vertex index")
			}
		}
		p, err := compactGLBPart(g.ctx, p)
		if err != nil {
			return err
		}
		position, err := g.floats(p.Positions, "VEC3", true)
		if err != nil {
			return err
		}
		normal, err := g.floats(p.Normals, "VEC3", false)
		if err != nil {
			return err
		}
		uv, err := g.floats(p.UVs, "VEC2", false)
		if err != nil {
			return err
		}
		indices, err := g.indices(p.Indices)
		if err != nil {
			return err
		}
		material := 0
		switch p.Mode {
		case "opaque":
		case "mask":
			material = 1
		case "blend":
			material = 2
		case "water":
			material = 5
		case "unlit", "billboard":
			material = 3
		case "unlit-blend":
			material = 4
		default:
			return fmt.Errorf("Unknown GLB material %q", p.Mode)
		}
		primitives = append(primitives, map[string]any{"attributes": map[string]int{"POSITION": position, "NORMAL": normal, "TEXCOORD_0": uv}, "indices": indices, "material": material, "mode": 4})
	}
	if len(primitives) == 0 {
		return nil
	}
	g.meshes = append(g.meshes, map[string]any{"name": fmt.Sprintf("chunk_%d_%d", c.X, c.Z), "primitives": primitives})
	g.roots = append(g.roots, len(g.nodes))
	g.nodes = append(g.nodes, map[string]any{"mesh": len(g.meshes) - 1, "translation": []float64{float64(c.X) * 16, 0, float64(c.Z) * 16}})
	return nil
}
func (g *GLBWriter) Finish(out io.Writer, atlas []byte, source string) error {
	if err := g.ctx.Err(); err != nil {
		return err
	}
	if len(g.meshes) == 0 {
		return fmt.Errorf("No captured 3D geometry available")
	}
	if len(atlas) == 0 {
		return fmt.Errorf("Texture atlas is missing")
	}
	if err := g.align(); err != nil {
		return err
	}
	start := g.size
	if err := g.write(atlas); err != nil {
		return err
	}
	imageView := g.view(start, 0)
	if err := g.align(); err != nil {
		return err
	}
	materials := []any{}
	for _, mode := range []string{"OPAQUE", "MASK", "BLEND"} {
		m := map[string]any{"name": strings.ToLower(mode), "alphaMode": mode, "doubleSided": mode != "OPAQUE", "pbrMetallicRoughness": map[string]any{"baseColorTexture": map[string]int{"index": 0}, "metallicFactor": 0, "roughnessFactor": 1}}
		if mode == "MASK" {
			m["alphaCutoff"] = .5
		}
		if mode == "BLEND" {
			m["pbrMetallicRoughness"].(map[string]any)["baseColorFactor"] = []float64{1, 1, 1, .72}
		}
		materials = append(materials, m)
	}
	for _, mode := range []string{"MASK", "BLEND"} {
		m := map[string]any{"name": "unlit", "alphaMode": mode, "doubleSided": true, "pbrMetallicRoughness": map[string]any{"baseColorTexture": map[string]int{"index": 0}, "metallicFactor": 0, "roughnessFactor": 1}, "extensions": map[string]any{"KHR_materials_unlit": map[string]any{}}}
		if mode == "MASK" {
			m["alphaCutoff"] = .05
		} else {
			m["name"] = "unlit-blend"
		}
		materials = append(materials, m)
	}
	doc := map[string]any{"asset": map[string]string{"version": "2.0", "generator": "CraftBox", "copyright": source}, "scene": 0, "scenes": []any{map[string]any{"nodes": g.roots}}, "nodes": g.nodes, "meshes": g.meshes, "materials": materials, "buffers": []any{map[string]uint64{"byteLength": g.size}}, "bufferViews": g.views, "accessors": g.accessors, "images": []any{map[string]any{"bufferView": imageView, "mimeType": "image/png"}}, "textures": []any{map[string]int{"sampler": 0, "source": 0}}, "samplers": []any{map[string]int{"magFilter": 9728, "minFilter": 9728, "wrapS": 33071, "wrapT": 33071}}, "extensionsUsed": []string{"KHR_materials_unlit"}}
	materials = append(materials, map[string]any{"name": "water", "alphaMode": "BLEND", "doubleSided": true, "pbrMetallicRoughness": map[string]any{"baseColorTexture": map[string]int{"index": 0}, "baseColorFactor": []float64{1, 1, 1, .85}, "metallicFactor": 0, "roughnessFactor": .2}})
	doc["materials"] = materials
	js, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}
	total := uint64(28+len(js)) + g.size
	if total > math.MaxUint32 {
		return fmt.Errorf("GLB format exceeds 4 GiB; export a smaller region")
	}
	var header [20]byte
	for i, v := range []uint32{0x46546c67, 2, uint32(total), uint32(len(js)), 0x4e4f534a} {
		binary.LittleEndian.PutUint32(header[i*4:], v)
	}
	w := glbContextWriter{g.ctx, out}
	if _, err = w.Write(header[:]); err != nil {
		return err
	}
	if _, err = io.CopyBuffer(w, strings.NewReader(string(js)), make([]byte, 64<<10)); err != nil {
		return err
	}
	var binHead [8]byte
	binary.LittleEndian.PutUint32(binHead[:], uint32(g.size))
	binary.LittleEndian.PutUint32(binHead[4:], 0x004e4942)
	if _, err = w.Write(binHead[:]); err != nil {
		return err
	}
	if _, err = g.bin.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = io.CopyBuffer(w, g.bin, make([]byte, 64<<10))
	return err
}

type glbContextWriter struct {
	ctx context.Context
	out io.Writer
}

func (w glbContextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.out.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}
