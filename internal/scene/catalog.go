package scene

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dblezek/tga"
	"github.com/tailscale/hujson"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path"
	"strings"
)

const atlasSize = 2048
const cellSize = 18

func nearGray(c color.NRGBA) bool {
	return abs(int(c.R)-int(c.G)) <= 8 && abs(int(c.G)-int(c.B)) <= 8 && abs(int(c.R)-int(c.B)) <= 8
}

type tile struct {
	x, y    int
	w, h    int
	alpha   bool
	missing bool
}
type Catalog struct {
	sources                   []fs.FS
	blocks                    map[string]map[string]string
	modes                     map[string]string
	terrain                   map[string]string
	tiles                     map[string]tile
	atlas                     *image.NRGBA
	revision                  uint64
	next                      int
	encoded                   string
	encodedRevision           uint64
	Source                    string
	changes                   []atlasChange
	changeIndex               map[[2]int]int
	atlasX, atlasY, rowHeight int
	geometries                map[string]blockGeometry
	definitions               map[string]blockDefinition
	actors                    map[string]actorDefinition
	renderControllers         map[string]map[string]any
	animations                map[string]map[string]any
	animationControllers      map[string]map[string]any
	textContents              map[string]string
}

func NewCatalog() *Catalog {
	return &Catalog{atlasX: cellSize, rowHeight: cellSize, geometries: map[string]blockGeometry{}, definitions: map[string]blockDefinition{}, changes: []atlasChange{{revision: 1, x: 0, y: 0}}, blocks: map[string]map[string]string{}, modes: map[string]string{}, terrain: map[string]string{}, tiles: map[string]tile{"": {x: 1, y: 1, w: 16, h: 16, missing: true}}, next: 1, revision: 1, Source: "No vanilla assets imported"}
}

func (c *Catalog) SetBlockComponents(name string, components map[string]any) {
	short := strings.TrimPrefix(name, "minecraft:")
	c.definitions[short] = blockDefinition{Components: components}
	materials := materialTable(components)
	faces := map[string]string{}
	for _, face := range faceNames {
		value := resolveMaterial(materials, face)
		if texture, ok := value["texture"].(string); ok {
			faces[face] = texture
		}
		if method, _ := value["render_method"].(string); method == "blend" {
			c.modes[short] = "blend"
		} else if strings.HasPrefix(method, "alpha_test") && c.modes[short] != "blend" {
			c.modes[short] = "mask"
		}
	}
	if len(faces) > 0 {
		c.blocks[short] = faces
	}
}

func (c *Catalog) ensureAtlas() {
	if c.atlas != nil {
		return
	}
	c.atlas = image.NewNRGBA(image.Rect(0, 0, atlasSize, atlasSize))
	for y := -1; y <= 16; y++ {
		for x := -1; x <= 16; x++ {
			sx, sy := max(0, min(15, x)), max(0, min(15, y))
			v := color.NRGBA{R: 220, B: 220, A: 255}
			if (sx/8+sy/8)%2 == 0 {
				v = color.NRGBA{R: 35, G: 35, B: 35, A: 255}
			}
			c.atlas.SetNRGBA(x+1, y+1, v)
		}
	}
}

func (c *Catalog) AddPack(f fs.FS) error {
	c.sources = append([]fs.FS{f}, c.sources...)
	var blocks map[string]json.RawMessage
	if b, e := fs.ReadFile(f, "blocks.json"); e == nil {
		if b, e = hujson.Standardize(b); e != nil {
			return e
		}
		if e = json.Unmarshal(b, &blocks); e != nil {
			return fmt.Errorf("blocks.json: %w", e)
		}
		for name, raw := range blocks {
			var v struct {
				Textures json.RawMessage `json:"textures"`
			}
			if json.Unmarshal(raw, &v) != nil {
				continue
			}
			var all string
			faces := map[string]string{}
			if json.Unmarshal(v.Textures, &all) == nil {
				for _, face := range faceNames {
					faces[face] = all
				}
			} else {
				_ = json.Unmarshal(v.Textures, &faces)
			}
			if len(faces) > 0 {
				c.blocks[strings.TrimPrefix(name, "minecraft:")] = faces
			}
		}
	}
	var terrain struct {
		Data map[string]struct {
			Textures json.RawMessage `json:"textures"`
		} `json:"texture_data"`
	}
	if b, e := fs.ReadFile(f, "textures/terrain_texture.json"); e == nil {
		if b, e = hujson.Standardize(b); e != nil {
			return e
		}
		if e = json.Unmarshal(b, &terrain); e != nil {
			return fmt.Errorf("terrain_texture.json: %w", e)
		}
		for k, v := range terrain.Data {
			if p := texturePath(v.Textures); p != "" {
				c.terrain[k] = p
			}
		}
	}
	c.tiles = map[string]tile{"": c.tiles[""]}
	c.next = 1
	c.atlasX = cellSize
	c.atlasY = 0
	c.rowHeight = cellSize
	c.revision++
	c.changes = []atlasChange{{revision: c.revision, x: 0, y: 0}}
	if err := c.loadBlockModels(f); err != nil {
		return err
	}
	return c.loadActorAssets(f)
}

func texturePath(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
		return texturePath(arr[0])
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) == nil {
		if p := texturePath(m["path"]); p != "" {
			return p
		}
		var v []json.RawMessage
		if json.Unmarshal(m["variations"], &v) == nil && len(v) > 0 {
			return texturePath(v[0])
		}
	}
	return ""
}

func (c *Catalog) addTile(key string, img image.Image, missing bool) tile {
	return c.addTileSized(key, img, missing, 16, 16)
}

func (c *Catalog) addTileSized(key string, img image.Image, missing bool, width, height int) tile {
	c.ensureAtlas()
	aw, ah := ((width+2+cellSize-1)/cellSize)*cellSize, ((height+2+cellSize-1)/cellSize)*cellSize
	existing, reuse := c.tiles[key]
	reuse = reuse && existing.w == width && existing.h == height
	if !reuse {
		if c.atlasX+aw > atlasSize {
			c.atlasX = 0
			c.atlasY += c.rowHeight
			c.rowHeight = 0
		}
		if c.atlasY+ah > atlasSize {
			return c.tiles[""]
		}
		c.atlasX += aw
		c.rowHeight = max(c.rowHeight, ah)
		c.next++
	}
	t := existing
	if !reuse {
		t = tile{x: c.atlasX - aw + 1, y: c.atlasY + 1, w: width, h: height, missing: missing}
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if h > w && !strings.Contains(key, "#model:") {
		h = w
	}

	tint := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	selectiveTint := false
	vegetation := strings.Contains(key, "grass_top") || strings.Contains(key, "grass_bottom") || (strings.Contains(key, "leaves") && !strings.Contains(key, "pale_oak")) || strings.Contains(key, "tallgrass") || strings.Contains(key, "fern") || strings.Contains(key, "/vine")
	if vegetation {
		gray, visible := 0, 0
		for sy := 0; sy < h; sy++ {
			for sx := 0; sx < w; sx++ {
				v := color.NRGBAModel.Convert(img.At(bounds.Min.X+sx, bounds.Min.Y+sy)).(color.NRGBA)
				if v.A > 0 {
					visible++
					if nearGray(v) {
						gray++
					}
				}
			}
		}
		if visible > 0 && gray*4 >= visible*3 {
			selectiveTint = true
			tint = color.NRGBA{R: 145, G: 189, B: 89, A: 255}
			if strings.Contains(key, "leaves") {
				tint = color.NRGBA{R: 119, G: 171, B: 47, A: 255}
			}
		}
	}
	if explicit, ok := materialTint(key); ok {
		tint = explicit
		selectiveTint = false
	}
	for y := -1; y <= height; y++ {
		for x := -1; x <= width; x++ {
			sx, sy := max(0, min(width-1, x)), max(0, min(height-1, y))
			v := color.NRGBAModel.Convert(img.At(bounds.Min.X+sx*w/width, bounds.Min.Y+sy*h/height)).(color.NRGBA)
			if strings.HasSuffix(key, "#grass-mask") || strings.Contains(key, "#masked-tint") {
				maskTint := tint
				if strings.HasSuffix(key, "#grass-mask") {
					maskTint = color.NRGBA{145, 189, 89, 255}
				}
				v.R = maskedTint(v.R, maskTint.R, v.A)
				v.G = maskedTint(v.G, maskTint.G, v.A)
				v.B = maskedTint(v.B, maskTint.B, v.A)
				v.A = 255
			} else {
				if !selectiveTint || nearGray(v) {
					v.R = uint8(uint16(v.R) * uint16(tint.R) / 255)
					v.G = uint8(uint16(v.G) * uint16(tint.G) / 255)
					v.B = uint8(uint16(v.B) * uint16(tint.B) / 255)
				}
			}
			if v.A < 255 {
				t.alpha = true
			}
			c.atlas.SetNRGBA(t.x+x, t.y+y, v)
		}
	}
	c.tiles[key] = t
	c.revision++
	for y := 0; y < ah; y += cellSize {
		for x := 0; x < aw; x += cellSize {
			c.recordAtlasChange(t.x-1+x, t.y-1+y)
		}
	}
	return t
}

var faceNames = []string{"east", "west", "up", "down", "south", "north"}

func (c *Catalog) texture(name string, face int) tile {
	short := strings.TrimPrefix(name, "minecraft:")

	if short == "grass_block" {
		short = "grass"
	}
	key := short
	if faces := c.blocks[short]; faces != nil {
		if s := faces[faceNames[face]]; s != "" {
			key = s
		} else if s := faces["side"]; s != "" {
			key = s
		} else if s := faces["all"]; s != "" {
			key = s
		} else if s := faces["up"]; s != "" {
			key = s
		}
	}
	p := c.terrain[key]
	if p == "" && key == short && short == "iron_chain" {
		key = "chain1"
		if face == 2 || face == 3 {
			key = "chain2"
		}
		p = c.terrain[key]
	}
	if p == "" {
		p = key
		if !strings.HasPrefix(p, "textures/") {
			p = "textures/blocks/" + key
		}
	}
	cacheKey := p
	if short == "water" || short == "flowing_water" {
		cacheKey += "#tint:water#"
	}
	if short == "grass" && face != 2 && face != 3 {
		cacheKey += "#grass-mask"
	}
	if t, ok := c.tiles[cacheKey]; ok {
		return t
	}
	if fs.ValidPath(p) {
		for _, source := range c.sources {
			for _, ext := range []string{"", ".png", ".tga"} {
				b, e := fs.ReadFile(source, p+ext)
				if e != nil {
					continue
				}
				var img image.Image
				if strings.HasSuffix(p+ext, ".tga") {
					img, e = tga.Decode(bytes.NewReader(b))
				} else {
					img, _, e = image.Decode(bytes.NewReader(b))
				}
				if e == nil && img.Bounds().Dx() > 0 && img.Bounds().Dy() > 0 {
					return c.addTile(cacheKey, img, false)
				}
			}
		}
	}
	t := c.tiles[""]
	t.missing = true
	c.tiles[cacheKey] = t
	return t
}

func (c *Catalog) atlasURL() string {
	c.ensureAtlas()
	if c.encodedRevision != c.revision {
		var b bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}
		_ = encoder.Encode(&b, c.atlas)
		c.encoded = "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
		c.encodedRevision = c.revision
	}
	return c.encoded
}

func LoadCatalog(directory string) (*Catalog, error) {
	c := NewCatalog()
	if err := c.AddPack(os.DirFS(directory)); err != nil {
		return nil, err
	}
	var provenance struct {
		Source string `json:"source"`
	}
	b, e := os.ReadFile(path.Join(directory, "provenance.json"))
	if e != nil {
		return nil, fmt.Errorf("3D assets are missing; run tools/prepare-3d-assets.py: %w", e)
	}
	if e = json.Unmarshal(b, &provenance); e != nil {
		return nil, e
	}
	c.Source = provenance.Source
	return c, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func maskedTint(value, tint, mask uint8) uint8 {
	return uint8(uint32(value) * (255*uint32(255-mask) + uint32(tint)*uint32(mask)) / (255 * 255))
}
