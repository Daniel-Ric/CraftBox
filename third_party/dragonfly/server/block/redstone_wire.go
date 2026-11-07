package block

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type RedstoneWire struct {
	empty
	transparent

	Power int

	Connections WireConnections
}

func (r RedstoneWire) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(tx, pos, face, r)
	if !used || !redstoneWireSupported(tx, pos) {
		return false
	}
	r.Connections = redstoneWireConnections(pos, loadedSource{tx: tx})
	place(tx, pos, r, user, ctx)
	if !placed(ctx) {
		return false
	}

	updateDiagonalRedstoneWires(pos, tx)
	return true
}

func (r RedstoneWire) RedstonePower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	if face == cube.FaceUp {
		return 0
	}
	if tx != nil && redstoneWireFaceHorizontal(face) && !redstoneWirePoweredHorizontalFaces(pos, loadedSource{tx: tx})[face] {
		return 0
	}
	return r.Power
}

func (RedstoneWire) RedstoneWeaklyPowersBlocks() bool {
	return true
}

func (RedstoneWire) RedstoneSignalLoss(cube.Pos, *world.Tx) int {
	return 1
}

func (RedstoneWire) RedstoneRelayerNeighbours(pos cube.Pos, tx *world.Tx) []cube.Pos {
	neighbours := make([]cube.Pos, 0, 12)
	src := loadedSource{tx: tx}
	faces := redstoneWirePoweredHorizontalFaces(pos, src)
	for _, face := range cube.HorizontalFaces() {
		if !faces[face] {
			continue
		}
		side := pos.Side(face)
		if side.OutOfBounds(tx.Range()) {
			continue
		}
		if links := redstoneWireLinksThrough(pos, src, face); links.connected() {
			neighbours = append(neighbours, links.positions(side)...)
			continue
		}
		if redstoneWireRelevantLoaded(tx, side) {
			neighbours = append(neighbours, side)
		}
	}
	return neighbours
}

func (r RedstoneWire) RedstonePowerUpdate(_ cube.Pos, _ *world.Tx, power int) (world.Block, bool) {
	power = world.ClampRedstonePower(power)
	if r.Power == power {
		return r, false
	}
	r.Power = power
	return r, true
}

func (r RedstoneWire) NeighbourUpdateTick(pos, changedNeighbour cube.Pos, tx *world.Tx) {
	if !redstoneWireSupported(tx, pos) {
		breakBlock(r, pos, tx)
		return
	}
	if r.updateConnections(pos, tx) && changedNeighbour == pos {

		updateDiagonalRedstoneWires(pos, tx)
	}
}

func (r RedstoneWire) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	r.Connections = redstoneWireConnections(pos, src)
	return r
}

func (r RedstoneWire) updateConnections(pos cube.Pos, tx *world.Tx) bool {
	connections := redstoneWireConnections(pos, loadedSource{tx: tx})
	if connections == r.Connections {
		return false
	}
	r.Connections = connections
	tx.SetBlock(pos, r, &world.SetOpts{DisableBlockUpdates: true, DisableRedstoneUpdates: true, KeepRedstonePower: true})
	return true
}

func (RedstoneWire) HasLiquidDrops() bool {
	return true
}

func (r RedstoneWire) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(RedstoneWire{})).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		updateDiagonalRedstoneWires(pos, tx)
	})
}

func (RedstoneWire) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (r RedstoneWire) EncodeBlock() (string, map[string]any) {
	properties := r.Connections.properties()
	properties["redstone_signal"] = int32(world.ClampRedstonePower(r.Power))
	return "minecraft:redstone_wire", properties
}

func (RedstoneWire) EncodeItem() (name string, meta int16) {
	return "minecraft:redstone", 0
}

func (RedstoneWire) TrimMaterial() string {
	return item.RedstoneWire{}.TrimMaterial()
}

func (RedstoneWire) MaterialColour() string {
	return item.RedstoneWire{}.MaterialColour()
}

func allRedstoneWires() (all []world.Block) {
	for i := range 16 {
		for _, c := range allWireConnections() {
			all = append(all, RedstoneWire{Power: i, Connections: c})
		}
	}
	return
}

func redstoneTicks(ticks int) time.Duration {
	return time.Duration(max(ticks, 1)) * time.Second / 10
}

func redstoneWireSupported(tx *world.Tx, pos cube.Pos) bool {
	below := pos.Side(cube.FaceDown)
	if below.OutOfBounds(tx.Range()) {
		return false
	}
	return tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx)
}

type loadedSource struct{ tx *world.Tx }

func (s loadedSource) Block(pos cube.Pos) world.Block {
	b, _ := s.tx.BlockLoaded(pos)
	return b
}

func updateDiagonalRedstoneWires(pos cube.Pos, tx *world.Tx) {
	src := loadedSource{tx: tx}
	for _, face := range cube.HorizontalFaces() {
		side := pos.Side(face)
		for _, diagonal := range [2]cube.Pos{side.Side(cube.FaceUp), side.Side(cube.FaceDown)} {
			if wire, ok := src.Block(diagonal).(RedstoneWire); ok {
				wire.updateConnections(diagonal, tx)
			}
		}
	}
}

func redstoneWireConnections(pos cube.Pos, src world.BlockSource) WireConnections {
	links := redstoneWireLinksAround(pos, src)
	faces := redstoneWireShape(links, redstoneWireLinks.drawn)
	connection := func(i int) WireConnection {
		if !faces[i] {
			return NoWireConnection()
		}

		face := cube.HorizontalFaces()[i]
		if side := pos.Side(face); links[i].drawnUp && src.Block(side).Model().FaceSolid(side, face.Opposite(), src) {
			return UpWireConnection()
		}
		return SideWireConnection()
	}
	return WireConnections{north: connection(0), east: connection(1), south: connection(2), west: connection(3)}
}

func redstoneWirePoweredHorizontalFaces(pos cube.Pos, src world.BlockSource) map[cube.Face]bool {
	faces := redstoneWireShape(redstoneWireLinksAround(pos, src), redstoneWireLinks.connected)
	connections := make(map[cube.Face]bool, len(faces))
	for i, face := range cube.HorizontalFaces() {
		if faces[i] {
			connections[face] = true
		}
	}
	return connections
}

func redstoneWireLinksAround(pos cube.Pos, src world.BlockSource) (links [4]redstoneWireLinks) {
	for i, face := range cube.HorizontalFaces() {
		links[i] = redstoneWireLinksThrough(pos, src, face)
	}
	return links
}

func redstoneWireShape(links [4]redstoneWireLinks, linked func(redstoneWireLinks) bool) (faces [4]bool) {
	connected := 0
	for i, l := range links {
		if linked(l) {
			faces[i] = true
			connected++
		}
	}
	switch connected {
	case 0:
		faces = [4]bool{true, true, true, true}
	case 1:

		for i := range faces {
			if faces[i] {
				faces[(i+2)%4] = true
				break
			}
		}
	}
	return faces
}

type redstoneWireLinks struct {
	side, up, down, drawnUp, drawnDown bool
}

func (l redstoneWireLinks) connected() bool {
	return l.side || l.up || l.down
}

func (l redstoneWireLinks) drawn() bool {
	return l.side || l.drawnUp || l.drawnDown
}

func (l redstoneWireLinks) positions(side cube.Pos) []cube.Pos {
	positions := make([]cube.Pos, 0, 3)
	if l.side {
		positions = append(positions, side)
	}
	if l.up {
		positions = append(positions, side.Side(cube.FaceUp))
	}
	if l.down {
		positions = append(positions, side.Side(cube.FaceDown))
	}
	return positions
}

func redstoneWireLinksThrough(pos cube.Pos, src world.BlockSource, face cube.Face) redstoneWireLinks {
	side, above := pos.Side(face), pos.Side(cube.FaceUp)
	sideAbove, sideBelow := side.Side(cube.FaceUp), side.Side(cube.FaceDown)
	upper := redstoneWireAt(src, sideAbove) && redstoneWireSupportedBy(src, sideAbove)
	lower := redstoneWireAt(src, sideBelow)
	return redstoneWireLinks{
		side:      redstoneWireDirectConnection(src, side, face.Opposite()),
		up:        upper && !redstoneWireBlocksConnection(src, above, cube.FaceDown),
		down:      lower && !redstoneWireBlocksConnection(src, side, cube.FaceUp) && redstoneWireCanTransmitDown(src, pos),
		drawnUp:   upper && !redstoneWireBlocksDrawnConnection(src, above, cube.FaceDown),
		drawnDown: lower && !redstoneWireBlocksDrawnConnection(src, side, cube.FaceUp),
	}
}

func redstoneWireSupportedBy(src world.BlockSource, pos cube.Pos) bool {
	below := pos.Side(cube.FaceDown)
	return src.Block(below).Model().FaceSolid(below, cube.FaceUp, src)
}

func redstoneWireBlocksConnection(src world.BlockSource, pos cube.Pos, face cube.Face) bool {
	b := src.Block(pos)
	return b.Model().FaceSolid(pos, face, src) && world.RedstoneFullPowerConductor(pos, b, src)
}

func redstoneWireBlocksDrawnConnection(src world.BlockSource, pos cube.Pos, face cube.Face) bool {
	switch src.Block(pos).(type) {
	case Barrier, Glowstone, Slime:
		return true
	case Barrel, Farmland, Ice, Jukebox, TintedGlass:
		return false
	}
	return redstoneWireBlocksConnection(src, pos, face)
}

func redstoneWireCanTransmitDown(src world.BlockSource, pos cube.Pos) bool {
	supportPos := pos.Side(cube.FaceDown)
	support := src.Block(supportPos)
	if !support.Model().FaceSolid(supportPos, cube.FaceUp, src) {
		return false
	}
	if stepDowner, ok := support.(RedstoneWireStepDowner); ok {
		return stepDowner.CanRedstoneWireStepDown(supportPos, pos, src)
	}
	for _, face := range cube.Faces() {
		if !support.Model().FaceSolid(supportPos, face, src) {
			return false
		}
	}
	return true
}

func redstoneWireDirectConnection(src world.BlockSource, pos cube.Pos, face cube.Face) bool {
	b := src.Block(pos)
	switch b.(type) {
	case RedstoneWire, world.RedstonePowerSource, world.RedstoneStrongPowerSource, world.RedstonePowerRelayer:
		return true
	case Jukebox, Lectern:

		return true
	}
	return redstoneWireNonSolidComponent(pos, b, src, face)
}

func redstoneWireNonSolidComponent(pos cube.Pos, b world.Block, src world.BlockSource, face cube.Face) bool {
	model := b.Model()
	if model == nil || model.FaceSolid(pos, face, src) {
		return false
	}
	switch b.(type) {
	case world.RedstonePowerConsumer, world.RedstonePowerAction, world.RedstonePowerContextAction:
		return true
	}
	return false
}

func redstoneWireAt(src world.BlockSource, pos cube.Pos) bool {
	_, ok := src.Block(pos).(RedstoneWire)
	return ok
}

func redstoneWireRelevantLoaded(tx *world.Tx, pos cube.Pos) bool {
	b, ok := tx.BlockLoaded(pos)
	return ok && redstoneWireRelevant(b)
}

func redstoneWireRelevant(b world.Block) bool {
	switch b.(type) {
	case world.RedstonePowerSource,
		world.RedstoneStrongPowerSource,
		world.RedstonePowerRelayer,
		world.RedstonePowerConsumer,
		world.RedstonePowerAction,
		world.RedstonePowerContextAction:
		return true
	}
	return false
}

func redstoneWireFaceHorizontal(face cube.Face) bool {
	switch face {
	case cube.FaceNorth, cube.FaceSouth, cube.FaceWest, cube.FaceEast:
		return true
	default:
		return false
	}
}
