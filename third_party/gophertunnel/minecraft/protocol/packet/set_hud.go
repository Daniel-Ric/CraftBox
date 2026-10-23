package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	HudElementPaperDoll = iota
	HudElementArmour
	HudElementToolTips
	HudElementTouchControls
	HudElementCrosshair
	HudElementHotBar
	HudElementHealth
	HudElementProgressBar
	HudElementHunger
	HudElementAirBubbles
	HudElementHorseHealth
	HudElementStatusEffects
	HudElementItemText
)

const (
	HudVisibilityHide = iota
	HudVisibilityReset
)

type SetHud struct {
	Elements []int32

	Visibility int32
}

func (*SetHud) ID() uint32 {
	return IDSetHud
}

func (pk *SetHud) Marshal(io protocol.IO) {
	protocol.FuncSlice(io, &pk.Elements, io.Varint32)
	io.Varint32(&pk.Visibility)
}
