package scene

import (
	"bytes"
	"github.com/df-mc/dragonfly/server/world"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
	"testing/fstest"
)

func TestFenceGateUsesClosedOpenAndWallGeometry(t *testing.T) {
	r := world.DefaultBlockRegistry
	r.Finalize()
	s := newTestStore(t, r, NewCatalog())
	for _, dir := range []string{"north", "south", "east", "west"} {
		for _, open := range []bool{false, true} {
			for _, wall := range []bool{false, true} {
				rid, ok := r.StateToRuntimeID("minecraft:birch_fence_gate", map[string]any{"minecraft:cardinal_direction": dir, "open_bit": open, "in_wall_bit": wall})
				if !ok {
					t.Fatal(dir, open, wall)
				}
				m := s.model(rid)
				if len(m.boxes) < 6 {
					t.Errorf("gate is only a post: %s open=%v wall=%v", dir, open, wall)
					continue
				}
				maxY := float32(0)
				for _, b := range m.boxes {
					maxY = max(maxY, b.max[1])
				}
				want := float32(1)
				if wall {
					want -= 3.0 / 16
				}
				if maxY != want {
					t.Fatal("wrong lowered gate height", dir, wall, maxY)
				}
				axis := 0
				if dir == "east" || dir == "west" {
					axis = 2
				}
				middle := false
				for _, b := range m.boxes {
					if b.min[axis] <= .5 && b.max[axis] >= .5 {
						middle = true
					}
				}
				if middle == open {
					t.Errorf("gate opening incorrect %s %v", dir, open)
				}
			}
		}
	}
}

func TestFenceGateSideAttachmentAndFenceFamilies(t *testing.T) {
	fence := &blockModel{name: "minecraft:oak_fence", connect: "fence", boxes: []box{{[3]float32{.375, 0, .375}, [3]float32{.625, 1, .625}}}}
	gate := &blockModel{name: "minecraft:birch_fence_gate", gateAxis: 2}
	n := [4]*blockModel{gate, &emptyBlockModel, &emptyBlockModel, &emptyBlockModel}
	if got := connectedBoxes(fence, n); len(got) != 3 {
		t.Fatal("fence failed to attach to side of gate")
	}
	n = [4]*blockModel{&emptyBlockModel, &emptyBlockModel, gate, &emptyBlockModel}
	if got := connectedBoxes(fence, n); len(got) != 1 {
		t.Fatal("fence incorrectly attached to front of gate")
	}
	n = [4]*blockModel{&blockModel{name: "minecraft:nether_brick_fence", connect: "fence"}, &emptyBlockModel, &emptyBlockModel, &emptyBlockModel}
	if got := connectedBoxes(fence, n); len(got) != 1 {
		t.Fatal("wood fence joined nether brick fence")
	}
}
func TestAnvilHeadTextureDoesNotRotateIntoTransparentMargin(t *testing.T) {
	c := NewCatalog()
	im := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 3; x < 13; x++ {
			im.SetNRGBA(x, y, color.NRGBA{220, 40, 30, 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, im)
	c.AddPack(fstest.MapFS{"blocks.json": {Data: []byte(`{"anvil":{"textures":{"up":"head","side":"base","down":"base"}}}`)}, "textures/blocks/head.png": {Data: buf.Bytes()}, "textures/blocks/base.png": {Data: materialImage(16, 16, color.NRGBA{80, 80, 80, 255})}})
	r := world.DefaultBlockRegistry
	r.Finalize()
	s := newTestStore(t, r, c)
	for _, dir := range []string{"north", "south", "east", "west"} {
		rid, _ := r.StateToRuntimeID("minecraft:anvil", map[string]any{"minecraft:cardinal_direction": dir})
		m := s.model(rid)
		if len(m.quads) != 24 {
			t.Errorf("anvil lacks model-local UVs %s", dir)
			continue
		}
		for _, q := range m.quads {
			if q.normal[1] < .9 {
				continue
			}
			head := q.points[0][1] == 1
			lo, hi := float32(99), float32(-99)
			for _, uv := range q.uv {
				u := uv[0]*atlasSize - float32(q.texture.x)
				lo = min(lo, u)
				hi = max(hi, u)
			}
			if head && (math.Abs(float64(lo-3)) > .0001 || math.Abs(float64(hi-13)) > .0001) {
				t.Fatal("head samples transparent columns", dir, lo, hi)
			}
			if !head && q.texture.alpha {
				t.Fatal("foot or neck uses head texture")
			}
		}
	}
}
func TestSaplingsAndVinesHaveOnlyTheirVisiblePlanes(t *testing.T) {
	r := world.DefaultBlockRegistry
	r.Finalize()
	s := newTestStore(t, r, NewCatalog())
	rid, _ := r.StateToRuntimeID("minecraft:jungle_sapling", map[string]any{"age_bit": true})
	m := s.model(rid)
	if len(m.quads) != 2 || len(m.boxes) != 0 {
		t.Error("sapling lacks two crossed planes")
	}
	for _, q := range m.quads {
		for _, p := range q.points {
			if p[0] <= 0 || p[0] >= 1 || p[2] <= 0 || p[2] >= 1 {
				t.Fatal("sapling still spans cube walls", p)
			}
		}
		length := float64(q.normal[0]*q.normal[0] + q.normal[1]*q.normal[1] + q.normal[2]*q.normal[2])
		if math.Abs(length-1) > .0001 {
			t.Fatal("incorrect plant normal")
		}
	}
	rid, _ = r.StateToRuntimeID("minecraft:vine", map[string]any{"vine_direction_bits": int32(2)})
	m = s.model(rid)
	if len(m.quads) != 1 || len(m.boxes) != 0 {
		t.Error("one-face vine has artificial side walls")
	}
}
