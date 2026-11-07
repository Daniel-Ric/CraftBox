package modelexport

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

type fbxArray struct {
	kind  byte
	count int
	value func(int) float64
}
type fbxWriter struct {
	model *model
	file  *os.File
	err   error
}

func (w *fbxWriter) write(raw []byte) {
	if w.err == nil {
		_, w.err = w.file.Write(raw)
	}
}
func (w *fbxWriter) number(value any) {
	if w.err == nil {
		w.err = binary.Write(w.file, binary.LittleEndian, value)
	}
}
func (w *fbxWriter) position() int64 {
	if w.err != nil {
		return 0
	}
	var position int64
	position, w.err = w.file.Seek(0, io.SeekCurrent)
	return position
}
func (w *fbxWriter) property(value any) {
	switch value := value.(type) {
	case string:
		w.write([]byte{'S'})
		w.number(uint32(len(value)))
		w.write([]byte(value))
	case int32:
		w.write([]byte{'I'})
		w.number(value)
	case int64:
		w.write([]byte{'L'})
		w.number(value)
	case float64:
		w.write([]byte{'D'})
		w.number(value)
	case bool:
		w.write([]byte{'C'})
		if value {
			w.write([]byte{1})
		} else {
			w.write([]byte{0})
		}
	case fbxArray:
		width := 4
		if value.kind == 'd' {
			width = 8
		}
		w.write([]byte{value.kind})
		w.number(uint32(value.count))
		w.number(uint32(0))
		w.number(uint32(value.count * width))
		var buffer [64 << 10]byte
		for start := 0; start < value.count; {
			if err := w.model.ctx.Err(); err != nil {
				w.err = err
				return
			}
			count := min(value.count-start, len(buffer)/width)
			for i := 0; i < count; i++ {
				if width == 8 {
					binary.LittleEndian.PutUint64(buffer[i*8:], math.Float64bits(value.value(start+i)))
				} else {
					binary.LittleEndian.PutUint32(buffer[i*4:], uint32(int32(value.value(start+i))))
				}
			}
			w.write(buffer[:count*width])
			start += count
		}
	default:
		w.err = fmt.Errorf("Unsupported FBX property %T", value)
	}
}
func (w *fbxWriter) node(name string, properties []any, children func()) {
	if w.err != nil {
		return
	}
	start := w.position()
	w.write(make([]byte, 13))
	w.write([]byte(name))
	propertyStart := w.position()
	for _, value := range properties {
		w.property(value)
	}
	propertyEnd := w.position()
	if children != nil {
		children()
		w.write(make([]byte, 13))
	}
	end := w.position()
	if w.err != nil {
		return
	}
	if end > math.MaxUint32 || len(name) > 255 {
		w.err = fmt.Errorf("FBX export exceeds the 4 GiB binary format limit")
		return
	}
	var header [13]byte
	binary.LittleEndian.PutUint32(header[:], uint32(end))
	binary.LittleEndian.PutUint32(header[4:], uint32(len(properties)))
	binary.LittleEndian.PutUint32(header[8:], uint32(propertyEnd-propertyStart))
	header[12] = byte(len(name))
	_, w.err = w.file.WriteAt(header[:], start)
}
func (w *fbxWriter) p(name, kind string, values ...any) {
	properties := []any{name, kind, "", "A"}
	properties = append(properties, values...)
	w.node("P", properties, nil)
}
func (w *fbxWriter) layer(name, mapping, reference, dataName string, values fbxArray) {
	w.node(name, []any{int32(0)}, func() {
		w.node("Version", []any{int32(101)}, nil)
		w.node("Name", []any{"UVMap"}, nil)
		w.node("MappingInformationType", []any{mapping}, nil)
		w.node("ReferenceInformationType", []any{reference}, nil)
		w.node(dataName, []any{values}, nil)
	})
}

func (m *model) writeFBX(out io.Writer, progress func(int, int)) error {
	file, err := os.CreateTemp(filepath.Dir(m.file.Name()), ".craftbox-fbx-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	w := &fbxWriter{model: m, file: file}
	w.write([]byte("Kaydara FBX Binary  \x00\x1a\x00"))
	w.number(uint32(7400))
	w.node("FBXHeaderExtension", nil, func() {
		w.node("FBXHeaderVersion", []any{int32(1003)}, nil)
		w.node("FBXVersion", []any{int32(7400)}, nil)
		w.node("EncryptionType", []any{int32(0)}, nil)
		w.node("Creator", []any{"CraftBox"}, nil)
	})
	w.node("GlobalSettings", nil, func() {
		w.node("Version", []any{int32(1000)}, nil)
		w.node("Properties70", nil, func() {
			up, front, frontSign := int32(1), int32(2), int32(-1)
			if m.options.UpAxis == "z" {
				up, front = 2, 1
				frontSign = 1
			}
			for _, setting := range []struct {
				name  string
				value int32
			}{{"UpAxis", up}, {"UpAxisSign", 1}, {"FrontAxis", front}, {"FrontAxisSign", frontSign}, {"CoordAxis", 0}, {"CoordAxisSign", 1}, {"OriginalUpAxis", up}, {"OriginalUpAxisSign", 1}} {
				w.p(setting.name, "int", setting.value)
			}
			w.p("UnitScaleFactor", "double", float64(100))
			w.p("OriginalUnitScaleFactor", "double", float64(100))
		})
	})
	w.node("Documents", nil, func() {
		w.node("Count", []any{int32(1)}, nil)
		w.node("Document", []any{int64(1), "Scene", "Scene"}, func() { w.node("RootNode", []any{int64(0)}, nil) })
	})
	parts := 0
	for _, node := range m.doc.Nodes {
		if node.Mesh != nil {
			parts += len(m.doc.Meshes[*node.Mesh].Primitives)
		}
	}
	w.node("Definitions", nil, func() {
		w.node("Version", []any{int32(100)}, nil)
		count := parts*2 + len(m.doc.Materials)
		if m.options.WithTextures() {
			count += 4
		}
		w.node("Count", []any{int32(count)}, nil)
		for _, definition := range []struct {
			name  string
			count int
		}{{"Geometry", parts}, {"Model", parts}, {"Material", len(m.doc.Materials)}} {
			w.node("ObjectType", []any{definition.name}, func() { w.node("Count", []any{int32(definition.count)}, nil) })
		}
		if m.options.WithTextures() {
			for _, name := range []string{"Video", "Texture"} {
				w.node("ObjectType", []any{name}, func() { w.node("Count", []any{int32(2)}, nil) })
			}
		}
	})
	type connection struct{ geometry, model, material int64 }
	connections := []connection{}
	w.node("Objects", nil, func() {
		for index, material := range m.doc.Materials {
			color := m.color(index)
			w.node("Material", []any{int64(1000 + index), material.Name + "\x00\x01Material", ""}, func() {
				w.node("Version", []any{int32(102)}, nil)
				w.node("ShadingModel", []any{"phong"}, nil)
				w.node("MultiLayer", []any{int32(0)}, nil)
				w.node("Properties70", nil, func() {
					w.p("DiffuseColor", "Color", color[0], color[1], color[2])
					w.p("DiffuseFactor", "Number", float64(1))
					w.p("TransparencyFactor", "Number", 1-color[3])
					w.p("Opacity", "Number", color[3])
					w.p("Shininess", "Number", float64(0))
				})
			})
		}
		if m.options.WithTextures() {
			w.node("Video", []any{int64(2002), "opacity\x00\x01Video", "Clip"}, func() {
				w.node("Type", []any{"Clip"}, nil)
				w.node("Filename", []any{"opacity.png"}, nil)
				w.node("RelativeFilename", []any{"opacity.png"}, nil)
			})
			w.node("Texture", []any{int64(2003), "opacity\x00\x01Texture", "TextureVideoClip"}, func() {
				w.node("Type", []any{"TextureVideoClip"}, nil)
				w.node("Version", []any{int32(202)}, nil)
				w.node("TextureName", []any{"Texture::opacity"}, nil)
				w.node("FileName", []any{"opacity.png"}, nil)
				w.node("RelativeFilename", []any{"opacity.png"}, nil)
			})
			w.node("Video", []any{int64(2000), "atlas\x00\x01Video", "Clip"}, func() {
				w.node("Type", []any{"Clip"}, nil)
				w.node("Filename", []any{"atlas.png"}, nil)
				w.node("RelativeFilename", []any{"atlas.png"}, nil)
			})
			w.node("Texture", []any{int64(2001), "atlas\x00\x01Texture", "TextureVideoClip"}, func() {
				w.node("Type", []any{"TextureVideoClip"}, nil)
				w.node("Version", []any{int32(202)}, nil)
				w.node("TextureName", []any{"Texture::atlas"}, nil)
				w.node("Media", []any{"Video::atlas"}, nil)
				w.node("FileName", []any{"atlas.png"}, nil)
				w.node("RelativeFilename", []any{"atlas.png"}, nil)
				w.node("ModelUVTranslation", []any{float64(0), float64(0)}, nil)
				w.node("ModelUVScaling", []any{float64(1), float64(1)}, nil)
				w.node("Properties70", nil, func() { w.p("UVSet", "KString", "UVMap") })
			})
		}
		err = m.parts(func(part meshPart) error {
			geometryID := int64(10000 + len(connections)*2)
			modelID := geometryID + 1
			connections = append(connections, connection{geometryID, modelID, int64(1000 + part.Material)})
			w.node("Geometry", []any{geometryID, part.Name + "\x00\x01Geometry", "Mesh"}, func() {
				w.node("GeometryVersion", []any{int32(124)}, nil)
				w.node("Vertices", []any{fbxArray{'d', len(part.Positions), func(i int) float64 { return m.point(part, i/3)[i%3] }}}, nil)
				w.node("PolygonVertexIndex", []any{fbxArray{'i', len(part.Indices), func(i int) float64 {
					value := int32(part.Indices[i])
					if i%3 == 2 {
						value = -value - 1
					}
					return float64(value)
				}}}, nil)
				w.layer("LayerElementNormal", "ByVertice", "Direct", "Normals", fbxArray{'d', len(part.Normals), func(i int) float64 { return m.normal(part, i/3)[i%3] }})
				w.layer("LayerElementUV", "ByVertice", "Direct", "UV", fbxArray{'d', len(part.UVs), func(i int) float64 {
					if i%2 == 1 {
						return 1 - float64(part.UVs[i])
					}
					return float64(part.UVs[i])
				}})
				w.layer("LayerElementMaterial", "AllSame", "IndexToDirect", "Materials", fbxArray{'i', 1, func(int) float64 { return 0 }})
				w.node("Layer", []any{int32(0)}, func() {
					w.node("Version", []any{int32(100)}, nil)
					for _, name := range []string{"LayerElementNormal", "LayerElementMaterial", "LayerElementUV"} {
						w.node("LayerElement", nil, func() { w.node("Type", []any{name}, nil); w.node("TypedIndex", []any{int32(0)}, nil) })
					}
				})
			})
			w.node("Model", []any{modelID, part.Name + "\x00\x01Model", "Mesh"}, func() {
				w.node("Version", []any{int32(232)}, nil)
				w.node("Properties70", nil, func() {
					w.p("Lcl Translation", "Lcl Translation", float64(0), float64(0), float64(0))
					w.p("Lcl Rotation", "Lcl Rotation", float64(0), float64(0), float64(0))
					w.p("Lcl Scaling", "Lcl Scaling", float64(1), float64(1), float64(1))
				})
				w.node("Shading", []any{true}, nil)
				w.node("Culling", []any{"CullingOff"}, nil)
			})
			return w.err
		}, progress)
		if err != nil {
			w.err = err
		}
	})
	w.node("Connections", nil, func() {
		for _, link := range connections {
			w.node("C", []any{"OO", link.geometry, link.model}, nil)
			w.node("C", []any{"OO", link.material, link.model}, nil)
			w.node("C", []any{"OO", link.model, int64(0)}, nil)
		}
		if m.options.WithTextures() {
			w.node("C", []any{"OO", int64(2000), int64(2001)}, nil)
			w.node("C", []any{"OO", int64(2002), int64(2003)}, nil)
			for index := range m.doc.Materials {
				w.node("C", []any{"OP", int64(2001), int64(1000 + index), "DiffuseColor"}, nil)
				if m.doc.Materials[index].AlphaMode == "MASK" || m.doc.Materials[index].AlphaMode == "BLEND" {
					w.node("C", []any{"OP", int64(2003), int64(1000 + index), "TransparentColor"}, nil)
				}
			}
		}
	})
	w.write(make([]byte, 13))
	w.write([]byte{0xfa, 0xbc, 0xab, 0x09, 0xd0, 0xc8, 0xd4, 0x66, 0xb1, 0x76, 0xfb, 0x83, 0x1c, 0xf7, 0x26, 0x7e})
	if padding := (16 - w.position()%16) % 16; padding != 0 {
		w.write(make([]byte, padding))
	}
	w.number(uint32(0))
	w.number(uint32(7400))
	w.write(make([]byte, 120))
	w.write([]byte{0xf8, 0x5a, 0x8c, 0x6a, 0xde, 0xf5, 0xd9, 0x7e, 0xec, 0xe9, 0x0c, 0xe3, 0x75, 0x8f, 0x29, 0x0b})
	if w.err != nil {
		return w.err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = io.CopyBuffer(contextWriter{m.ctx, out}, file, make([]byte, 64<<10))
	return err
}
