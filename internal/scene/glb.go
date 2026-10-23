package scene

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

func EncodeGLB(s Snapshot) ([]byte, error) {
	if !s.Reset {
		return nil, fmt.Errorf("GLB requires a complete scene snapshot")
	}
	var bin bytes.Buffer
	views := []any{}
	accessors := []any{}
	meshes := []any{}
	nodes := []any{}
	roots := []int{}
	view := func(data []byte, target int) int {
		for bin.Len()%4 != 0 {
			bin.WriteByte(0)
		}
		start := bin.Len()
		bin.Write(data)
		v := map[string]any{"buffer": 0, "byteOffset": start, "byteLength": len(data)}
		if target != 0 {
			v["target"] = target
		}
		views = append(views, v)
		return len(views) - 1
	}
	floats := func(values []float32, kind string, bounds bool) int {
		data := make([]byte, len(values)*4)
		for i, f := range values {
			binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
		}
		width := 3
		if kind == "VEC2" {
			width = 2
		}
		a := map[string]any{"bufferView": view(data, 34962), "componentType": 5126, "count": len(values) / width, "type": kind}
		if bounds {
			lo, hi := []float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}, []float32{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
			for i, f := range values {
				lo[i%3] = min(lo[i%3], f)
				hi[i%3] = max(hi[i%3], f)
			}
			a["min"] = lo
			a["max"] = hi
		}
		accessors = append(accessors, a)
		return len(accessors) - 1
	}
	for _, c := range s.Chunks {
		primitives := []any{}
		for _, p := range c.Parts {
			if len(p.Positions) == 0 {
				continue
			}
			position, normal, uv := floats(p.Positions, "VEC3", true), floats(p.Normals, "VEC3", false), floats(p.UVs, "VEC2", false)
			idx := make([]byte, len(p.Indices)*4)
			for i, v := range p.Indices {
				binary.LittleEndian.PutUint32(idx[i*4:], v)
			}
			accessors = append(accessors, map[string]any{"bufferView": view(idx, 34963), "componentType": 5125, "count": len(p.Indices), "type": "SCALAR"})
			material := 0
			if p.Mode == "mask" {
				material = 1
			} else if p.Mode == "blend" {
				material = 2
			} else if p.Mode == "unlit" || p.Mode == "billboard" {
				material = 3
			} else if p.Mode == "unlit-blend" {
				material = 4
			}
			primitives = append(primitives, map[string]any{"attributes": map[string]int{"POSITION": position, "NORMAL": normal, "TEXCOORD_0": uv}, "indices": len(accessors) - 1, "material": material, "mode": 4})
		}
		if len(primitives) == 0 {
			continue
		}
		meshes = append(meshes, map[string]any{"name": fmt.Sprintf("chunk_%d_%d", c.X, c.Z), "primitives": primitives})
		roots = append(roots, len(nodes))
		nodes = append(nodes, map[string]any{"mesh": len(meshes) - 1, "translation": []float64{float64(c.X) * 16, 0, float64(c.Z) * 16}})
	}
	if len(meshes) == 0 {
		return nil, fmt.Errorf("No captured 3D geometry available")
	}
	pngData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s.Atlas, "data:image/png;base64,"))
	if err != nil || len(pngData) == 0 {
		return nil, fmt.Errorf("Texture atlas is missing or invalid")
	}
	imageView := view(pngData, 0)
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
	materials = append(materials, map[string]any{"name": "unlit", "alphaMode": "MASK", "alphaCutoff": .05, "doubleSided": true, "pbrMetallicRoughness": map[string]any{"baseColorTexture": map[string]int{"index": 0}, "metallicFactor": 0, "roughnessFactor": 1}, "extensions": map[string]any{"KHR_materials_unlit": map[string]any{}}})
	materials = append(materials, map[string]any{"name": "unlit-blend", "alphaMode": "BLEND", "doubleSided": true, "pbrMetallicRoughness": map[string]any{"baseColorTexture": map[string]int{"index": 0}, "metallicFactor": 0, "roughnessFactor": 1}, "extensions": map[string]any{"KHR_materials_unlit": map[string]any{}}})
	doc := map[string]any{"asset": map[string]string{"version": "2.0", "generator": "CraftBox", "copyright": s.Source}, "scene": 0, "scenes": []any{map[string]any{"nodes": roots}}, "nodes": nodes, "meshes": meshes, "materials": materials, "buffers": []any{map[string]int{"byteLength": bin.Len()}}, "bufferViews": views, "accessors": accessors, "images": []any{map[string]any{"bufferView": imageView, "mimeType": "image/png"}}, "textures": []any{map[string]int{"sampler": 0, "source": 0}}, "samplers": []any{map[string]int{"magFilter": 9728, "minFilter": 9728, "wrapS": 33071, "wrapT": 33071}}}
	doc["extensionsUsed"] = []string{"KHR_materials_unlit"}
	js, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}
	for bin.Len()%4 != 0 {
		bin.WriteByte(0)
	}
	total := uint64(12 + 8 + len(js) + 8 + bin.Len())
	if total > math.MaxUint32 {
		return nil, fmt.Errorf("GLB exceeds 4 GiB; select a smaller region")
	}
	var out bytes.Buffer
	for _, v := range []uint32{0x46546c67, 2, uint32(total), uint32(len(js)), 0x4e4f534a} {
		_ = binary.Write(&out, binary.LittleEndian, v)
	}
	out.Write(js)
	_ = binary.Write(&out, binary.LittleEndian, uint32(bin.Len()))
	_ = binary.Write(&out, binary.LittleEndian, uint32(0x004e4942))
	out.Write(bin.Bytes())
	return out.Bytes(), nil
}
