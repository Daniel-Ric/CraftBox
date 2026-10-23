package block

import (
	"math"
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type intRange struct {
	min, max int
}

func (i intRange) at() int {
	if i.min >= i.max-1 {
		return i.min
	}
	return i.min + rand.IntN(i.max-i.min)
}

func (i intRange) atInclusive() int {
	if i.max <= i.min {
		return i.min
	}
	return i.min + rand.IntN(i.max-i.min+1)
}

type weightedInt struct {
	value, weight int
}

type weightedInts []weightedInt

func (w weightedInts) at() int {
	total := 0
	for _, e := range w {
		total += e.weight
	}
	if total <= 0 {
		return 0
	}
	choice := rand.IntN(total)
	for _, e := range w {
		if choice -= e.weight; choice < 0 {
			return e.value
		}
	}
	return w[len(w)-1].value
}

type heightRange struct {
	base      int
	intervals []int

	inclusive bool
}

func (h heightRange) at() int {
	height := h.base
	for _, interval := range h.intervals {
		if h.inclusive {
			height += rand.IntN(interval + 1)
			continue
		}
		if interval > 0 {
			height += rand.IntN(interval)
		}
	}
	return height
}

type treeChance float64

func (c treeChance) roll() bool {
	if c >= 100 {
		return true
	}
	if c <= 0 {
		return false
	}
	return float64(c) > rand.Float64()*100
}

var treeDirections = [4]cube.Pos{{0, 0, 1}, {-1, 0, 0}, {0, 0, -1}, {1, 0, 0}}

func treeSoil(b world.Block) bool {
	switch b.(type) {
	case Grass, Dirt, Podzol, Farmland, Mud, MuddyMangroveRoots:
		return true
	}
	return false
}

func treeGrowsThrough(tx *world.Tx, pos cube.Pos) bool {
	if pos.OutOfBounds(tx.Range()) {
		return false
	}
	switch b := tx.Block(pos).(type) {
	case Air, Leaves, Vines, Sapling, ShortGrass, Fern, DoubleTallGrass:
		return true
	default:
		return treeSoil(b)
	}
}

func treeReplaceable(tx *world.Tx, pos cube.Pos, b world.Block) bool {
	if pos.OutOfBounds(tx.Range()) {
		return false
	}
	switch tx.Block(pos).(type) {
	case Air, Leaves, Vines, Sapling:
		return true
	}
	return replaceableWith(tx, pos, b)
}

func treeLeafReplaceable(tx *world.Tx, pos cube.Pos) bool {
	if pos.OutOfBounds(tx.Range()) {
		return false
	}
	switch tx.Block(pos).(type) {
	case Air, Leaves:
		return true
	}
	return false
}

func treeAir(tx *world.Tx, pos cube.Pos) bool {
	if pos.OutOfBounds(tx.Range()) {
		return false
	}
	_, air := tx.Block(pos).(Air)
	return air
}

func treeLog(tx *world.Tx, pos cube.Pos, wood WoodType, axis cube.Axis) bool {
	l := Log{Wood: wood, Axis: axis}
	if !treeReplaceable(tx, pos, l) {
		return false
	}
	tx.SetBlock(pos, l, nil)
	return true
}

func treeLeaf(tx *world.Tx, pos cube.Pos, leaves LeavesType, replaceable func(*world.Tx, cube.Pos) bool) bool {
	if !replaceable(tx, pos) {
		return false
	}
	tx.SetBlock(pos, Leaves{Type: leaves}, nil)
	return true
}

func poplarLeafReplaceable(tx *world.Tx, pos cube.Pos) bool {
	switch tx.Block(pos).(type) {
	case ShelfMushroom, RedShrub:
		return !pos.OutOfBounds(tx.Range())
	}
	return treeLeafReplaceable(tx, pos)
}

func treeRoom(tx *world.Tx, pos cube.Pos, height int) bool {
	if !treeSoil(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}
	for y := pos[1]; y <= pos[1]+height+1; y++ {
		r := 1
		if y == pos[1] {
			r = 0
		}
		if y >= pos[1]+height-1 {
			r = 2
		}
		for x := pos[0] - r; x <= pos[0]+r; x++ {
			for z := pos[2] - r; z <= pos[2]+r; z++ {
				if !treeGrowsThrough(tx, cube.Pos{x, y, z}) {
					return false
				}
			}
		}
	}
	return true
}

type vineDecoration struct {
	chance treeChance

	steps int
}

func treeVines(tx *world.Tx, pos cube.Pos, d vineDecoration, sides [4]bool) {
	if d.chance <= 0 || d.steps <= 0 {
		return
	}
	for i, v := range [4]struct {
		side cube.Pos
		vine Vines
	}{
		{cube.Pos{-1}, Vines{EastDirection: true}},
		{cube.Pos{1}, Vines{WestDirection: true}},
		{cube.Pos{0, 0, -1}, Vines{SouthDirection: true}},
		{cube.Pos{0, 0, 1}, Vines{NorthDirection: true}},
	} {
		if !sides[i] || !d.chance.roll() {
			continue
		}
		at := pos.Add(v.side)
		if !treeAir(tx, at) {
			continue
		}

		for range d.steps {
			tx.SetBlock(at, v.vine, nil)
			if at = at.Side(cube.FaceDown); !treeAir(tx, at) {
				break
			}
		}
	}
}

type treeCanopy interface {
	place(tx *world.Tx, origin cube.Pos, width int, leaves LeavesType)
}

type treeTrunk interface {
	height() int

	place(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) bool
}

type simpleCanopy struct {
	offset intRange

	minWidth int

	slopeRun  int
	slopeRise float64

	variation []treeChance

	vines vineDecoration
}

func (c simpleCanopy) radius(offset int) int {
	slope := func(o int) int {
		return int(float64(o*c.slopeRun) * c.slopeRise)
	}
	return c.minWidth + slope(c.offset.max) - slope(offset)
}

func (c simpleCanopy) place(tx *world.Tx, origin cube.Pos, _ int, leaves LeavesType) {
	for offset := c.offset.min; offset <= c.offset.max; offset++ {
		r, variation := c.radius(offset), c.variation[min(offset-c.offset.min, len(c.variation)-1)]
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {

				if abs(dx) == r && abs(dz) == r && variation.roll() {
					continue
				}
				treeLeaf(tx, origin.Add(cube.Pos{dx, offset, dz}), leaves, treeLeafReplaceable)
			}
		}
	}
	if c.vines.chance <= 0 {
		return
	}

	for offset := c.offset.min; offset <= c.offset.max; offset++ {
		r := c.radius(offset)
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				at := origin.Add(cube.Pos{dx, offset, dz})
				if l, ok := tx.Block(at).(Leaves); ok && l.Type == leaves {
					treeVines(tx, at, c.vines, [4]bool{true, true, true, true})
				}
			}
		}
	}
}

type spruceCanopy struct {
	lowerOffset, upperOffset intRange

	maxRadius intRange
}

func (c spruceCanopy) place(tx *world.Tx, origin cube.Pos, _ int, leaves LeavesType) {

	ground := origin[1]
	for {
		if ground <= tx.Range().Min() {
			return
		}
		if treeSoil(tx.Block(cube.Pos{origin[0], ground - 1, origin[2]})) {
			break
		}
		ground--
	}

	lower, upper, widest := c.lowerOffset.at(), c.upperOffset.at(), c.maxRadius.at()
	radius := rand.IntN(max(1, c.maxRadius.max-c.maxRadius.min))
	reach, narrowest := c.maxRadius.max-c.maxRadius.min-1, 0

	top := origin[1] + upper
	for row := 0; row < top-(ground+lower)+1; row++ {
		y := top - row
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if radius != 0 && abs(dx) == radius && abs(dz) == radius {
					continue
				}
				treeLeaf(tx, cube.Pos{origin[0] + dx, y, origin[2] + dz}, leaves, treeLeafReplaceable)
			}
		}
		next := min(reach+1, widest)
		if reach <= radius {
			radius, narrowest, reach = narrowest, 1, next
			continue
		}
		radius++
	}
}

type roofedCanopy struct {
	height int

	coreWidth int

	outerRadius, innerRadius int
}

func (c roofedCanopy) layers() int {
	return max(1, c.height-3)
}

func (c roofedCanopy) place(tx *world.Tx, origin cube.Pos, width int, leaves LeavesType) {
	size := width
	if size <= 0 {
		size = c.coreWidth
	}
	r := c.outerRadius
	for dx := -r; dx <= size-1+r; dx++ {
		for dz := -r; dz <= size-1+r; dz++ {
			treeLeaf(tx, origin.Add(cube.Pos{dx, -1, dz}), leaves, treeAir)

			if outX, outZ := max(-dx, dx-size+1, 0), max(-dz, dz-size+1, 0); outX+outZ >= r*2-1 {
				continue
			}
			treeLeaf(tx, origin.Add(cube.Pos{dx, c.layers(), dz}), leaves, treeAir)
		}
	}

	r = c.innerRadius
	for y := 0; y < c.layers(); y++ {
		for dx := -r; dx < size+r; dx++ {
			for dz := -r; dz < size+r; dz++ {
				if abs(dx) >= r && abs(dz) >= r {
					continue
				}
				treeLeaf(tx, origin.Add(cube.Pos{dx, y, dz}), leaves, treeAir)
			}
		}
	}

	if rand.IntN(2) != 0 {
		return
	}
	for dx := range size {
		for dz := range size {
			treeLeaf(tx, origin.Add(cube.Pos{dx, c.layers() + 1, dz}), leaves, treeAir)
		}
	}
}

type acaciaCanopy struct {
	size int

	simplified bool
}

func (c acaciaCanopy) place(tx *world.Tx, origin cube.Pos, _ int, leaves LeavesType) {
	m := max(1, c.size)
	if c.simplified {
		for dx := 1 - m; dx < m; dx++ {
			for dz := 1 - m; dz < m; dz++ {
				treeLeaf(tx, origin.Add(cube.Pos{dx, 1, dz}), leaves, treeLeafReplaceable)
			}
		}
	} else {
		k := max(2, c.size)
		for dx := 2 - k; dx < k-1; dx++ {
			for dz := 2 - k; dz < k-1; dz++ {
				treeLeaf(tx, origin.Add(cube.Pos{dx, 1, dz}), leaves, treeLeafReplaceable)
			}
		}
		for _, arm := range [4]cube.Pos{{m - 1, 1}, {1 - m, 1}, {0, 1, m - 1}, {0, 1, 1 - m}} {
			treeLeaf(tx, origin.Add(arm), leaves, treeLeafReplaceable)
		}
	}

	for dx := -c.size; dx <= c.size; dx++ {
		for dz := -c.size; dz <= c.size; dz++ {
			if abs(dx) == c.size && abs(dz) == c.size {
				continue
			}
			treeLeaf(tx, origin.Add(cube.Pos{dx, 0, dz}), leaves, treeLeafReplaceable)
		}
	}
}

func treeRow(tx *world.Tx, pos cube.Pos, r, sizeX, sizeZ int, leaves LeavesType) {
	full := func(tx *world.Tx, p cube.Pos) bool {
		return treeReplaceable(tx, p, Leaves{Type: leaves})
	}
	for dx := -r; dx < sizeX+r; dx++ {
		for dz := -r; dz < sizeZ+r; dz++ {
			if dx*dx+dz*dz > r*r {
				if (dx-1)*(dx-1)+(dz-1)*(dz-1) > r*r && dx*dx+(dz-1)*(dz-1) > r*r && (dx-1)*(dx-1)+dz*dz > r*r {
					continue
				}
			}
			treeLeaf(tx, pos.Add(cube.Pos{dx, 0, dz}), leaves, full)
		}
	}
}

type megaCanopy struct {
	height intRange

	baseRadius int

	coreWidth int

	simplified bool
}

func (c megaCanopy) place(tx *world.Tx, origin cube.Pos, width int, leaves LeavesType) {
	size := width
	if c.simplified || size <= 0 {
		size = c.coreWidth
	}
	h := c.height.at()
	for row := h - 1; row >= 0; row-- {
		treeRow(tx, cube.Pos{origin[0], origin[1] - row, origin[2]}, c.coreWidth/2+c.baseRadius+row, size, size, leaves)
	}
}

type megaPineCanopy struct {
	height intRange

	baseRadius int

	radiusStep float64

	coreWidth int
}

func (c megaPineCanopy) place(tx *world.Tx, origin cube.Pos, width int, leaves LeavesType) {
	size := width
	if size <= 0 {
		size = c.coreWidth
	}
	h, last := c.height.at(), 0
	for down := h; down >= 0; down-- {
		y := origin[1] - down
		r := c.baseRadius + int(math.Floor(float64(down)/float64(h)*c.radiusStep))
		frill := r
		if down >= 1 && r == last && y%2 == 0 {
			frill++
		}
		treeRow(tx, cube.Pos{origin[0], y, origin[2]}, frill, size, size, leaves)
		last = r
	}
}

type cherryCanopy struct {
	height, radius intRange

	coreWidth int

	wideBottomHole treeChance

	cornerHole treeChance

	hangingLeaves, hangingExtension treeChance
}

func (c cherryCanopy) place(tx *world.Tx, origin cube.Pos, _ int, leaves LeavesType) {
	r, h := c.radius.atInclusive(), c.height.atInclusive()

	widest := c.coreWidth + r - 2
	c.layer(tx, origin, h-3, widest-2, leaves)
	c.layer(tx, origin, h-4, widest-1, leaves)
	for dy := h - 5; dy >= 0; dy-- {
		c.layer(tx, origin, dy, widest, leaves)
	}
	for _, l := range [2]struct{ dy, radius int }{{-1, widest}, {-2, widest - 1}} {
		c.layer(tx, origin, l.dy, l.radius, leaves)
		c.hang(tx, origin, l.dy, l.radius, leaves)
	}
}

func (c cherryCanopy) layer(tx *world.Tx, origin cube.Pos, dy, r int, leaves LeavesType) {
	w := c.coreWidth
	for dx := -r; dx < w+r; dx++ {
		for dz := -r; dz < w+r; dz++ {
			adx, adz := abs(dx), abs(dz)
			if w >= 2 {
				adx, adz = min(adx, abs(dx-(w-1))), min(adz, abs(dz-(w-1)))
			}

			if dy == -1 && (adx == r || adz == r) && c.wideBottomHole.roll() {
				continue
			}
			corner := adx == r && adz == r
			if (corner || (r >= 3 && adx+adz > 2*r-2)) && c.cornerHole.roll() {
				continue
			}
			treeLeaf(tx, origin.Add(cube.Pos{dx, dy, dz}), leaves, treeLeafReplaceable)
		}
	}
}

func (c cherryCanopy) hang(tx *world.Tx, origin cube.Pos, dy, r int, leaves LeavesType) {
	e := c.coreWidth + r - 1
	for _, edge := range [4]func(t int) cube.Pos{
		func(t int) cube.Pos { return cube.Pos{e, dy, t} },
		func(t int) cube.Pos { return cube.Pos{-t, dy, e} },
		func(t int) cube.Pos { return cube.Pos{-r, dy, -t} },
		func(t int) cube.Pos { return cube.Pos{t, dy, -r} },
	} {
		for t := -r; t < e; t++ {
			above := origin.Add(edge(t))
			if l, ok := tx.Block(above).(Leaves); !ok || l.Type != leaves {
				continue
			}
			if !c.hangingLeaves.roll() {
				continue
			}

			below := above.Side(cube.FaceDown)
			if !treeLeaf(tx, below, leaves, treeLeafReplaceable) {
				continue
			}
			if c.hangingExtension.roll() {
				treeLeaf(tx, below.Side(cube.FaceDown), leaves, treeLeafReplaceable)
			}
		}
	}
}

type simpleTrunk struct {
	trunkHeight    intRange
	heightModifier intRange
	canopy         treeCanopy

	vines vineDecoration
}

func (t simpleTrunk) height() int {
	return t.trunkHeight.at()
}

func (t simpleTrunk) place(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) bool {
	if !treeRoom(tx, pos, height) {
		return false
	}
	logs, top, placed := height+t.heightModifier.at(), pos, false
	for i := range logs {
		at := pos.Add(cube.Pos{0, i})
		if !treeLog(tx, at, wood, cube.Y) {

			continue
		}
		treeVines(tx, at, t.vines, [4]bool{true, true, true, true})
		top, placed = at, true
	}
	if !placed {

		return false
	}
	t.canopy.place(tx, top.Side(cube.FaceUp), 1, leaves)

	if below := pos.Side(cube.FaceDown); !treeSoil(tx.Block(below)) {
		return true
	} else if _, dirt := tx.Block(below).(Dirt); !dirt {
		tx.SetBlock(below, Dirt{}, nil)
	}
	return true
}

type acaciaTrunk struct {
	trunkHeight heightRange
	trunkWidth  int

	minHeightForCanopy int

	leanHeight, leanSteps intRange

	lean bool

	branchLength, branchPosition intRange
	branchChance                 treeChance
	branchCanopy                 treeCanopy
	canopy                       treeCanopy
}

func (t acaciaTrunk) height() int {
	return t.trunkHeight.at()
}

func (t acaciaTrunk) place(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) bool {
	if !treeRoom(tx, pos, height) {
		return false
	}

	for dx := range t.trunkWidth {
		for dz := range t.trunkWidth {
			below := pos.Add(cube.Pos{dx, -1, dz})
			if _, dirt := tx.Block(below).(Dirt); !dirt {
				tx.SetBlock(below, Dirt{}, nil)
			}
		}
	}

	dir := rand.IntN(4)
	leanHeight, leanSteps := t.leanHeight.at(), t.leanSteps.at()
	leanStart, cur := height-leanHeight, pos
	var anchor cube.Pos
	var anchored bool

	for i := range height {
		if i >= leanStart && leanSteps > 0 {
			cur, leanSteps = cur.Add(treeDirections[dir]), leanSteps-1
		}
		cur[1] = pos[1] + i
		base, y := cur, cur[1]
		if !t.lean {
			base, y = pos, pos[1]+i
		}
		for dx := range t.trunkWidth {
			for dz := range t.trunkWidth {
				at := cube.Pos{base[0] + dx, y, base[2] + dz}
				if !treeLog(tx, at, wood, cube.Y) {
					continue
				}
				if i >= t.minHeightForCanopy {
					anchor, anchored = at, true
				}
			}
		}
	}
	if !anchored {
		return true
	}
	t.canopy.place(tx, anchor, t.trunkWidth, leaves)
	t.branches(tx, pos, anchor, dir, height, leanStart, wood, leaves)
	return true
}

func (t acaciaTrunk) branches(tx *world.Tx, pos, anchor cube.Pos, dir, height, leanStart int, wood WoodType, leaves LeavesType) {
	if t.lean {

		to := rand.IntN(4)
		if to == dir || !t.branchChance.roll() {
			return
		}

		position, length := t.branchPosition.at(), t.branchLength.at()
		cur := cube.Pos{pos[0], anchor[1], pos[2]}
		for y := leanStart - position; y < height && length > 0; y, length = y+1, length-1 {
			if y < 1 {

				continue
			}
			cur = cur.Add(treeDirections[to])
			cur[1] = pos[1] + y
			treeLog(tx, cur, wood, cube.Y)
		}
		if t.branchCanopy != nil {
			t.branchCanopy.place(tx, cur, 1, leaves)
		}
		return
	}
	for dx := -1; dx <= t.trunkWidth; dx++ {
		for dz := -1; dz <= t.trunkWidth; dz++ {
			if dx >= 0 && dx < t.trunkWidth && dz >= 0 && dz < t.trunkWidth {
				continue
			}
			if !t.branchChance.roll() {
				continue
			}
			length, position := t.branchLength.at(), t.branchPosition.at()
			for k := range length {
				treeLog(tx, cube.Pos{pos[0] + dx, anchor[1] - k - position, pos[2] + dz}, wood, cube.Y)
			}
			if t.branchCanopy != nil {

				t.branchCanopy.place(tx, cube.Pos{anchor[0] + dx, anchor[1] - position, anchor[2] + dz}, 1, leaves)
			}
		}
	}
}

type megaTrunk struct {
	trunkHeight heightRange
	trunkWidth  int
	canopy      treeCanopy

	branchLength   int
	branchInterval intRange
	branchSlope    float64

	altitudeFactor [2]float64
	branchCanopy   treeCanopy

	vines vineDecoration

	base world.Block

	cluster baseCluster
}

func (t megaTrunk) height() int {
	return t.trunkHeight.at()
}

func (t megaTrunk) place(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) bool {
	if !treeSoil(tx.Block(pos.Side(cube.FaceDown))) {
		return false
	}

	for i := 0; i <= height+1; i++ {
		r := t.trunkWidth
		if i == 0 {
			r = max(0, t.trunkWidth-1)
		}
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if !treeGrowsThrough(tx, pos.Add(cube.Pos{dx, i, dz})) {
					return false
				}
			}
		}
	}

	t.canopy.place(tx, pos.Add(cube.Pos{0, height}), t.trunkWidth, leaves)
	t.branches(tx, pos, height, wood, leaves)

	for i := range height {
		for dx := range t.trunkWidth {
			for dz := range t.trunkWidth {
				at := pos.Add(cube.Pos{dx, i, dz})
				if !treeLog(tx, at, wood, cube.Y) {
					continue
				}
				treeVines(tx, at, t.vines, [4]bool{
					dx == 0, dx == t.trunkWidth-1, dz == 0, dz == t.trunkWidth-1,
				})
			}
		}
	}

	for dx := range t.trunkWidth {
		for dz := range t.trunkWidth {
			if below := pos.Add(cube.Pos{dx, -1, dz}); tx.Block(below) != t.base {
				tx.SetBlock(below, t.base, nil)
			}
		}
	}
	t.cluster.place(tx, pos, t.trunkWidth)
	return true
}

func (t megaTrunk) branches(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) {
	if t.branchLength <= 0 {
		return
	}
	y := pos[1] - t.branchInterval.at() + int(t.altitudeFactor[1]*float64(height))
	for y > pos[1]+int(t.altitudeFactor[0]*float64(height)) {
		angle := rand.Float64() * 2 * math.Pi
		bx, bz := pos[0]+1, pos[2]+1
		base := float64(y-1) - float64(t.branchLength-1)*t.branchSlope
		for b := range t.branchLength {
			bx = pos[0] + int(1.5+math.Cos(angle)*float64(b))
			bz = pos[2] + int(1.5+math.Sin(angle)*float64(b))
			treeLog(tx, cube.Pos{bx, int(base + float64(b)*t.branchSlope), bz}, wood, cube.Y)
		}
		if t.branchCanopy != nil {
			t.branchCanopy.place(tx, cube.Pos{bx, y, bz}, 1, leaves)
		}
		y -= t.branchInterval.at()
	}
}

type baseCluster struct {
	clusters, radius int
}

func (c baseCluster) place(tx *world.Tx, pos cube.Pos, width int) {
	if c.clusters <= 0 {
		return
	}
	for _, corner := range [4]cube.Pos{
		{pos[0] - 1, pos[1], pos[2] - 1}, {pos[0] + width, pos[1], pos[2] - 1},
		{pos[0] - 1, pos[1], pos[2] + width}, {pos[0] + width, pos[1], pos[2] + width},
	} {
		c.patch(tx, corner, pos[1])
	}
	for range c.clusters {

		n := rand.IntN(64)
		if q := n % 8; q == 0 || q == 7 || n < 8 || n >= 56 {
			c.patch(tx, cube.Pos{pos[0] - 3 + q, pos[1], pos[2] - 3 + n/8}, pos[1])
		}
	}
}

func (c baseCluster) patch(tx *world.Tx, at cube.Pos, y int) {
	for dx := -c.radius; dx <= c.radius; dx++ {
		for dz := -c.radius; dz <= c.radius; dz++ {
			if abs(dx) == c.radius && abs(dz) == c.radius {
				continue
			}

			for dy := 2; dy >= -3; dy-- {
				pos := cube.Pos{at[0] + dx, y - 1 + dy, at[2] + dz}
				switch tx.Block(pos).(type) {
				case Podzol:
					dy = -4
				case Grass, Dirt:
					tx.SetBlock(pos, Podzol{}, nil)
					dy = -4
				}
			}
		}
	}
}

type cherryTrunk struct {
	trunkHeight heightRange

	branchLength intRange

	branchOffset intRange

	branchRise intRange
	canopy     treeCanopy
}

func (t cherryTrunk) height() int {
	return t.trunkHeight.at()
}

func (t cherryTrunk) place(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) bool {
	if !treeRoom(tx, pos, height) {
		return false
	}
	shape := rand.IntN(3)
	start := max(0, t.branchOffset.atInclusive()+height-1)
	end := max(0, t.branchOffset.at()+height-1)
	if start <= end {
		end++
	}
	dir := rand.IntN(4)

	logs := height
	switch shape {
	case 0:
		logs = start + 1
	case 1:
		logs = max(start, end) + 1
	}
	below := pos.Side(cube.FaceDown)
	if _, dirt := tx.Block(below).(Dirt); !dirt && treeSoil(tx.Block(below)) {
		tx.SetBlock(below, Dirt{}, nil)
	}
	for i := range logs {

		treeLog(tx, pos.Add(cube.Pos{0, i}), wood, cube.Y)
	}

	crowns := make([]cube.Pos, 0, 3)
	if shape == 2 {
		crowns = append(crowns, pos.Add(cube.Pos{0, height}))
	}
	crowns = append(crowns, t.branch(tx, pos, treeDirections[dir], height, start, logs, wood))
	if shape >= 1 {
		to := cube.Pos{-treeDirections[dir][0], 0, -treeDirections[dir][2]}
		crowns = append(crowns, t.branch(tx, pos, to, height, end, logs, wood))
	}
	for _, crown := range crowns {
		t.canopy.place(tx, crown, 1, leaves)
	}
	return true
}

func (t cherryTrunk) branch(tx *world.Tx, pos, dir cube.Pos, height, at, logs int, wood WoodType) cube.Pos {
	rise := height - 1 + t.branchRise.atInclusive()

	long := at < logs-1 || rise < at
	length := t.branchLength.atInclusive()
	if long {
		length++
	}

	axis := cube.X
	if dir[2] != 0 {
		axis = cube.Z
	}
	cur, steps := cube.Pos{pos[0], pos[1] + at, pos[2]}, 1
	if long {
		steps = 2
	}
	for range steps {
		cur = cur.Add(dir)
		treeLog(tx, cur, wood, axis)
	}

	target := cube.Pos{pos[0] + dir[0]*length, pos[1] + rise, pos[2] + dir[2]*length}
	for cur != target {
		dx, dy, dz := target[0]-cur[0], target[1]-cur[1], target[2]-cur[2]
		if rand.Float64() < float64(abs(dy))/float64(abs(dx)+abs(dy)+abs(dz)) {
			cur[1] += sign(dy)
			treeLog(tx, cur, wood, cube.Y)
			continue
		}
		if abs(dx) >= abs(dz) {
			cur[0] += sign(dx)
		} else {
			cur[2] += sign(dz)
		}
		treeLog(tx, cur, wood, axis)
	}
	return target.Side(cube.FaceUp)
}

var (
	blobVariation = []treeChance{50, 50, 50, 100}

	oakTree = simpleTrunk{
		trunkHeight: intRange{4, 7},
		canopy: simpleCanopy{
			offset: intRange{-3, 0}, minWidth: 1, slopeRun: 1, slopeRise: 0.5, variation: blobVariation,
		},
	}

	birchTree = simpleTrunk{
		trunkHeight: intRange{5, 8},
		canopy: simpleCanopy{
			offset: intRange{-3, 0}, minWidth: 1, slopeRun: 1, slopeRise: 0.5, variation: blobVariation,
		},
	}

	jungleTree = simpleTrunk{
		trunkHeight: intRange{4, 11},
		vines:       vineDecoration{chance: 33.33, steps: 1},
		canopy: simpleCanopy{
			offset: intRange{-3, 0}, minWidth: 1, slopeRun: 1, slopeRise: 0.5, variation: blobVariation,
			vines: vineDecoration{chance: 25, steps: 4},
		},
	}

	spruceTree = simpleTrunk{
		trunkHeight:    intRange{6, 10},
		heightModifier: intRange{-2, 1},
		canopy: spruceCanopy{
			lowerOffset: intRange{1, 3}, upperOffset: intRange{0, 3}, maxRadius: intRange{2, 4},
		},
	}

	savannaTree = acaciaTrunk{
		trunkHeight:        heightRange{base: 5, intervals: []int{3, 3}},
		trunkWidth:         1,
		minHeightForCanopy: 3,
		leanHeight:         intRange{1, 5},
		leanSteps:          intRange{1, 4},
		lean:               true,
		branchLength:       intRange{1, 4},
		branchPosition:     intRange{1, 3},
		branchChance:       100,
		branchCanopy:       acaciaCanopy{size: 2, simplified: true},
		canopy:             acaciaCanopy{size: 3},
	}

	roofedTree = acaciaTrunk{
		trunkHeight:        heightRange{base: 6, intervals: []int{3, 2}},
		trunkWidth:         2,
		minHeightForCanopy: 3,
		leanHeight:         intRange{0, 4},
		leanSteps:          intRange{0, 3},
		branchLength:       intRange{2, 5},
		branchPosition:     intRange{1, 1},
		branchChance:       100.0 / 3.0,
		branchCanopy:       acaciaCanopy{size: 2, simplified: true},
		canopy:             roofedCanopy{height: 4, coreWidth: 2, outerRadius: 2, innerRadius: 3},
	}

	megaSpruceTree = megaTrunk{
		trunkHeight: heightRange{base: 13, intervals: []int{3, 15}},
		trunkWidth:  2,
		canopy: megaPineCanopy{
			height: intRange{13, 18}, baseRadius: 0, radiusStep: 3.5, coreWidth: 2,
		},
		base:    Podzol{},
		cluster: baseCluster{clusters: 5, radius: 2},
	}

	megaJungleTree = megaTrunk{
		trunkHeight:    heightRange{base: 10, intervals: []int{3, 20}},
		trunkWidth:     2,
		branchLength:   5,
		branchInterval: intRange{2, 6},
		branchSlope:    0.5,
		altitudeFactor: [2]float64{0.5, 1},
		base:           Dirt{},
		vines:          vineDecoration{chance: 100.0 / 3.0, steps: 1},
		canopy:         megaCanopy{height: intRange{3, 3}, baseRadius: 2, coreWidth: 2},
		branchCanopy:   megaCanopy{height: intRange{2, 4}, baseRadius: 1, coreWidth: 1, simplified: true},
	}

	poplarTree = poplarTrunk{
		trunkHeight: intRange{7, 11},
		remaining:   intRange{4, 4},
		branches:    intRange{1, 4},

		mushroomChance: 25,
		canopy: poplarCanopy{
			radius: weightedInts{
				{value: 5, weight: 5}, {value: 6, weight: 5}, {value: 7, weight: 1}, {value: 8, weight: 1},
			},
			height:         intRange{5, 6},
			sideHoleChance: 15,
		},
	}

	cherryTree = cherryTrunk{
		trunkHeight:  heightRange{base: 7, intervals: []int{1}, inclusive: true},
		branchLength: intRange{2, 4},
		branchOffset: intRange{-4, -3},
		branchRise:   intRange{-1, 0},
		canopy: cherryCanopy{
			height: intRange{5, 5}, radius: intRange{4, 4}, coreWidth: 1,
			wideBottomHole: 25, cornerHole: 50, hangingLeaves: 16.6666, hangingExtension: 33.3333,
		},
	}
)

func poplarFullerQuadrant(x, z int, sameSign bool) bool {
	if sameSign {
		return x < 0 && z < 0 || x > 0 && z > 0
	}
	return x < 0 && z > 0 || x > 0 && z < 0
}

type poplarCanopy struct {
	radius weightedInts

	height intRange

	sideHoleChance treeChance

	wood WoodType
}

func (c poplarCanopy) leafLayer(tx *world.Tx, origin cube.Pos, leaves LeavesType, height, radius, y int, sameSign bool) {

	topTwo := y == height-1 || y == height-2
	for x := -radius; x <= radius; x++ {
		for z := -radius; z <= radius; z++ {
			if topTwo && (abs(x) == radius || abs(z) == radius) {
				continue
			}
			bonus := 0
			if topTwo {
				bonus = -1
			}
			if poplarFullerQuadrant(x, z, sameSign) {
				bonus = 1
			}

			edge := 0
			if !c.sideHoleChance.roll() {
				edge = 1
			}
			if abs(x)+abs(z) > radius+bonus-1+edge {
				continue
			}
			treeLeaf(tx, origin.Add(cube.Pos{x, y, z}), leaves, poplarLeafReplaceable)
		}
	}
}

func (c poplarCanopy) place(tx *world.Tx, origin cube.Pos, _ int, leaves LeavesType) {
	r, h := c.radius.at()-1, c.height.atInclusive()
	sameSign := rand.IntN(2) == 0

	c.leafLayer(tx, origin, leaves, h, r-2, h-1, sameSign)
	c.leafLayer(tx, origin, leaves, h, r-1, h-2, sameSign)
	c.leafLayer(tx, origin, leaves, h, r-1, h-3, sameSign)
	for y := h - 4; y >= 1; y-- {
		c.leafLayer(tx, origin, leaves, h, r, y, sameSign)
	}

	for x := -r; x <= r; x++ {
		for z := -r; z <= r; z++ {
			onAxis := x == 0 && r-abs(z) > 3 || z == 0 && r-abs(x) > 3
			if !onAxis {
				continue
			}
			bonus := 0
			if poplarFullerQuadrant(x, z, sameSign) {
				bonus = 1
			}
			if abs(x)+abs(z) > r-2+bonus {
				continue
			}
			axis := cube.Z
			if z == 0 {
				axis = cube.X
			}
			treeLog(tx, origin.Add(cube.Pos{x, h - 4, z}), c.wood, axis)
		}
	}

	c.leafLayer(tx, origin, leaves, h, r-1, 0, sameSign)
	bottom := 2
	if r < 3 {
		bottom = 1
	} else if r < 5 {
		bottom = r - 2
	}
	c.leafLayer(tx, origin, leaves, h, bottom, -1, sameSign)
}

type poplarTrunk struct {
	trunkHeight intRange

	remaining intRange

	branches intRange

	mushroomChance treeChance
	canopy         poplarCanopy
}

func treeShelfMushroom(tx *world.Tx, pos cube.Pos, chance treeChance) {
	if !chance.roll() {
		return
	}
	dirs := [4]cube.Direction{cube.North, cube.East, cube.South, cube.West}
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
	for _, d := range dirs {
		m := ShelfMushroom{Facing: d}
		at := pos.Side(d.Face())
		if !treeReplaceable(tx, at, m) {
			continue
		}

		adjacent := false
		at.Neighbours(func(n cube.Pos) {
			if _, ok := tx.Block(n).(ShelfMushroom); ok {
				adjacent = true
			}
		}, tx.Range())
		if adjacent {
			continue
		}
		tx.SetBlock(at, m, nil)
		return
	}
}

func (t poplarTrunk) height() int {
	return t.trunkHeight.atInclusive()
}

func (t poplarTrunk) place(tx *world.Tx, pos cube.Pos, height int, wood WoodType, leaves LeavesType) bool {
	if !treeRoom(tx, pos, height) {
		return false
	}

	remaining := t.remaining.atInclusive()
	for y := range height {

		at := pos.Add(cube.Pos{0, y})
		if treeLog(tx, at, wood, cube.Y) {
			treeShelfMushroom(tx, at, t.mushroomChance)
		}
	}

	dirs := treeDirections
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
	offset := height - remaining
	for _, d := range dirs[:min(t.branches.atInclusive(), len(dirs))] {
		axis := cube.X
		if d[2] != 0 {
			axis = cube.Z
		}
		treeLog(tx, cube.Pos{pos[0] + d[0], pos[1] + offset - 1, pos[2] + d[2]}, wood, axis)
	}

	c := t.canopy
	c.wood = wood
	c.place(tx, pos.Add(cube.Pos{0, offset}), 1, leaves)

	if below := pos.Side(cube.FaceDown); !treeSoil(tx.Block(below)) {
		return true
	} else if _, dirt := tx.Block(below).(Dirt); !dirt {
		tx.SetBlock(below, Dirt{}, nil)
	}
	return true
}

func growTree(t SaplingType, pos cube.Pos, tx *world.Tx) bool {
	wood, leaves := t.Wood(), t.Leaves()
	if t == PoplarSapling() {

		colours := PoplarLeavesTypes()
		leaves = colours[rand.IntN(len(colours))]
	}

	var trunk treeTrunk
	switch t {
	case DarkOakSapling(), PaleOakSapling():
		corner, ok := saplingSquare(pos, tx, t)
		if !ok {
			return false
		}
		pos, trunk = corner, roofedTree
	case SpruceSapling():
		if corner, ok := saplingSquare(pos, tx, t); ok {
			pos, trunk = corner, megaSpruceTree
			break
		}
		trunk = spruceTree
	case JungleSapling():
		if corner, ok := saplingSquare(pos, tx, t); ok {
			pos, trunk = corner, megaJungleTree
			break
		}
		trunk = jungleTree
	case OakSapling():
		trunk = oakTree
	case BirchSapling():
		trunk = birchTree
	case AcaciaSapling():
		trunk = savannaTree
	case CherrySapling():
		trunk = cherryTree
	case PoplarSapling():
		trunk = poplarTree
	default:
		return false
	}
	return trunk.place(tx, pos, trunk.height(), wood, leaves)
}

func saplingSquare(pos cube.Pos, tx *world.Tx, t SaplingType) (cube.Pos, bool) {
	for _, corner := range []cube.Pos{{}, {-1}, {0, 0, -1}, {-1, 0, -1}} {
		corner = pos.Add(corner)
		if saplingAt(corner, tx, t) && saplingAt(corner.Add(cube.Pos{1}), tx, t) &&
			saplingAt(corner.Add(cube.Pos{0, 0, 1}), tx, t) && saplingAt(corner.Add(cube.Pos{1, 0, 1}), tx, t) {
			return corner, true
		}
	}
	return cube.Pos{}, false
}

func saplingAt(pos cube.Pos, tx *world.Tx, t SaplingType) bool {
	s, ok := tx.Block(pos).(Sapling)
	return ok && s.Type == t
}
