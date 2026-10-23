package scene

import (
	"github.com/go-gl/mathgl/mgl32"
	"image/color"
	"math"
	"testing"
	"testing/fstest"
)

func TestBedrockPortalCornerRotationFitsItsFrame(t *testing.T) {
	matrix := geometryRotationMatrix([3]float32{8, 8, 0}, [3]float32{0, 0, -90})
	minP, maxP := [3]float32{100, 100, 100}, [3]float32{-100, -100, -100}
	for _, x := range []float32{6, 16} {
		for _, y := range []float32{8, 18} {
			for _, z := range []float32{-6, 6} {
				p := matrix.Mul4x1(mgl32.Vec4{x, y, z, 1})
				for a := 0; a < 3; a++ {
					minP[a] = min(minP[a], p[a])
					maxP[a] = max(maxP[a], p[a])
				}
			}
		}
	}
	for a := 0; a < 3; a++ {
		if math.Abs(float64(minP[a]-[3]float32{-2, 6, -6}[a])) > 0.0001 || math.Abs(float64(maxP[a]-[3]float32{8, 16, 6}[a])) > 0.0001 {
			t.Fatal("portal corner points outside frame", minP, maxP)
		}
	}
}

func TestBedrockModelXReflectionAndWorldFaceUVs(t *testing.T) {
	c := NewCatalog()
	err := c.AddPack(fstest.MapFS{
		"models/blocks/fixture.json": {Data: []byte(`{"minecraft:geometry":[{"description":{"identifier":"geometry.uv","texture_width":16,"texture_height":16},"bones":[{"name":"root","cubes":[{"origin":[-8,0,-8],"size":[16,16,16],"uv":{"north":{"uv":[0,0]},"south":{"uv":[0,0]},"east":{"uv":[0,0]},"west":{"uv":[0,0]},"up":{"uv":[0,0]},"down":{"uv":[0,0]}}}]}]}]}`)},
		"textures/fixture.png":       {Data: materialImage(16, 16, color.NRGBA{100, 100, 100, 255})},
	})
	if err != nil {
		t.Fatal(err)
	}
	c.SetBlockComponents("test:uv", map[string]any{"minecraft:geometry": "geometry.uv", "minecraft:material_instances": map[string]any{"*": map[string]any{"texture": "textures/fixture"}}})
	m, _ := c.customModel("test:uv", nil)
	for _, q := range m.quads {
		for i, p := range q.points {
			var u, v float32
			switch {
			case q.normal[0] > .9:
				u, v = 1-p[2], 1-p[1]
			case q.normal[0] < -.9:
				u, v = p[2], 1-p[1]
			case q.normal[1] > .9:
				u, v = 1-p[0], 1-p[2]
			case q.normal[1] < -.9:
				u, v = 1-p[0], p[2]
			case q.normal[2] > .9:
				u, v = p[0], 1-p[1]
			default:
				u, v = 1-p[0], 1-p[1]
			}
			gotU := (q.uv[i][0]*atlasSize - float32(q.texture.x)) / 16
			gotV := (q.uv[i][1]*atlasSize - float32(q.texture.y)) / 16
			if math.Abs(float64(gotU-u)) > .0001 || math.Abs(float64(gotV-v)) > .0001 {
				t.Errorf("mirrored face normal=%v point=%v got UV %v,%v want %v,%v", q.normal, p, gotU, gotV, u, v)
			}
		}
	}
	c.geometries["geometry.uv"] = blockGeometry{Bones: []geometryBone{{Name: "root", Cubes: []geometryCube{{Origin: [3]float32{0, 0, -5}, Size: [3]float32{8, 16, 10}}}}}}
	m, _ = c.customModel("test:uv", nil)
	for _, q := range m.quads {
		for _, p := range q.points {
			if p[0] < 0 || p[0] > .50001 {
				t.Fatal("Bedrock positive X must be west of block center", p)
			}
		}
	}
}
