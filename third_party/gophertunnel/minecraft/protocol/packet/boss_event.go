package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	BossEventShow = iota
	BossEventRegisterPlayer
	BossEventHide
	BossEventUnregisterPlayer
	BossEventHealthPercentage
	BossEventTitle
	BossEventAppearanceProperties
	BossEventTexture
	BossEventRequest
)

const (
	BossEventColourPink = iota
	BossEventColourBlue
	BossEventColourRed
	BossEventColourGreen
	BossEventColourYellow
	BossEventColourPurple
	BossEventColourRebeccaPurple
	BossEventColourWhite
)

const (
	BossEventOverlayProgress = iota
	BossEventOverlayNotched6
	BossEventOverlayNotched10
	BossEventOverlayNotched12
	BossEventOverlayNotched20
)

type BossEvent struct {
	BossEntityUniqueID int64

	EventType uint8

	BossBarTitle string

	FilteredBossBarTitle string

	HealthPercentage float32

	Colour uint8

	Overlay uint8
}

func (*BossEvent) ID() uint32 {
	return IDBossEvent
}

func (pk *BossEvent) Marshal(io protocol.IO) {
	io.ActorUniqueID(&pk.BossEntityUniqueID)
	io.Uint8(&pk.EventType)
	io.String(&pk.BossBarTitle)
	io.String(&pk.FilteredBossBarTitle)
	io.Float32(&pk.HealthPercentage)
	io.Uint8(&pk.Colour)
	io.Uint8(&pk.Overlay)
}
