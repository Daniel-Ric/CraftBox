package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"math/rand/v2"
	"time"
)

type CoralBlock struct {
	solid
	bassDrum

	Type CoralType

	Dead bool
}

func (c CoralBlock) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if c.Dead {
		return
	}
	tx.ScheduleBlockUpdate(pos, c, time.Second*5/2)
}

func (c CoralBlock) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	adjacentWater := false
	pos.Neighbours(func(neighbour cube.Pos) {
		if liquid, ok := tx.Liquid(neighbour); ok {
			if _, ok := liquid.(Water); ok {
				adjacentWater = true
			}
		}
	}, tx.Range())
	if !adjacentWater {
		c.Dead = true
		tx.SetBlock(pos, c, nil)
	}
}

func (c CoralBlock) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, pickaxeHarvestable, pickaxeEffective, silkTouchOneOf(CoralBlock{Type: c.Type, Dead: true}, c)).withBlastResistance(6)
}

func (c CoralBlock) EncodeBlock() (name string, properties map[string]any) {
	if c.Dead {
		return "minecraft:dead_" + c.Type.String() + "_coral_block", nil
	}
	return "minecraft:" + c.Type.String() + "_coral_block", nil
}

func (c CoralBlock) EncodeItem() (name string, meta int16) {
	if c.Dead {
		return "minecraft:dead_" + c.Type.String() + "_coral_block", 0
	}
	return "minecraft:" + c.Type.String() + "_coral_block", 0
}

func allCoralBlocks() (c []world.Block) {
	f := func(dead bool) {
		for _, t := range CoralTypes() {
			c = append(c, CoralBlock{Type: t, Dead: dead})
		}
	}
	f(true)
	f(false)
	return
}
