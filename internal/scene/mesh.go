package scene

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
	"strings"
)

type box struct{ min, max [3]float32 }
type blockModel struct {
	name                string
	boxes               []box
	mode                string
	cross, full, approx bool
	quads               []modelQuad
	plantFace           int
	sunflowerHead       bool
	connect             string
	fluid               string
	gateAxis            int
}

var unit = box{[3]float32{0, 0, 0}, [3]float32{1, 1, 1}}

var plants = map[string]bool{"poppy": true, "dandelion": true, "allium": true, "blue_orchid": true, "azure_bluet": true, "lily_of_the_valley": true, "oxeye_daisy": true, "red_tulip": true, "orange_tulip": true, "white_tulip": true, "pink_tulip": true, "wither_rose": true, "torchflower": true, "short_dry_grass": true, "tall_dry_grass": true, "large_fern": true}

func number(v any) int {
	switch n := v.(type) {
	case int32:
		return int(n)
	case int:
		return n
	case uint8:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
func flag(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return number(v) != 0
}

func (s *Store) model(rid uint32) blockModel {
	name, props, ok := s.registry.RuntimeIDToState(rid)
	if !ok {
		name = "unknown"
	}
	if model, ok := s.catalog.customModel(name, props); ok {
		return model
	}
	m := blockModel{name: name, mode: "opaque", boxes: []box{unit}, full: true, plantFace: 2}
	n := strings.TrimPrefix(name, "minecraft:")
	if n == "fence_gate" || strings.HasSuffix(n, "_fence_gate") {
		return fenceGateModel(name, props)
	}
	if strings.Contains(n, "sapling") {
		return s.catalog.saplingModel(name)
	}
	if n == "vine" {
		return s.catalog.vineModel(number(props["vine_direction_bits"]))
	}
	if isCrystal(n) {
		face, _ := props["minecraft:block_face"].(string)
		return s.catalog.crystalModel(name, face)
	}
	if n == "flower_pot" {
		return s.catalog.flowerPotModel("")
	}
	if n == "anvil" || strings.HasSuffix(n, "_anvil") {
		return s.catalog.texturedAnvilModel(name, props)
	}
	if n == "lantern" || n == "soul_lantern" || strings.HasSuffix(n, "copper_lantern") {
		return s.catalog.lanternModel(name, flag(props["hanging"]))
	}
	if mode := s.catalog.modes[n]; mode != "" {
		m.mode = mode
		m.full = mode == "opaque"
	}
	if n == "air" || n == "cave_air" || n == "void_air" || n == "structure_void" || n == "invisible_bedrock" || n == "barrier" || n == "light_block" || strings.HasPrefix(n, "light_block_") {
		m.boxes = nil
		m.full = false
		return m
	}
	if strings.Contains(n, "glass") || strings.Contains(n, "water") || strings.Contains(n, "ice") || strings.Contains(n, "portal") {
		m.mode = "blend"
		m.full = false
	}
	if strings.Contains(n, "leaves") {
		m.mode = "mask"
		m.full = false
	}
	if strings.Contains(n, "slab") && !strings.Contains(n, "double") {
		m.full = false
		m.boxes[0].max[1] = .5
		if flag(props["top_slot_bit"]) || props["minecraft:vertical_half"] == "top" {
			m.boxes[0].min[1] = .5
			m.boxes[0].max[1] = 1
		}
	}
	if strings.Contains(n, "stairs") {
		m.full = false
		a, b := unit, unit
		a.max[1] = .5
		b.min[1] = .5
		if flag(props["upside_down_bit"]) {
			a.min[1] = .5
			a.max[1] = 1
			b.min[1] = 0
			b.max[1] = .5
		}
		switch number(props["weirdo_direction"]) {
		case 0:
			b.min[0] = .5
		case 1:
			b.max[0] = .5
		case 2:
			b.min[2] = .5
		case 3:
			b.max[2] = .5
		}
		m.boxes = []box{a, b}
	}
	if n == "snow_layer" {
		m.full = false
		m.boxes[0].max[1] = float32(number(props["height"])+1) / 8
	}
	if strings.Contains(n, "carpet") {
		m.full = false
		m.boxes[0].max[1] = 1.0 / 16
	}
	if n == "water" || n == "flowing_water" || n == "lava" || n == "flowing_lava" {
		m.full = false
		m.mode = "blend"
		if strings.Contains(n, "lava") {
			m.mode = "opaque"
		}
		depth := number(props["liquid_depth"])
		m.fluid = strings.TrimPrefix(n, "flowing_")
		m.boxes[0].max[1] = float32(8-(depth&7)) / 9
		if depth&8 != 0 {
			m.boxes[0].max[1] = 1
		}
	}
	if strings.HasSuffix(n, "pressure_plate") {
		h := float32(1.0 / 16)
		if number(props["redstone_signal"]) > 0 || flag(props["pressed_bit"]) {
			h = 1.0 / 32
		}
		m.full = false
		m.boxes = []box{{[3]float32{1.0 / 16, 0, 1.0 / 16}, [3]float32{15.0 / 16, h, 15.0 / 16}}}
	}
	if strings.Contains(n, "pane") || n == "iron_bars" {
		m.full = false
		m.approx = true
		m.connect = "pane"
		m.boxes = []box{{[3]float32{.4375, 0, .4375}, [3]float32{.5625, 1, .5625}}}
	}
	if strings.Contains(n, "fence") {
		m.full = false
		m.boxes = []box{{[3]float32{.375, 0, .375}, [3]float32{.625, 1, .625}}}
		m.approx = true
		if !strings.Contains(n, "fence_gate") {
			m.connect = "fence"
		}
	}
	if strings.HasSuffix(n, "_wall") {
		m.full = false
		m.connect = "wall"
		m.boxes = []box{{[3]float32{.25, 0, .25}, [3]float32{.75, 1, .75}}}
		m.approx = true
	}
	if plants[n] || strings.Contains(n, "flower") || strings.Contains(n, "sapling") || n == "tallgrass" || n == "short_grass" || n == "fern" || n == "deadbush" || n == "red_mushroom" || n == "brown_mushroom" || n == "wheat" {
		m.full = false
		m.cross = true
		m.mode = "mask"
		m.boxes = nil
	}
	if n == "sunflower" || n == "rose_bush" || n == "peony" || n == "lilac" || n == "tall_grass" || n == "large_fern" {
		m.full = false
		m.cross = true
		m.boxes = nil
		m.mode = "mask"
		m.plantFace = 3
		if flag(props["upper_block_bit"]) || props["minecraft:vertical_half"] == "top" {
			m.plantFace = 2
			m.sunflowerHead = n == "sunflower"
		}
	}
	if n == "closed_eyeblossom" || n == "open_eyeblossom" || n == "firefly_bush" || n == "seagrass" || n == "web" || n == "reeds" || n == "sugar_cane" || n == "bush" || n == "soul_fire" || n == "fire" {
		m.full = false
		m.cross = true
		m.boxes = nil
		m.mode = "mask"
	}
	if n == "chain" || strings.HasSuffix(n, "_chain") {
		return s.catalog.chainModel(name, props)
	}
	if n == "standing_sign" || n == "wall_sign" || strings.HasSuffix(n, "_sign") {
		return s.catalog.signModel(name, props)
	}
	if strings.HasSuffix(n, "trapdoor") {
		facing := cube.Direction((3 - number(props["direction"])) & 3)
		for _, d := range []cube.Direction{cube.North, cube.South, cube.West, cube.East} {
			if props["minecraft:cardinal_direction"] == d.String() {
				facing = d
			}
		}
		bounds := (model.Trapdoor{Facing: facing, Open: flag(props["open_bit"]), Top: flag(props["upside_down_bit"]) || props["minecraft:vertical_half"] == "top"}).BBox(cube.Pos{}, nil)[0]
		m.full = false
		m.mode = "mask"
		m.boxes = []box{{min: [3]float32{float32(bounds.Min()[0]), float32(bounds.Min()[1]), float32(bounds.Min()[2])}, max: [3]float32{float32(bounds.Max()[0]), float32(bounds.Max()[1]), float32(bounds.Max()[2])}}}
	}

	if (strings.Contains(n, "door") && !strings.HasSuffix(n, "trapdoor")) || strings.Contains(n, "rail") || strings.Contains(n, "torch") || strings.Contains(n, "sign") || strings.Contains(n, "wall") || strings.Contains(n, "chest") || n == "bed" || strings.HasSuffix(n, "_bed") {
		m.approx = true
	}
	if !strings.HasPrefix(name, "minecraft:") {
		m.approx = true
	}
	if m.full {
		for face := range directions {
			if s.catalog.texture(name, face).alpha {
				m.full = false
				m.mode = "mask"
				break
			}
		}
	}
	return m
}

var directions = [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}

var corners = [][4][3]int{
	{{1, 0, 1}, {1, 0, 0}, {1, 1, 0}, {1, 1, 1}}, {{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}},
	{{0, 1, 1}, {1, 1, 1}, {1, 1, 0}, {0, 1, 0}}, {{0, 0, 0}, {1, 0, 0}, {1, 0, 1}, {0, 0, 1}},
	{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}}, {{1, 0, 0}, {0, 0, 0}, {0, 1, 0}, {1, 1, 0}},
}

func (s *Store) mesh(pos world.ChunkPos, keep func(int32, int32) bool) ChunkMesh {
	out := ChunkMesh{X: pos[0], Z: pos[1], Parts: []Part{}}
	parts := map[string]*Part{}
	getModel := s.cachedModel
	getLayer := func(x, y, z int, layer uint8) *blockModel {
		p := world.ChunkPos{pos[0] + int32(x>>4), pos[1] + int32(z>>4)}
		if keep != nil && !keep(p[0], p[1]) {
			return &emptyBlockModel
		}
		c := s.chunks[p]
		if c == nil || y < c.Range()[0] || y > c.Range()[1] {
			return &emptyBlockModel
		}
		return getModel(c.Block(uint8(x&15), int16(y), uint8(z&15), layer))
	}
	get := func(x, y, z int) *blockModel { return getLayer(x, y, z, 0) }
	fluidAt := func(x, y, z int, fluid string) *blockModel {
		for layer := uint8(0); layer < 2; layer++ {
			m := getLayer(x, y, z, layer)
			if m.fluid == fluid {
				return m
			}
		}
		return &emptyBlockModel
	}
	c := s.chunks[pos]
	solid := s.solidField(c, get)
	deltas := [6]int{1, -1, 324, -324, 18, -18}
	for sy, sub := range c.Sub() {
		if s.ctx != nil && s.ctx.Err() != nil {
			return out
		}
		if sub.Empty() {
			continue
		}
		for x := 0; x < 16; x++ {
			for z := 0; z < 16; z++ {
				for dy := 0; dy < 16; dy++ {
					y := c.Range()[0] + sy*16 + dy
					for layer := uint8(0); layer < 2; layer++ {
						rid := c.Block(uint8(x), int16(y), uint8(z), layer)
						m := getModel(rid)
						if level := s.emissionLevel(rid, m.name); level > 0 {
							exposed := false
							for _, d := range [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
								neighbour := get(x+d[0], y+d[1], z+d[2])
								if !neighbour.full && neighbour.name != m.name {
									exposed = true
									break
								}
								if neighbour.name == "minecraft:air" {
									exposed = true
									break
								}
							}
							if exposed {
								out.Lights = append(out.Lights, SceneLight{Position: [3]float32{float32(x) + .5, float32(y) + .5, float32(z) + .5}, Level: level, Color: s.catalog.emissionColor(m.name)})
							}
						}
						if m.name == "minecraft:flower_pot" {
							if plant := s.potPlants[cube.Pos{int(pos[0])*16 + x, y, int(pos[1])*16 + z}]; plant != "" {
								copy := s.catalog.flowerPotModel(plant)
								m = &copy
							}
						}
						if m.fluid != "" && fluidAt(x, y+1, z, m.fluid).fluid != "" {
							copy := *m
							copy.boxes = []box{m.boxes[0]}
							copy.boxes[0].max[1] = 1
							m = &copy
						}
						if m.connect != "" {
							copy := *m
							copy.boxes = connectedBoxes(m, [4]*blockModel{get(x+1, y, z), get(x-1, y, z), get(x, y, z+1), get(x, y, z-1)})
							m = &copy
						}
						if len(m.boxes) == 0 && !m.cross && len(m.quads) == 0 {
							continue
						}
						if m.approx {
							out.ApproximateBlocks++
						}
						missing := false
						if len(m.quads) > 0 {
							for _, q := range m.quads {
								p := parts[q.mode]
								if p == nil {
									p = newPart(q.mode)
									parts[q.mode] = p
								}
								missing = missing || q.texture.missing
								emitModelQuad(p, q, [3]float32{float32(x), float32(y), float32(z)})
							}
						} else if m.full && len(m.boxes) == 1 && m.boxes[0] == unit {
							index := (y-c.Range()[0]+1)*324 + (z+1)*18 + x + 1
							for face, d := range directions {
								if solid[index+deltas[face]] != 0 {
									continue
								}
								t := s.catalog.texture(m.name, face)
								missing = missing || t.missing
								mode := m.mode
								if t.alpha && mode == "opaque" {
									mode = "mask"
								}
								part := parts[mode]
								if part == nil {
									part = newPart(mode)
									parts[mode] = part
								}
								var q [4][3]float32
								for i, v := range corners[face] {
									q[i] = [3]float32{float32(v[0]), float32(v[1]), float32(v[2])}
								}
								emitQuad(part, q, [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, [3]float32{float32(x), float32(y), float32(z)}, t)
							}
						} else if m.cross {
							for _, q := range [][4][3]float32{{{0, 0, 0}, {1, 0, 1}, {1, 1, 1}, {0, 1, 0}}, {{1, 0, 0}, {0, 0, 1}, {0, 1, 1}, {1, 1, 0}}} {
								t := s.catalog.texture(m.name, m.plantFace)
								missing = missing || t.missing
								p := parts["mask"]
								if p == nil {
									p = newPart("mask")
									parts["mask"] = p
								}
								emitQuad(p, q, [3]float32{0, 0, 1}, [3]float32{float32(x), float32(y), float32(z)}, t)
							}
							if m.sunflowerHead {
								t := s.catalog.texture(m.name, 0)
								missing = missing || t.missing
								emitQuad(parts["mask"], [4][3]float32{{.5, 0, 0}, {.5, 0, 1}, {.5, 1, 1}, {.5, 1, 0}}, [3]float32{1, 0, 0}, [3]float32{float32(x), float32(y), float32(z)}, t)
							}
						} else {
							for bi, b := range m.boxes {
								for face, d := range directions {

									axis := face / 2
									value := b.max[axis]
									if face%2 == 1 {
										value = b.min[axis]
									}
									covered := false
									for oi, other := range m.boxes {
										if oi == bi {
											continue
										}
										boundary := other.min[axis]
										if face%2 == 1 {
											boundary = other.max[axis]
										}
										if boundary != value {
											continue
										}
										cover := true
										for a := 0; a < 3; a++ {
											if a != axis && (other.min[a] > b.min[a] || other.max[a] < b.max[a]) {
												cover = false
											}
										}
										if cover {
											covered = true
											break
										}
									}
									if covered {
										continue
									}
									neighbor := get(x+d[0], y+d[1], z+d[2])
									atEdge := value == 0 || value == 1
									if m.fluid != "" {
										fluid := fluidAt(x+d[0], y+d[1], z+d[2], m.fluid)
										if fluid.fluid != "" {
											if face == 2 && b.max[1] == 1 {
												continue
											}
											if face == 3 {
												continue
											}
											if axis != 1 && (fluid.boxes[0].max[1] >= b.max[1] || fluidAt(x+d[0], y+1, z+d[2], m.fluid).fluid != "") {
												continue
											}
										}
									}
									sameShape := neighbor.name == m.name && len(neighbor.boxes) == 1 && len(m.boxes) == 1 && neighbor.boxes[0] == b
									if atEdge && (neighbor.full || (sameShape && (m.mode == "blend" || strings.Contains(m.name, "leaves")))) {
										continue
									}
									t := s.catalog.texture(m.name, face)
									missing = missing || t.missing
									mode := m.mode
									if t.alpha && mode == "opaque" {
										mode = "mask"
									}
									p := parts[mode]
									if p == nil {
										p = newPart(mode)
										parts[mode] = p
									}
									var q [4][3]float32
									for i, v := range corners[face] {
										for a := 0; a < 3; a++ {
											q[i][a] = b.min[a]
											if v[a] == 1 {
												q[i][a] = b.max[a]
											}
										}
									}
									normal := [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}
									emitBoxQuad(p, q, normal, [3]float32{float32(x), float32(y), float32(z)}, t, face)
								}
							}
						}
						if missing {
							out.MissingTextures++
						}
					}
				}
			}
		}
	}
	s.emitVisuals(pos, parts, &out)
	for _, mode := range []string{"opaque", "mask", "blend", "unlit", "billboard", "unlit-blend"} {
		if p := parts[mode]; p != nil {
			out.Parts = append(out.Parts, *p)
		}
	}
	return out
}

func emitQuad(p *Part, q [4][3]float32, normal, offset [3]float32, t tile) {
	base := uint32(len(p.Positions) / 3)
	uv := [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
	for i, v := range q {
		p.Positions = append(p.Positions, v[0]+offset[0], v[1]+offset[1], v[2]+offset[2])
		p.Normals = append(p.Normals, normal[:]...)
		p.UVs = append(p.UVs, (float32(t.x)+uv[i][0]*16)/atlasSize, (float32(t.y)+uv[i][1]*16)/atlasSize)
	}
	p.Indices = append(p.Indices, base, base+1, base+2, base, base+2, base+3)
}
