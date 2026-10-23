package scene

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"strings"
	"sync"
)

var textFontOnce sync.Once
var parsedTextFont *opentype.Font
var chatColors = []color.NRGBA{{0, 0, 0, 255}, {0, 0, 170, 255}, {0, 170, 0, 255}, {0, 170, 170, 255}, {170, 0, 0, 255}, {170, 0, 170, 255}, {255, 170, 0, 255}, {170, 170, 170, 255}, {85, 85, 85, 255}, {85, 85, 255, 255}, {85, 255, 85, 255}, {85, 255, 255, 255}, {255, 85, 85, 255}, {255, 85, 255, 255}, {255, 255, 85, 255}, {255, 255, 255, 255}}

func plainRawText(text string) string {
	if !strings.HasPrefix(strings.TrimSpace(text), "{") {
		return text
	}
	var doc struct {
		Raw []struct {
			Text      string `json:"text"`
			Translate string `json:"translate"`
		} `json:"rawtext"`
	}
	if json.Unmarshal([]byte(text), &doc) != nil {
		return text
	}
	var out strings.Builder
	for _, part := range doc.Raw {
		out.WriteString(part.Text)
		out.WriteString(part.Translate)
	}
	return out.String()
}

func (c *Catalog) privateGlyph(r rune) image.Image {
	if r < 0xe000 || r > 0xf8ff {
		return nil
	}
	key := fmt.Sprintf("font/glyph_%02X.png", r>>8)
	for _, source := range c.sources {
		raw, e := fs.ReadFile(source, key)
		if e != nil {
			continue
		}
		im, e := png.Decode(bytes.NewReader(raw))
		if e != nil {
			return nil
		}
		b := im.Bounds()
		w, h := b.Dx()/16, b.Dy()/16
		if w < 1 || h < 1 {
			return nil
		}
		cell := image.Rect(int(r&15)*w, int((r>>4)&15)*h, int(r&15+1)*w, int((r>>4)&15+1)*h)
		out := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(out, out.Bounds(), im, cell.Min, draw.Src)
		return out
	}
	return nil
}

func (c *Catalog) textTile(text string, base color.NRGBA, background bool) (tile, float32) {
	return c.textTileSlot(text, base, background, "")
}

func (c *Catalog) textTileSlot(text string, base color.NRGBA, background bool, slot string) (tile, float32) {
	text = plainRawText(text)
	runes := []rune(text)
	if len(runes) > 2048 {
		runes = runes[:2048]
	}
	text = string(runes)
	key := fmt.Sprintf("text#model:%x", sha256.Sum256([]byte(fmt.Sprintf("%v/%v/%s", base, background, text))))
	contentKey := key
	if slot != "" {
		key = "text#model:slot:" + slot
	}
	if t, ok := c.tiles[key]; ok && (slot == "" || c.textContents[key] == contentKey) {
		return t, float32(t.w) / float32(t.h)
	}
	textFontOnce.Do(func() {
		parsedTextFont, _ = opentype.Parse(gomono.TTF)
	})
	textFont, _ := opentype.NewFace(parsedTextFont, &opentype.FaceOptions{Size: 14, DPI: 72, Hinting: font.HintingFull})
	defer textFont.Close()
	lines := strings.Split(text, "\n")
	if len(lines) > 12 {
		lines = lines[:12]
	}
	width := 1
	for _, line := range lines {
		n := 0
		skip := false
		for _, r := range line {
			if skip {
				skip = false
				continue
			}
			if r == '§' {
				skip = true
				continue
			}
			if r >= 0xe000 && r <= 0xf8ff {
				n += 16
			} else {
				n += 9
			}
		}
		width = max(width, n)
	}
	width = min(width+4, 1024)
	height := len(lines)*17 + 4
	if slot != "" {
		width = ((width + 63) / 64) * 64
		if old, ok := c.tiles[key]; ok {
			width = max(width, old.w)
			height = max(height, old.h)
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	if background {
		draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{0, 0, 0, 85}), image.Point{}, draw.Src)
	}
	ink := base
	bold := false
	for row, line := range lines {
		rs := []rune(line)
		advance := 0
		for i := 0; i < len(rs); i++ {
			r := rs[i]
			if r == '§' && i+1 < len(rs) {
				i++
				continue
			}
			if r >= 0xe000 && r <= 0xf8ff {
				advance += 16
			} else {
				advance += 9
			}
		}
		x := max(2, (width-advance)/2)
		y := row*17 + 15
		for i := 0; i < len(rs) && x < width-2; i++ {
			r := rs[i]
			if r == '§' && i+1 < len(rs) {
				i++
				code := strings.IndexRune("0123456789abcdef", rs[i])
				if code >= 0 {
					ink = chatColors[code]
					bold = false
				} else if rs[i] == 'r' {
					ink = base
					bold = false
				} else if rs[i] == 'l' {
					bold = true
				}
				continue
			}
			if glyph := c.privateGlyph(r); glyph != nil {
				b := glyph.Bounds()
				for gy := 0; gy < 16; gy++ {
					for gx := 0; gx < 16; gx++ {
						img.Set(x+gx, y-13+gy, glyph.At(gx*b.Dx()/16, gy*b.Dy()/16))
					}
				}
				x += 16
				continue
			}
			d := font.Drawer{Dst: img, Src: image.NewUniform(ink), Face: textFont, Dot: fixed.P(x, y)}
			d.DrawString(string(r))
			if bold {
				d.Dot = fixed.P(x+1, y)
				d.DrawString(string(r))
			}
			x += 9
		}
	}
	t := c.addTileSized(key, img, false, width, height)
	if c.textContents == nil {
		c.textContents = map[string]string{}
	}
	c.textContents[key] = contentKey
	return t, float32(width) / float32(height)
}

func imageQuad(tex tile, width, height, z float32, mode string) modelQuad {
	q := texturedQuad([4][3]float32{{-width / 2, 0, z}, {width / 2, 0, z}, {width / 2, height, z}, {-width / 2, height, z}}, tex)
	q.mode = mode
	for i, uv := range [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}} {
		q.uv[i] = [2]float32{(float32(tex.x) + uv[0]*float32(tex.w)) / atlasSize, (float32(tex.y) + uv[1]*float32(tex.h)) / atlasSize}
	}
	return q
}

func (s *Store) emitVisuals(pos world.ChunkPos, parts map[string]*Part, out *ChunkMesh) {
	emit := func(q modelQuad, offset [3]float32) {
		p := parts[q.mode]
		if p == nil {
			p = newPart(q.mode)
			parts[q.mode] = p
		}
		emitModelQuad(p, q, offset)
		if q.texture.missing {
			out.MissingTextures++
		}
	}
	for p, data := range s.signs {
		ch := s.chunks[pos]
		rid := ch.Block(uint8(p[0]&15), int16(p[1]), uint8(p[2]&15), 0)
		name, props, _ := s.registry.RuntimeIDToState(rid)
		for side, key := range []string{"FrontText", "BackText"} {
			face, _ := data[key].(map[string]any)
			text, _ := face["Text"].(string)
			if side == 0 && text == "" {
				text, _ = data["Text"].(string)
			}
			if text == "" {
				continue
			}
			value := uint32(int64(numeric(face["SignTextColor"])))
			ink := color.NRGBA{uint8(value >> 16), uint8(value >> 8), uint8(value), 255}
			tex, aspect := s.catalog.textTile(text, ink, false)
			w, h := float32(.9), float32(.44)
			if w/h > aspect {
				w = h * aspect
			} else {
				h = w / aspect
			}
			q := imageQuad(tex, w, h, 0, "mask")
			if truth(face["IgnoreLighting"]) {
				q.mode = "unlit"
			}
			y, z := float32(.75)-h/2, float32(9.0/16+.001)
			if strings.Contains(name, "wall_sign") {
				y = .53125 - h/2
				z = .126
			}
			if strings.Contains(name, "hanging_sign") {
				y = .46875 - h/2
			}
			if side == 1 {
				z = 7.0/16 - .001
				if strings.Contains(name, "wall_sign") {
					z = -.001
				}
				for i, v := range q.points {
					q.points[i][0] = -v[0]
				}
				q.normal[2] = -1
			}
			for i := range q.points {
				q.points[i][0] += .5
				q.points[i][1] += y
				q.points[i][2] = z
			}
			m := blockModel{quads: []modelQuad{q}}
			transformQuads(&m, signMatrix(name, props))
			emit(m.quads[0], [3]float32{float32(p[0] - int(pos[0])*16), float32(p[1]), float32(p[2] - int(pos[1])*16)})
		}
	}
	if s.omitActors {
		return
	}
	for _, a := range s.visualActors {
		if !actorAllowed(a) {
			continue
		}
		offset := [3]float32{a.Position[0] - float32(pos[0])*16, a.Position[1], a.Position[2] - float32(pos[1])*16}
		model := s.catalog.actorModel(a)
		if model.approx {
			out.ApproximateBlocks++
		}
		height := a.Height
		for _, q := range model.quads {
			emit(q, offset)
			for _, p := range q.points {
				height = max(height, p[1])
			}
		}
		if a.Name != "" && a.NameVisible {
			tex, aspect := s.catalog.textTileSlot(a.Name, color.NRGBA{255, 255, 255, 255}, true, fmt.Sprint(a.ID))
			h := float32(tex.h) / 17 * .25
			w := h * aspect
			q := imageQuad(tex, w, h, 0, "billboard")
			m := blockModel{quads: []modelQuad{q}}
			transformQuads(&m, rotationMatrix([3]float32{}, [3]float32{0, -a.Yaw, 0}))
			off := offset
			off[1] += height + .25
			emit(m.quads[0], off)
		}
	}
}
