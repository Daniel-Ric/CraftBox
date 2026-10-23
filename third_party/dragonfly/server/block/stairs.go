package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

type Stairs struct {
	transparent
	sourceWaterDisplacer

	Block world.Block

	UpsideDown bool

	Facing cube.Direction

	Corner StairsCorner
}

func (s Stairs) UseOnBlock(pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, face, used = firstReplaceable(tx, pos, face, s)
	if !used {
		return
	}
	s.Facing = user.Rotation().Direction()
	if face == cube.FaceDown || (clickPos[1] > 0.5 && face != cube.FaceUp) {
		s.UpsideDown = true
	}

	s.Corner = s.calculateCorner(tx, pos)
	place(tx, pos, s, user, ctx)
	return placed(ctx)
}

func (s Stairs) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if corner := s.calculateCorner(tx, pos); corner != s.Corner {
		s.Corner = corner
		tx.SetBlock(pos, s, nil)
	}
}

func (s Stairs) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	s.Corner = s.calculateCorner(src, pos)
	return s
}

func (s Stairs) calculateCorner(src world.BlockSource, pos cube.Pos) StairsCorner {
	rotated := s.Facing.RotateRight()
	if closed, ok := src.Block(pos.Side(s.Facing.Face())).(Stairs); ok && closed.UpsideDown == s.UpsideDown {
		if closed.Facing == rotated {
			return OuterRightStairsCorner()
		} else if closed.Facing == rotated.Opposite() {
			if s.continued(src, pos) {
				return NoStairsCorner()
			}
			return OuterLeftStairsCorner()
		}
	}
	if open, ok := src.Block(pos.Side(s.Facing.Opposite().Face())).(Stairs); ok && open.UpsideDown == s.UpsideDown {
		if open.Facing == rotated && !s.continued(src, pos) {
			return InnerRightStairsCorner()
		} else if open.Facing == rotated.Opposite() {
			return InnerLeftStairsCorner()
		}
	}
	return NoStairsCorner()
}

func (s Stairs) continued(src world.BlockSource, pos cube.Pos) bool {
	side, ok := src.Block(pos.Side(s.Facing.RotateRight().Face())).(Stairs)
	return ok && side.Facing == s.Facing && side.UpsideDown == s.UpsideDown
}

func (s Stairs) Model() world.BlockModel {
	return model.Stair{
		Facing:     s.Facing,
		UpsideDown: s.UpsideDown,
		Corner:     s.Corner != NoStairsCorner(),
		Inner:      s.Corner.Inner(),
		Left:       s.Corner.Left(),
	}
}

func (s Stairs) BreakInfo() BreakInfo {
	breakInfo := s.Block.(Breakable).BreakInfo()
	return newBreakInfo(breakInfo.Hardness, breakInfo.Harvestable, breakInfo.Effective, oneOf(s)).withBlastResistance(breakInfo.BlastResistance)
}

func (s Stairs) Instrument() sound.Instrument {
	if _, ok := s.Block.(Planks); ok {
		return sound.Bass()
	}
	if _, ok := s.Block.(BambooMosaic); ok {
		return sound.Bass()
	}
	if _, ok := s.Block.(Wool); ok {
		return sound.Guitar()
	}
	return sound.BassDrum()
}

func (s Stairs) FlammabilityInfo() FlammabilityInfo {
	if flammable, ok := s.Block.(Flammable); ok {
		return flammable.FlammabilityInfo()
	}
	return newFlammabilityInfo(0, 0, false)
}

func (s Stairs) FuelInfo() item.FuelInfo {
	if fuel, ok := s.Block.(item.Fuel); ok {
		return fuel.FuelInfo()
	}
	return item.FuelInfo{}
}

func (s Stairs) EncodeItem() (name string, meta int16) {
	return "minecraft:" + encodeStairsBlock(s.Block) + "_stairs", 0
}

func (s Stairs) EncodeBlock() (name string, properties map[string]any) {
	return "minecraft:" + encodeStairsBlock(s.Block) + "_stairs", map[string]any{"minecraft:corner": s.Corner.String(), "upside_down_bit": s.UpsideDown, "weirdo_direction": toStairsDirection(s.Facing)}
}

func toStairsDirection(v cube.Direction) int32 {
	return int32(3 - v)
}

func (s Stairs) SideClosed(pos, side cube.Pos, tx *world.Tx) bool {
	return s.Model().FaceSolid(pos, pos.Face(side), tx)
}

func (Stairs) CanRedstoneWireStepDown(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func allStairs() (stairs []world.Block) {
	f := func(facing cube.Direction, upsideDown bool) {
		for _, s := range StairsBlocks() {
			for _, corner := range StairsCorners() {
				stairs = append(stairs, Stairs{Facing: facing, UpsideDown: upsideDown, Block: s, Corner: corner})
			}
		}
	}
	for i := cube.Direction(0); i <= 3; i++ {
		f(i, true)
		f(i, false)
	}
	return
}
