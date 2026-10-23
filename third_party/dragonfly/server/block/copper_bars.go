package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

type CopperBars struct {
	transparent
	thin
	sourceWaterDisplacer

	Oxidation OxidationType

	Waxed bool

	Connections Connections
}

func (c CopperBars) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if connections := calculateConnections(c.Model().(connector), tx, pos); connections != c.Connections {
		c.Connections = connections
		tx.SetBlock(pos, c, nil)
	}
}

func (c CopperBars) DeriveState(pos cube.Pos, src world.BlockSource) world.Block {
	c.Connections = calculateConnections(c.Model().(connector), src, pos)
	return c
}

func (c CopperBars) BreakInfo() BreakInfo {
	return newBreakInfo(5, pickaxeHarvestable, pickaxeEffective, oneOf(c)).withBlastResistance(6)
}

func (c CopperBars) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (c CopperBars) Wax(cube.Pos, mgl64.Vec3) (world.Block, bool) {
	if c.Waxed {
		return c, false
	}
	c.Waxed = true
	return c, true
}

func (c CopperBars) Strip() (world.Block, world.Sound, bool) {
	if c.Waxed {
		c.Waxed = false
		return c, sound.WaxRemoved{}, true
	} else if ot, ok := c.Oxidation.Decrease(); ok {
		c.Oxidation = ot
		return c, sound.CopperScraped{}, true
	}
	return c, nil, false
}

func (c CopperBars) CanOxidate() bool {
	return !c.Waxed
}

func (c CopperBars) OxidationLevel() OxidationType {
	return c.Oxidation
}

func (c CopperBars) WithOxidationLevel(o OxidationType) Oxidisable {
	c.Oxidation = o
	return c
}

func (c CopperBars) RandomTick(pos cube.Pos, tx *world.Tx, r *rand.Rand) {
	attemptOxidation(pos, tx, r, c)
}

func (c CopperBars) EncodeItem() (name string, meta int16) {
	return copperBlockName("copper_bars", c.Oxidation, c.Waxed), 0
}

func (c CopperBars) EncodeBlock() (name string, properties map[string]any) {
	return copperBlockName("copper_bars", c.Oxidation, c.Waxed), c.Connections.properties()
}

func allCopperBars() (bars []world.Block) {
	f := func(waxed bool) {
		for _, o := range OxidationTypes() {
			for _, conn := range allConnections() {
				bars = append(bars, CopperBars{Oxidation: o, Waxed: waxed, Connections: conn})
			}
		}
	}
	f(true)
	f(false)
	return
}
