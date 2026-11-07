package modelexport

// The lossless pass changes resource references and storage only. Attribute
// samples, UVs, transforms, material settings and PNG scanlines are untouched.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

var glbResourceOrder = []string{"bufferViews", "accessors", "images", "samplers", "textures", "materials", "meshes", "skins", "cameras"}
var safeGLBExtensions = map[string]bool{"KHR_materials_unlit": true, "KHR_texture_transform": true, "KHR_mesh_quantization": true}

type glbRange struct{ offset, length int64 }
type glbOptimizer struct {
	m        *model
	used     map[string][]bool
	remap    map[string][]int
	temp     *os.File
	size     int64
	packed   map[[32]byte][]glbRange
	pngViews map[int]bool
	buffer   []byte
}

func glbObject(v any) map[string]any { m, _ := v.(map[string]any); return m }
func glbArray(v any) []any           { a, _ := v.([]any); return a }
func glbIndex(v any) (int, error) {
	switch n := v.(type) {
	case json.Number:
		i, e := n.Int64()
		if e == nil && i >= 0 && uint64(i) <= uint64(math.MaxInt) {
			return int(i), nil
		}
	case float64:
		if n >= 0 && n < float64(math.MaxInt) && math.Trunc(n) == n {
			return int(n), nil
		}
	case int:
		if n >= 0 {
			return n, nil
		}
	}
	return 0, fmt.Errorf("Invalid glTF resource index")
}
func safeGLBLayout(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for key, value := range x {
			if key == "extensions" {
				for name := range glbObject(value) {
					if !safeGLBExtensions[name] {
						return false
					}
				}
			}
			if key == "extensionsUsed" || key == "extensionsRequired" {
				for _, name := range glbArray(value) {
					s, ok := name.(string)
					if !ok || !safeGLBExtensions[s] {
						return false
					}
				}
			}
			if !safeGLBLayout(value) {
				return false
			}
		}
	case []any:
		for _, value := range x {
			if !safeGLBLayout(value) {
				return false
			}
		}
	}
	return true
}
func (m *model) writeLosslessGLB(out io.Writer) error {
	// Unknown extensions may contain their own buffer offsets or resource IDs.
	// Preserve that layout rather than guessing how to rewrite opaque data.
	if !safeGLBLayout(m.raw) || len(glbArray(m.raw["buffers"])) != 1 {
		return writeGLBDocument(m.ctx, out, m.raw, m.file, m.binOffset, m.binLength)
	}
	o := &glbOptimizer{m: m, used: map[string][]bool{}, remap: map[string][]int{}, packed: map[[32]byte][]glbRange{}, pngViews: map[int]bool{}, buffer: make([]byte, 64<<10)}
	for _, table := range glbResourceOrder {
		o.used[table] = make([]bool, len(glbArray(m.raw[table])))
	}
	for _, node := range glbArray(m.raw["nodes"]) {
		n := glbObject(node)
		for field, table := range map[string]string{"mesh": "meshes", "skin": "skins", "camera": "cameras"} {
			if e := o.markRef(n, field, table); e != nil {
				return e
			}
		}
	}
	for _, animation := range glbArray(m.raw["animations"]) {
		for _, sampler := range glbArray(glbObject(animation)["samplers"]) {
			for _, key := range []string{"input", "output"} {
				if e := o.markRef(glbObject(sampler), key, "accessors"); e != nil {
					return e
				}
			}
		}
	}
	// Recompression is safe only for views used exclusively as PNG images.
	for i, value := range glbArray(m.raw["images"]) {
		if !o.used["images"][i] {
			continue
		}
		image := glbObject(value)
		if view, ok := image["bufferView"]; ok {
			index, e := glbIndex(view)
			if e != nil {
				return e
			}
			png := image["mimeType"] == "image/png"
			previous, seen := o.pngViews[index]
			o.pngViews[index] = png && (!seen || previous)
		}
	}
	for i, value := range glbArray(m.raw["accessors"]) {
		if !o.used["accessors"][i] {
			continue
		}
		a := glbObject(value)
		o.excludePNG(a, "bufferView")
		sparse := glbObject(a["sparse"])
		o.excludePNG(glbObject(sparse["indices"]), "bufferView")
		o.excludePNG(glbObject(sparse["values"]), "bufferView")
	}
	f, e := os.CreateTemp("", "craftbox-glb-lossless-*.bin")
	if e != nil {
		return e
	}
	o.temp = f
	defer func() { f.Close(); os.Remove(f.Name()) }()
	for _, table := range glbResourceOrder {
		if e := o.compact(table); e != nil {
			return e
		}
	}
	for _, node := range glbArray(m.raw["nodes"]) {
		n := glbObject(node)
		for field, table := range map[string]string{"mesh": "meshes", "skin": "skins", "camera": "cameras"} {
			if e := o.rewriteRef(n, field, table); e != nil {
				return e
			}
		}
	}
	for _, animation := range glbArray(m.raw["animations"]) {
		for _, sampler := range glbArray(glbObject(animation)["samplers"]) {
			for _, key := range []string{"input", "output"} {
				if e := o.rewriteRef(glbObject(sampler), key, "accessors"); e != nil {
					return e
				}
			}
		}
	}
	if o.size > 0 {
		buffer := glbObject(glbArray(m.raw["buffers"])[0])
		buffer["byteLength"] = o.size
		m.raw["buffers"] = []any{buffer}
	} else {
		delete(m.raw, "buffers")
	}
	return writeGLBDocument(m.ctx, out, m.raw, f, 0, o.size)
}
func (o *glbOptimizer) excludePNG(m map[string]any, key string) {
	if value, ok := m[key]; ok {
		if i, e := glbIndex(value); e == nil {
			o.pngViews[i] = false
		}
	}
}
func (o *glbOptimizer) markRef(m map[string]any, key, table string) error {
	if value, ok := m[key]; ok {
		return o.mark(table, value)
	}
	return nil
}
func (o *glbOptimizer) mark(table string, value any) error {
	if e := o.m.ctx.Err(); e != nil {
		return e
	}
	i, e := glbIndex(value)
	if e != nil || i >= len(o.used[table]) {
		return fmt.Errorf("Invalid glTF %s reference", table)
	}
	if o.used[table][i] {
		return nil
	}
	o.used[table][i] = true
	m := glbObject(glbArray(o.m.raw[table])[i])
	if m == nil {
		return fmt.Errorf("Invalid glTF %s object", table)
	}
	switch table {
	case "meshes":
		for _, raw := range glbArray(m["primitives"]) {
			p := glbObject(raw)
			for _, v := range glbObject(p["attributes"]) {
				if e = o.mark("accessors", v); e != nil {
					return e
				}
			}
			for _, target := range glbArray(p["targets"]) {
				for _, v := range glbObject(target) {
					if e = o.mark("accessors", v); e != nil {
						return e
					}
				}
			}
			if e = o.markRef(p, "indices", "accessors"); e != nil {
				return e
			}
			if e = o.markRef(p, "material", "materials"); e != nil {
				return e
			}
		}
	case "materials":
		return o.textureRefs(m, false)
	case "textures":
		if e = o.markRef(m, "source", "images"); e != nil {
			return e
		}
		return o.markRef(m, "sampler", "samplers")
	case "images":
		return o.markRef(m, "bufferView", "bufferViews")
	case "accessors":
		if e = o.markRef(m, "bufferView", "bufferViews"); e != nil {
			return e
		}
		sparse := glbObject(m["sparse"])
		for _, key := range []string{"indices", "values"} {
			if e = o.markRef(glbObject(sparse[key]), "bufferView", "bufferViews"); e != nil {
				return e
			}
		}
	case "skins":
		return o.markRef(m, "inverseBindMatrices", "accessors")
	}
	return nil
}
func (o *glbOptimizer) textureRefs(m map[string]any, rewrite bool) error {
	for key, value := range m {
		if key == "extras" {
			continue
		}
		child := glbObject(value)
		if child == nil {
			continue
		}
		if strings.HasSuffix(strings.ToLower(key), "texture") {
			var e error
			if rewrite {
				e = o.rewriteRef(child, "index", "textures")
			} else {
				e = o.markRef(child, "index", "textures")
			}
			if e != nil {
				return e
			}
		} else if e := o.textureRefs(child, rewrite); e != nil {
			return e
		}
	}
	return nil
}
func (o *glbOptimizer) rewriteRef(m map[string]any, key, table string) error {
	value, ok := m[key]
	if !ok {
		return nil
	}
	i, e := glbIndex(value)
	if e != nil || i >= len(o.remap[table]) || o.remap[table][i] < 0 {
		return fmt.Errorf("Invalid retained glTF %s reference", table)
	}
	m[key] = o.remap[table][i]
	return nil
}
func (o *glbOptimizer) compact(table string) error {
	values := glbArray(o.m.raw[table])
	mapping := make([]int, len(values))
	for i := range mapping {
		mapping[i] = -1
	}
	o.remap[table] = mapping
	result := []any{}
	seen := map[string]int{}
	for i, value := range values {
		if !o.used[table][i] {
			continue
		}
		if e := o.m.ctx.Err(); e != nil {
			return e
		}
		m := glbObject(value)
		var e error
		switch table {
		case "bufferViews":
			e = o.packView(i, m)
		case "accessors":
			e = o.rewriteRef(m, "bufferView", "bufferViews")
			if e == nil {
				for _, key := range []string{"indices", "values"} {
					if e = o.rewriteRef(glbObject(glbObject(m["sparse"])[key]), "bufferView", "bufferViews"); e != nil {
						break
					}
				}
			}
		case "images":
			e = o.rewriteRef(m, "bufferView", "bufferViews")
		case "textures":
			e = o.rewriteRef(m, "source", "images")
			if e == nil {
				e = o.rewriteRef(m, "sampler", "samplers")
			}
		case "materials":
			e = o.textureRefs(m, true)
		case "meshes":
			for _, raw := range glbArray(m["primitives"]) {
				p := glbObject(raw)
				for key := range glbObject(p["attributes"]) {
					if e = o.rewriteRef(glbObject(p["attributes"]), key, "accessors"); e != nil {
						break
					}
				}
				if e != nil {
					break
				}
				for _, target := range glbArray(p["targets"]) {
					for key := range glbObject(target) {
						if e = o.rewriteRef(glbObject(target), key, "accessors"); e != nil {
							break
						}
					}
					if e != nil {
						break
					}
				}
				if e != nil {
					break
				}
				if e = o.rewriteRef(p, "indices", "accessors"); e != nil {
					break
				}
				if e = o.rewriteRef(p, "material", "materials"); e != nil {
					break
				}
			}
		case "skins":
			e = o.rewriteRef(m, "inverseBindMatrices", "accessors")
		}
		if e != nil {
			return e
		}
		key, e := json.Marshal(m)
		if e != nil {
			return e
		}
		// Preserve mesh and material identities: even opaque coplanar surfaces
		// can render in a different order if renderer material IDs are merged.
		// Their underlying geometry, image and texture data still share storage.
		if id, ok := seen[string(key)]; ok && table != "meshes" && table != "materials" {
			mapping[i] = id
		} else {
			mapping[i] = len(result)
			seen[string(key)] = len(result)
			result = append(result, m)
		}
	}
	if len(result) > 0 {
		o.m.raw[table] = result
	} else {
		delete(o.m.raw, table)
	}
	return nil
}
func (o *glbOptimizer) packView(index int, m map[string]any) error {
	buffer, e := glbIndex(m["buffer"])
	if e != nil || buffer != 0 {
		return fmt.Errorf("Unsupported GLB buffer")
	}
	offset := 0
	if value, ok := m["byteOffset"]; ok {
		offset, e = glbIndex(value)
		if e != nil {
			return e
		}
	}
	n, e := glbIndex(m["byteLength"])
	if e != nil || int64(offset) > o.m.binLength || int64(n) > o.m.binLength-int64(offset) {
		return fmt.Errorf("Invalid GLB buffer range")
	}
	var source io.ReaderAt = io.NewSectionReader(o.m.file, o.m.binOffset+int64(offset), int64(n))
	length := int64(n)
	if o.pngViews[index] && n <= 64<<20 {
		raw := make([]byte, n)
		if _, e = source.ReadAt(raw, 0); e != nil {
			return e
		}
		optimized, err := losslessPNG(o.m.ctx, raw)
		if err != nil {
			return err
		}
		source = bytes.NewReader(optimized)
		length = int64(len(optimized))
	}
	hash := sha256.New()
	if _, e = io.CopyBuffer(contextWriter{o.m.ctx, hash}, io.NewSectionReader(source, 0, length), o.buffer); e != nil {
		return e
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	newOffset := int64(-1)
	for _, existing := range o.packed[digest] {
		if existing.length != length {
			continue
		}
		equal, err := equalGLBRange(o.m.ctx, source, o.temp, existing.offset, length)
		if err != nil {
			return err
		}
		if equal {
			newOffset = existing.offset
			break
		}
	}
	if newOffset < 0 {
		padding := (4 - o.size%4) % 4
		if padding > 0 {
			if e = o.write(make([]byte, padding)); e != nil {
				return e
			}
		}
		newOffset = o.size
		writer := glbPackedWriter{o}
		if _, e = io.CopyBuffer(writer, io.NewSectionReader(source, 0, length), o.buffer); e != nil {
			return e
		}
		o.packed[digest] = append(o.packed[digest], glbRange{newOffset, length})
	}
	m["buffer"] = 0
	m["byteOffset"] = newOffset
	m["byteLength"] = length
	return nil
}
func (o *glbOptimizer) write(p []byte) error {
	if e := o.m.ctx.Err(); e != nil {
		return e
	}
	n, e := o.temp.Write(p)
	o.size += int64(n)
	if e == nil && n != len(p) {
		e = io.ErrShortWrite
	}
	return e
}

type glbPackedWriter struct{ o *glbOptimizer }

func (w glbPackedWriter) Write(p []byte) (int, error) {
	before := w.o.size
	e := w.o.write(p)
	return int(w.o.size - before), e
}
func equalGLBRange(ctx context.Context, source, stored io.ReaderAt, offset, length int64) (bool, error) {
	var a, b [64 << 10]byte
	for at := int64(0); at < length; {
		if e := ctx.Err(); e != nil {
			return false, e
		}
		n := int(min(int64(len(a)), length-at))
		if _, e := source.ReadAt(a[:n], at); e != nil {
			return false, e
		}
		if _, e := stored.ReadAt(b[:n], offset+at); e != nil {
			return false, e
		}
		if !bytes.Equal(a[:n], b[:n]) {
			return false, nil
		}
		at += int64(n)
	}
	return true, nil
}
func writeGLBDocument(ctx context.Context, out io.Writer, doc map[string]any, source io.ReaderAt, offset, length int64) error {
	meta, e := json.Marshal(doc)
	if e != nil {
		return e
	}
	for len(meta)%4 != 0 {
		meta = append(meta, ' ')
	}
	padding := (4 - length%4) % 4
	total := int64(20 + len(meta))
	if length > 0 {
		total += 8 + length + padding
	}
	if total > math.MaxUint32 {
		return fmt.Errorf("GLB export exceeds 4 GiB")
	}
	w := contextWriter{ctx, out}
	var header [20]byte
	for i, v := range []uint32{0x46546c67, 2, uint32(total), uint32(len(meta)), 0x4e4f534a} {
		binary.LittleEndian.PutUint32(header[i*4:], v)
	}
	if _, e = w.Write(header[:]); e != nil {
		return e
	}
	if _, e = w.Write(meta); e != nil {
		return e
	}
	if length == 0 {
		return nil
	}
	var bin [8]byte
	binary.LittleEndian.PutUint32(bin[:], uint32(length+padding))
	binary.LittleEndian.PutUint32(bin[4:], 0x004e4942)
	if _, e = w.Write(bin[:]); e != nil {
		return e
	}
	if _, e = io.CopyBuffer(w, io.NewSectionReader(source, offset, length), make([]byte, 64<<10)); e != nil {
		return e
	}
	if padding > 0 {
		_, e = w.Write(make([]byte, padding))
	}
	return e
}
