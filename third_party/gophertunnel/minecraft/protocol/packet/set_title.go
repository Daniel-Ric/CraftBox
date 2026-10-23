package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	TitleActionClear = iota
	TitleActionReset
	TitleActionSetTitle
	TitleActionSetSubtitle
	TitleActionSetActionBar
	TitleActionSetDurations
	TitleActionTitleTextObject
	TitleActionSubtitleTextObject
	TitleActionActionbarTextObject
)

type SetTitle struct {
	ActionType int32

	Text string

	FadeInDuration int32

	RemainDuration int32

	FadeOutDuration int32

	XUID string

	PlatformOnlineID string

	FilteredMessage string
}

func (*SetTitle) ID() uint32 {
	return IDSetTitle
}

func (pk *SetTitle) Marshal(io protocol.IO) {
	io.Varint32(&pk.ActionType)
	io.String(&pk.Text)
	io.Varint32(&pk.FadeInDuration)
	io.Varint32(&pk.RemainDuration)
	io.Varint32(&pk.FadeOutDuration)
	io.String(&pk.XUID)
	io.String(&pk.PlatformOnlineID)
	io.String(&pk.FilteredMessage)
}
