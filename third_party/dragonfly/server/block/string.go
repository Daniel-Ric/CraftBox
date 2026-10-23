package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type String struct {
	empty
	transparent

	Attached bool

	Disarmed bool

	Powered bool

	Suspended bool

	Connections Connections
}

func (s String) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, _, canPlace := firstReplaceable(tx, pos, face, s)
	if !canPlace {
		return false
	}
	below := pos.Side(cube.FaceDown)
	s.Suspended = !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx)
	s.Connections = calculateConnections(s, tx, pos)
	place(tx, pos, s, user, ctx)
	return placed(ctx)
}

func (s String) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	below := pos.Side(cube.FaceDown)
	suspended, connections := !tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx), calculateConnections(s, tx, pos)
	if suspended == s.Suspended && connections == s.Connections {
		return
	}
	s.Suspended, s.Connections = suspended, connections
	tx.SetBlock(pos, s, nil)
}

func (s String) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	s.Connections = calculateConnections(s, src, pos)
	return s
}

func (String) Connects(pos cube.Pos, face cube.Face, src world.BlockSource) bool {
	_, ok := src.Block(pos.Side(face)).(String)
	return ok
}

func (s String) HasLiquidDrops() bool {
	return true
}

func (s String) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(String{}))
}

func (s String) EncodeItem() (name string, meta int16) {
	return "minecraft:string", 0
}

func (s String) EncodeBlock() (name string, properties map[string]any) {
	properties = s.Connections.properties()
	properties["attached_bit"] = boolByte(s.Attached)
	properties["disarmed_bit"] = boolByte(s.Disarmed)
	properties["powered_bit"] = boolByte(s.Powered)
	properties["suspended_bit"] = boolByte(s.Suspended)
	return "minecraft:trip_wire", properties
}

func allString() (blocks []world.Block) {
	for meta := 0; meta < 16; meta++ {
		for _, c := range allConnections() {
			blocks = append(blocks, String{
				Powered:     meta&0x1 != 0,
				Suspended:   meta&0x2 != 0,
				Attached:    meta&0x4 != 0,
				Disarmed:    meta&0x8 != 0,
				Connections: c,
			})
		}
	}
	return blocks
}
