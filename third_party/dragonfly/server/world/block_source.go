package world

import "github.com/df-mc/dragonfly/server/block/cube"

type BlockSource interface {
	Block(cube.Pos) Block
}

type StateDeriver interface {
	Block

	DeriveState(pos cube.Pos, src BlockSource) Block
}

type worldSource struct{ tx *Tx }

func (w worldSource) Block(pos cube.Pos) Block { return w.tx.block(pos) }
