package block

import (
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
)

type solid struct{}

func (solid) Model() world.BlockModel {
	return model.Solid{}
}

type empty struct{}

func (empty) Model() world.BlockModel {
	return model.Empty{}
}

type chest struct{}

func (chest) Model() world.BlockModel {
	return model.Chest{}
}

type carpet struct{}

func (carpet) Model() world.BlockModel {
	return model.Carpet{}
}

type tilledGrass struct{}

func (tilledGrass) Model() world.BlockModel {
	return model.TilledGrass{}
}

type leaves struct{}

func (leaves) Model() world.BlockModel {
	return model.Leaves{}
}

type thin struct{}

func (thin) Model() world.BlockModel {
	return model.Thin{}
}
