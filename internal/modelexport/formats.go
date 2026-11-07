package modelexport

import (
	"archive/zip"
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

func (m *model) writeGLTF(out io.Writer, archive *zip.Writer) error {
	length := m.binLength
	if !m.options.WithTextures() {
		for _, key := range []string{"images", "textures", "samplers"} {
			delete(m.raw, key)
		}
		for _, material := range glbArray(m.raw["materials"]) {
			removeGLBTextures(glbObject(material))
		}
		if len(m.doc.Images) == 1 && m.doc.Images[0].BufferView == len(m.doc.BufferViews)-1 {
			length = m.doc.BufferViews[len(m.doc.BufferViews)-1].ByteOffset
			if views, ok := m.raw["bufferViews"].([]any); ok {
				m.raw["bufferViews"] = views[:len(views)-1]
			}
		}
	}
	if m.options.Scale != 1 || m.options.Origin == "center" {
		nodes, ok := m.raw["nodes"].([]any)
		if !ok || len(m.doc.Scenes) == 0 {
			return fmt.Errorf("Model root missing")
		}
		scale := m.options.Scale
		matrix := []float64{scale, 0, 0, 0, 0, scale, 0, 0, 0, 0, scale, 0, -m.center[0] * scale, -m.center[1] * scale, -m.center[2] * scale, 1}
		m.raw["nodes"] = append(nodes, map[string]any{"name": "CraftBox export transform", "matrix": matrix, "children": m.doc.Scenes[0].Nodes})
		m.raw["scenes"] = []any{map[string]any{"nodes": []int{len(nodes)}}}
	}
	if archive == nil {
		return m.writeLosslessGLB(out)
	}
	m.raw["buffers"] = []any{map[string]any{"byteLength": length, "uri": "scene.bin"}}
	metadata, err := json.Marshal(m.raw)
	if err != nil {
		return err
	}
	entry, err := archive.Create("scene.gltf")
	if err != nil {
		return err
	}
	if _, err = entry.Write(metadata); err != nil {
		return err
	}
	binaryFile, err := archive.Create("scene.bin")
	if err != nil {
		return err
	}
	_, err = io.CopyBuffer(contextWriter{m.ctx, binaryFile}, io.NewSectionReader(m.file, m.binOffset, length), make([]byte, 64<<10))
	return err
}

func removeGLBTextures(m map[string]any) {
	for key, value := range m {
		if key == "extras" {
			continue
		}
		child := glbObject(value)
		if child == nil {
			continue
		}
		if strings.HasSuffix(strings.ToLower(key), "texture") {
			if _, ok := child["index"]; ok {
				delete(m, key)
			}
		} else {
			removeGLBTextures(child)
		}
	}
}

func (m *model) writeOBJ(archive *zip.Writer, progress func(int, int)) error {
	entry, err := archive.Create("scene.obj")
	if err != nil {
		return err
	}
	out := bufio.NewWriterSize(entry, 64<<10)
	fmt.Fprintln(out, "# CraftBox static model\nmtllib scene.mtl")
	offset := uint64(1)
	err = m.parts(func(part meshPart) error {
		fmt.Fprintf(out, "o %s\nusemtl material_%d\n", strings.ReplaceAll(part.Name, " ", "_"), part.Material)
		for i := 0; i < len(part.Positions)/3; i++ {
			if i%1024 == 0 {
				if err := m.ctx.Err(); err != nil {
					return err
				}
			}
			point := m.point(part, i)
			fmt.Fprintf(out, "v %.9g %.9g %.9g\n", point[0], point[1], point[2])
		}
		for i := 0; i < len(part.UVs)/2; i++ {
			if i%1024 == 0 {
				if err := m.ctx.Err(); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(out, "vt %.9g %.9g\n", part.UVs[i*2], 1-part.UVs[i*2+1]); err != nil {
				return err
			}
		}
		for i := 0; i < len(part.Normals)/3; i++ {
			if i%1024 == 0 {
				if err := m.ctx.Err(); err != nil {
					return err
				}
			}
			normal := m.normal(part, i)
			if _, err := fmt.Fprintf(out, "vn %.9g %.9g %.9g\n", normal[0], normal[1], normal[2]); err != nil {
				return err
			}
		}
		for i := 0; i < len(part.Indices); i += 3 {
			if i%3072 == 0 {
				if err := m.ctx.Err(); err != nil {
					return err
				}
			}
			fmt.Fprint(out, "f")
			for _, vertex := range part.Indices[i : i+3] {
				index := offset + uint64(vertex)
				fmt.Fprintf(out, " %d/%d/%d", index, index, index)
			}
			fmt.Fprintln(out)
		}
		offset += uint64(len(part.Positions) / 3)
		return out.Flush()
	}, progress)
	if err != nil {
		return err
	}
	if err = out.Flush(); err != nil {
		return err
	}
	material, err := archive.Create("scene.mtl")
	if err != nil {
		return err
	}
	for index := range m.doc.Materials {
		color := m.color(index)
		if _, err = fmt.Fprintf(material, "newmtl material_%d\nKd %.6g %.6g %.6g\nKa 0 0 0\nKs 0 0 0\nillum 2\nd %.6g\n", index, color[0], color[1], color[2], color[3]); err != nil {
			return err
		}
		if m.options.WithTextures() {
			if _, err = fmt.Fprintln(material, "map_Kd atlas.png"); err != nil {
				return err
			}
			if m.doc.Materials[index].AlphaMode == "MASK" || m.doc.Materials[index].AlphaMode == "BLEND" {
				if _, err = fmt.Fprintln(material, "map_d opacity.png"); err != nil {
					return err
				}
			}
		}
	}
	if m.options.WithTextures() {
		atlas, err := m.atlas()
		if err != nil {
			return err
		}
		texture, err := archive.Create("atlas.png")
		if err != nil {
			return err
		}
		if _, err = texture.Write(atlas); err != nil {
			return err
		}
		return m.writeOpacity(archive)
	}
	return nil
}

func (m *model) writeSTL(writer io.Writer, progress func(int, int)) error {
	count := uint64(0)
	for _, node := range m.doc.Nodes {
		if node.Mesh == nil {
			continue
		}
		if *node.Mesh < 0 || *node.Mesh >= len(m.doc.Meshes) {
			return fmt.Errorf("Invalid STL mesh")
		}
		for _, part := range m.doc.Meshes[*node.Mesh].Primitives {
			if part.Indices < 0 || part.Indices >= len(m.doc.Accessors) {
				return fmt.Errorf("Invalid STL indices")
			}
			count += uint64(m.doc.Accessors[part.Indices].Count / 3)
		}
	}
	if count > math.MaxUint32 {
		return fmt.Errorf("STL has too many triangles")
	}
	var header [84]byte
	copy(header[:], "CraftBox binary STL; geometry only")
	binary.LittleEndian.PutUint32(header[80:], uint32(count))
	if _, err := writer.Write(header[:]); err != nil {
		return err
	}
	out := bufio.NewWriterSize(writer, 64<<10)
	err := m.parts(func(part meshPart) error {
		var triangle [50]byte
		for i := 0; i < len(part.Indices); i += 3 {
			if i%3072 == 0 {
				if err := m.ctx.Err(); err != nil {
					return err
				}
			}
			a, b, c := m.point(part, int(part.Indices[i])), m.point(part, int(part.Indices[i+1])), m.point(part, int(part.Indices[i+2]))
			u, v := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}, [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
			normal := [3]float64{u[1]*v[2] - u[2]*v[1], u[2]*v[0] - u[0]*v[2], u[0]*v[1] - u[1]*v[0]}
			length := math.Sqrt(normal[0]*normal[0] + normal[1]*normal[1] + normal[2]*normal[2])
			if length != 0 {
				for j := range normal {
					normal[j] /= length
				}
			}
			offset := 0
			for _, point := range [][3]float64{normal, a, b, c} {
				for _, value := range point {
					binary.LittleEndian.PutUint32(triangle[offset:], math.Float32bits(float32(value)))
					offset += 4
				}
			}
			if _, err := out.Write(triangle[:]); err != nil {
				return err
			}
		}
		return nil
	}, progress)
	if err != nil {
		return err
	}
	return out.Flush()
}
