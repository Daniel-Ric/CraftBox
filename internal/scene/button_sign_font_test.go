package scene

import (
	"github.com/Daniel-Ric/CraftBox/assets"
	"image/color"
	"math"
	"testing"
)

func TestWallButtonsStayWiderThanTall(t *testing.T) {
	c := mechanicalCatalog(t)
	for _, name := range []string{"minecraft:wooden_button", "minecraft:stone_button", "minecraft:spruce_button"} {
		for _, pressed := range []bool{false, true} {
			for _, face := range []int32{2, 3, 4, 5} {
				m := c.buttonModel(name, map[string]any{"facing_direction": face, "button_pressed_bit": pressed})
				low, high := [3]float32{1, 1, 1}, [3]float32{}
				for _, q := range m.quads {
					for _, p := range q.points {
						for a := range p {
							low[a] = min(low[a], p[a])
							high[a] = max(high[a], p[a])
						}
					}
				}
				width := high[0] - low[0]
				depth := high[2] - low[2]
				if face == 4 || face == 5 {
					width, depth = depth, width
				}
				wantDepth := float32(.125)
				if pressed {
					wantDepth = .0625
				}
				if math.Abs(float64(width-.375)) > 1e-6 || math.Abs(float64(high[1]-low[1]-.25)) > 1e-6 || math.Abs(float64(depth-wantDepth)) > 1e-6 {
					t.Fatalf("%s face %d pressed %t: size %v, want horizontal 6x4 pixel button", name, face, pressed, [3]float32{high[0] - low[0], high[1] - low[1], high[2] - low[2]})
				}
			}
		}
	}
}

func TestSignUsesMinecraftGlyphMetricsAndPixelShapes(t *testing.T) {
	c := mechanicalCatalog(t)
	narrow, _ := c.textTile("iiii", color.NRGBA{255, 255, 255, 255}, false)
	wide, _ := c.textTile("WWWW", color.NRGBA{255, 255, 255, 255}, false)
	if wide.w <= narrow.w*2 {
		t.Fatalf("sign still uses monospace metrics: iiii=%d WWWW=%d", narrow.w, wide.w)
	}
	// Mojangles' native two-pixel grid produces opaque pixels without the
	// antialiased curved strokes of the old Go Mono fallback.
	for y := wide.y; y < wide.y+wide.h; y++ {
		for x := wide.x; x < wide.x+wide.w; x++ {
			a := c.atlas.NRGBAAt(x, y).A
			if a != 0 && a != 255 {
				t.Fatalf("sign glyph has antialiased coverage %d instead of a Minecraft pixel", a)
			}
		}
	}
}

func TestBundledSignFontUsesMinecraftMetrics(t *testing.T) {
	source, err := assets.Vanilla()
	if err != nil {
		t.Fatal(err)
	}
	c, err := LoadCatalogFS(source)
	if err != nil {
		t.Fatal(err)
	}
	narrow, _ := c.textTile("iiii", color.NRGBA{255, 255, 255, 255}, false)
	wide, _ := c.textTile("WWWW", color.NRGBA{255, 255, 255, 255}, false)
	if wide.w <= narrow.w*2 {
		t.Fatalf("shipped EXE still uses fallback font: %d vs %d", narrow.w, wide.w)
	}
}
