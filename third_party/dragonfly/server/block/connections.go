package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type Connections struct {
	north, east, south, west bool
}

func (c Connections) Uint8() uint8 {
	return boolByte(c.north) | boolByte(c.east)<<1 | boolByte(c.south)<<2 | boolByte(c.west)<<3
}

func (c Connections) properties() map[string]any {
	return map[string]any{
		"minecraft:connection_north": c.north,
		"minecraft:connection_east":  c.east,
		"minecraft:connection_south": c.south,
		"minecraft:connection_west":  c.west,
	}
}

type connector interface {
	Connects(pos cube.Pos, face cube.Face, src world.BlockSource) bool
}

func calculateConnections(c connector, src world.BlockSource, pos cube.Pos) Connections {
	return Connections{
		north: c.Connects(pos, cube.FaceNorth, src),
		east:  c.Connects(pos, cube.FaceEast, src),
		south: c.Connects(pos, cube.FaceSouth, src),
		west:  c.Connects(pos, cube.FaceWest, src),
	}
}

func allConnections() (all []Connections) {
	for i := range 16 {
		all = append(all, Connections{north: i&0x1 != 0, east: i&0x2 != 0, south: i&0x4 != 0, west: i&0x8 != 0})
	}
	return
}
