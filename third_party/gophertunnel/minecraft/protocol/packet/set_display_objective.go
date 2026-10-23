package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ScoreboardSortOrderAscending = iota
	ScoreboardSortOrderDescending
)

const (
	ScoreboardSlotList      = "list"
	ScoreboardSlotSidebar   = "sidebar"
	ScoreboardSlotBelowName = "belowname"
)

type SetDisplayObjective struct {
	DisplaySlot string

	ObjectiveName string

	DisplayName string

	CriteriaName string

	SortOrder int32
}

func (*SetDisplayObjective) ID() uint32 {
	return IDSetDisplayObjective
}

func (pk *SetDisplayObjective) Marshal(io protocol.IO) {
	io.String(&pk.DisplaySlot)
	io.String(&pk.ObjectiveName)
	io.String(&pk.DisplayName)
	io.String(&pk.CriteriaName)
	io.Varint32(&pk.SortOrder)
}
