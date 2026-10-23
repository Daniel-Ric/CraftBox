package scene

import (
	"encoding/json"
	"errors"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/tailscale/hujson"
	"io/fs"
	"math"
	"strings"
)

type blockGeometry struct {
	Description struct {
		Identifier string  `json:"identifier"`
		Width      float32 `json:"texture_width"`
		Height     float32 `json:"texture_height"`
	} `json:"description"`
	Bones []geometryBone `json:"bones"`
}
type geometryBone struct {
	Name     string         `json:"name"`
	Parent   string         `json:"parent"`
	Pivot    [3]float32     `json:"pivot"`
	Rotation [3]float32     `json:"rotation"`
	Mirror   bool           `json:"mirror"`
	Cubes    []geometryCube `json:"cubes"`
	Position [3]float32
	Scale    *[3]float32
}
type geometryCube struct {
	Origin   [3]float32      `json:"origin"`
	Size     [3]float32      `json:"size"`
	Pivot    *[3]float32     `json:"pivot"`
	Rotation [3]float32      `json:"rotation"`
	Inflate  float32         `json:"inflate"`
	Mirror   *bool           `json:"mirror"`
	UV       json.RawMessage `json:"uv"`
}
type geometryFace struct {
	UV       [2]float32  `json:"uv"`
	Size     *[2]float32 `json:"uv_size"`
	Material string      `json:"material_instance"`
	Rotation float32     `json:"uv_rotation"`
}
type modelQuad struct {
	points  [4][3]float32
	normal  [3]float32
	uv      [4][2]float32
	texture tile
	mode    string
}

func (c *Catalog) loadBlockModels(f fs.FS) error {
	for _, root := range []string{"models/blocks", "models/block", "models/entity"} {
		count := 0
		err := fs.WalkDir(f, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if p == root && errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".json") {
				return nil
			}
			count++
			if count > 4096 {
				return fs.SkipAll
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Size() > 8<<20 {
				return nil
			}
			raw, err := fs.ReadFile(f, p)
			if err != nil {
				return err
			}
			raw, err = hujson.Standardize(raw)
			if err != nil {
				return err
			}
			var file struct {
				Models []blockGeometry `json:"minecraft:geometry"`
			}
			if err = json.Unmarshal(raw, &file); err != nil {
				return err
			}
			for _, g := range file.Models {
				if g.Description.Identifier != "" {
					c.geometries[g.Description.Identifier] = g
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func rotationMatrix(pivot, rotation [3]float32) mgl32.Mat4 {
	r := mgl32.HomogRotate3DZ(rotation[2] * math.Pi / 180).Mul4(mgl32.HomogRotate3DY(rotation[1] * math.Pi / 180)).Mul4(mgl32.HomogRotate3DX(rotation[0] * math.Pi / 180))
	return mgl32.Translate3D(pivot[0], pivot[1], pivot[2]).Mul4(r).Mul4(mgl32.Translate3D(-pivot[0], -pivot[1], -pivot[2]))
}

func geometryRotationMatrix(pivot, rotation [3]float32) mgl32.Mat4 {

	rotation[0] = -rotation[0]
	rotation[2] = -rotation[2]
	return rotationMatrix(pivot, rotation)
}
func vectorValue(v any, fallback [3]float32) [3]float32 {
	switch a := v.(type) {
	case []any:
		if len(a) == 3 {
			for i := range fallback {
				fallback[i] = float32(numeric(a[i]))
			}
		}
	case []float32:
		if len(a) == 3 {
			copy(fallback[:], a)
		}
	case []float64:
		if len(a) == 3 {
			for i := range fallback {
				fallback[i] = float32(a[i])
			}
		}
	}
	return fallback
}
func blockTransform(components map[string]any) mgl32.Mat4 {
	t, _ := components["minecraft:transformation"].(map[string]any)
	if t == nil {
		return mgl32.Ident4()
	}
	rotation := vectorValue(t["rotation"], [3]float32{})
	if _, ok := t["rotation"]; !ok {
		rotation = [3]float32{float32(numeric(t["RX"])) * 90, float32(numeric(t["RY"])) * 90, float32(numeric(t["RZ"])) * 90}
	}
	scale := vectorValue(t["scale"], [3]float32{1, 1, 1})
	translation := vectorValue(t["translation"], [3]float32{})
	if _, ok := t["translation"]; !ok {
		translation = [3]float32{float32(numeric(t["TX"])), float32(numeric(t["TY"])), float32(numeric(t["TZ"]))}
	}
	if _, ok := t["scale"]; !ok {
		for i, k := range []string{"SX", "SY", "SZ"} {
			if v, ok := t[k]; ok {
				scale[i] = float32(numeric(v))
			}
		}
	}
	rp := vectorValue(t["rotation_pivot"], [3]float32{})
	sp := vectorValue(t["scale_pivot"], [3]float32{})
	for i := 0; i < 3; i++ {
		rp[i] += .5
		sp[i] += .5
	}
	return mgl32.Translate3D(translation[0], translation[1], translation[2]).Mul4(rotationMatrix(rp, rotation)).Mul4(mgl32.Translate3D(sp[0], sp[1], sp[2])).Mul4(mgl32.Scale3D(scale[0], scale[1], scale[2])).Mul4(mgl32.Translate3D(-sp[0], -sp[1], -sp[2]))
}

func (c *Catalog) customModel(name string, states map[string]any) (blockModel, bool) {
	d, ok := c.definitions[strings.TrimPrefix(name, "minecraft:")]
	if !ok {
		return blockModel{}, false
	}
	return c.definitionModel(name, states, d)
}

func (c *Catalog) definitionModel(name string, states map[string]any, d blockDefinition) (blockModel, bool) {
	components := make(map[string]any, len(d.Components))
	for k, v := range d.Components {
		components[k] = v
	}
	approximate := false
	for _, p := range d.Permutations {
		matches, supported := blockCondition(p.Condition, states)
		if !supported {
			approximate = true
			continue
		}
		if matches {
			for k, v := range p.Components {
				components[k] = v
			}
		}
	}

	if embedded, ok := components["minecraft:embedded_visual"].(map[string]any); ok {
		for _, k := range []string{"material_instances", "geometry"} {
			if v, ok := embedded[k]; ok {
				components["minecraft:"+k] = v
			}
		}
	}
	table := materialTable(components)
	geometry, _ := components["minecraft:geometry"].(map[string]any)
	id, _ := geometry["identifier"].(string)
	if id == "" {
		id, _ = components["minecraft:geometry"].(string)
	}
	g, found := c.geometries[id]
	if !found {
		if id != "" && id != "minecraft:geometry.full_block" {
			approximate = true
		}
		g.Description.Width = 16
		g.Description.Height = 16
		g.Bones = []geometryBone{{Name: "root", Cubes: []geometryCube{{Origin: [3]float32{-8, 0, -8}, Size: [3]float32{16, 16, 16}}}}}
	}
	tw, th := g.Description.Width, g.Description.Height
	if tw <= 0 {
		tw = 16
	}
	if th <= 0 {
		th = 16
	}
	out := blockModel{name: name, mode: "opaque", approx: approximate}
	transform := blockTransform(components)
	bones := map[string]geometryBone{}
	for _, b := range g.Bones {
		bones[b.Name] = b
	}
	matrices := map[string]mgl32.Mat4{}
	var boneMatrix func(string, map[string]bool) mgl32.Mat4
	boneMatrix = func(name string, seen map[string]bool) mgl32.Mat4 {
		if m, ok := matrices[name]; ok {
			return m
		}
		if seen[name] || len(seen) > 64 {
			out.approx = true
			return mgl32.Ident4()
		}
		seen[name] = true
		b := bones[name]
		m := geometryRotationMatrix(b.Pivot, b.Rotation)
		if b.Scale != nil {
			m = m.Mul4(mgl32.Translate3D(b.Pivot[0], b.Pivot[1], b.Pivot[2])).Mul4(mgl32.Scale3D(b.Scale[0], b.Scale[1], b.Scale[2])).Mul4(mgl32.Translate3D(-b.Pivot[0], -b.Pivot[1], -b.Pivot[2]))
		}
		m = mgl32.Translate3D(b.Position[0], b.Position[1], b.Position[2]).Mul4(m)
		if _, ok := bones[b.Parent]; ok && b.Parent != "" {
			m = boneMatrix(b.Parent, seen).Mul4(m)
		}
		matrices[name] = m
		return m
	}
	visibility, _ := geometry["bone_visibility"].(map[string]any)
	for _, bone := range g.Bones {
		if v, exists := visibility[bone.Name]; exists {
			visible := truth(v)
			if expr, ok := v.(string); ok {
				var supported bool
				visible, supported = blockCondition(expr, states)
				if !supported {
					visible = true
					out.approx = true
				}
			}
			if !visible {
				continue
			}
		}
		bm := boneMatrix(bone.Name, map[string]bool{})
		for _, cube := range bone.Cubes {
			pivot := [3]float32{cube.Origin[0] + cube.Size[0]/2, cube.Origin[1] + cube.Size[1]/2, cube.Origin[2] + cube.Size[2]/2}
			if cube.Pivot != nil {
				pivot = *cube.Pivot
			}
			cm := bm.Mul4(geometryRotationMatrix(pivot, cube.Rotation))
			mirror := bone.Mirror
			if cube.Mirror != nil {
				mirror = *cube.Mirror
			}
			var faces map[string]geometryFace
			var boxUV []float32
			if len(cube.UV) > 0 && cube.UV[0] == '{' {
				if json.Unmarshal(cube.UV, &faces) != nil {
					out.approx = true
					continue
				}
			} else if len(cube.UV) > 0 {
				if json.Unmarshal(cube.UV, &boxUV) != nil || len(boxUV) != 2 {
					out.approx = true
					continue
				}
			}
			for face, direction := range directions {
				uvFace := face
				if face < 2 {
					uvFace = 1 - face
				}
				if mirror && face < 2 {
					uvFace = 1 - uvFace
				}
				faceName := faceNames[uvFace]
				f, visible := faces[faceName]
				if faces != nil && !visible {
					continue
				}
				size := [2]float32{cube.Size[0], cube.Size[1]}
				if face < 2 {
					size[0] = cube.Size[2]
				}
				if face == 2 || face == 3 {
					size[1] = cube.Size[2]
				}
				if faces == nil {
					if len(boxUV) == 2 {
						u, v, dx, dy, dz := boxUV[0], boxUV[1], cube.Size[0], cube.Size[1], cube.Size[2]
						switch uvFace {
						case 0:
							f.UV = [2]float32{u, v + dz}
						case 1:
							f.UV = [2]float32{u + dz + dx, v + dz}
						case 2:
							f.UV = [2]float32{u + dz, v}
						case 3:
							f.UV = [2]float32{u + dz + dx, v}
						case 4:
							f.UV = [2]float32{u + dz + dx + dz, v + dz}
						case 5:
							f.UV = [2]float32{u + dz, v + dz}
						}
						_ = dy
					} else {
						size = [2]float32{tw, th}
					}
				} else if f.Size != nil {
					size = *f.Size
				}
				if size[0] == 0 || size[1] == 0 {
					continue
				}
				matName := f.Material
				if matName == "" {
					matName = faceName
				}
				material := resolveMaterial(table, matName)
				tex := c.materialTexture(material, tw, th)
				q := modelQuad{texture: tex, mode: renderMode(material)}
				uv := [4][2]float32{{1, 1}, {0, 1}, {0, 0}, {1, 0}}
				if face == 2 || face == 3 {
					uv = [4][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
				}
				if face == 3 && len(boxUV) == 2 {
					uv = [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
				}
				for i, corner := range corners[face] {
					point := mgl32.Vec4{cube.Origin[0] - cube.Inflate, cube.Origin[1] - cube.Inflate, cube.Origin[2] - cube.Inflate, 1}
					for a := 0; a < 3; a++ {
						point[a] += float32(corner[a]) * (cube.Size[a] + 2*cube.Inflate)
					}
					point = cm.Mul4x1(point)
					point = mgl32.Vec4{.5 - point[0]/16, point[1] / 16, point[2]/16 + .5, 1}
					point = transform.Mul4x1(point)
					q.points[i] = [3]float32{point[0], point[1], point[2]}
					u, v := uv[i][0], uv[i][1]
					if mirror {
						u = 1 - u
					}
					for n := 0; n < int(f.Rotation/90)%4; n++ {
						u, v = v, 1-u
					}
					q.uv[i] = [2]float32{(float32(tex.x) + (f.UV[0]+u*size[0])/tw*float32(tex.w)) / atlasSize, (float32(tex.y) + (f.UV[1]+v*size[1])/th*float32(tex.h)) / atlasSize}
				}

				q.points[1], q.points[3] = q.points[3], q.points[1]
				q.uv[1], q.uv[3] = q.uv[3], q.uv[1]

				a := mgl32.Vec3(q.points[1]).Sub(mgl32.Vec3(q.points[0]))
				b := mgl32.Vec3(q.points[2]).Sub(mgl32.Vec3(q.points[0]))
				n := a.Cross(b)
				if n.Len() < 1e-8 {
					continue
				}
				n = n.Normalize()
				q.normal = [3]float32(n)
				_ = direction
				out.quads = append(out.quads, q)
			}
		}
	}
	return out, true
}

func emitModelQuad(p *Part, q modelQuad, offset [3]float32) {
	base := uint32(len(p.Positions) / 3)
	for i, v := range q.points {
		p.Positions = append(p.Positions, v[0]+offset[0], v[1]+offset[1], v[2]+offset[2])
		p.Normals = append(p.Normals, q.normal[:]...)
		p.UVs = append(p.UVs, q.uv[i][:]...)
	}
	p.Indices = append(p.Indices, base, base+1, base+2, base, base+2, base+3)
}
