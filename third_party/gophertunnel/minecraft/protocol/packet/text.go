package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	TextTypeRaw = iota
	TextTypeChat
	TextTypeTranslation
	TextTypePopup
	TextTypeJukeboxPopup
	TextTypeTip
	TextTypeSystem
	TextTypeWhisper
	TextTypeAnnouncement
	TextTypeObjectWhisper
	TextTypeObject
	TextTypeObjectAnnouncement
)

const (
	TextCategoryMessageOnly = iota
	TextCategoryAuthoredMessage
	TextCategoryMessageWithParameters
)

type Text struct {
	TextType byte

	NeedsTranslation bool

	SourceName string

	Message string

	Parameters []string

	XUID string

	PlatformChatID string

	FilteredMessage protocol.Optional[string]
}

func (*Text) ID() uint32 {
	return IDText
}

func (pk *Text) Marshal(io protocol.IO) {
	io.Bool(&pk.NeedsTranslation)

	var categoryType uint8
	switch pk.TextType {
	case TextTypeRaw, TextTypeTip, TextTypeSystem, TextTypeObjectWhisper, TextTypeObjectAnnouncement, TextTypeObject:
		categoryType = TextCategoryMessageOnly
	case TextTypeChat, TextTypeWhisper, TextTypeAnnouncement:
		categoryType = TextCategoryAuthoredMessage
	default:
		categoryType = TextCategoryMessageWithParameters
	}
	io.Uint8(&categoryType)
	io.Uint8(&pk.TextType)
	switch pk.TextType {
	case TextTypeChat, TextTypeWhisper, TextTypeAnnouncement:
		io.String(&pk.SourceName)
		if len(pk.SourceName) > 256 {
			io.InvalidValue(pk.SourceName, "source name", "string too long")
		}
		io.String(&pk.Message)
	case TextTypeRaw, TextTypeTip, TextTypeSystem, TextTypeObject, TextTypeObjectWhisper, TextTypeObjectAnnouncement:
		io.String(&pk.Message)
	case TextTypeTranslation, TextTypePopup, TextTypeJukeboxPopup:
		io.String(&pk.Message)
		protocol.FuncSlice(io, &pk.Parameters, io.String)
		if len(pk.Parameters) > 4 {
			io.InvalidValue(pk.Parameters, "parameters", "too many parameters")
		}
	}

	if len(pk.Message) == 0 {
		io.InvalidValue(pk.Message, "message", "string cannot be empty")
	}
	if len(pk.Message) > 65536 {
		io.InvalidValue(pk.Message, "message", "string too long")
	}
	io.String(&pk.XUID)
	if len(pk.XUID) > 64 {
		io.InvalidValue(pk.XUID, "XUID", "string too long")
	}
	io.String(&pk.PlatformChatID)
	if len(pk.PlatformChatID) > 256 {
		io.InvalidValue(pk.PlatformChatID, "platform chat ID", "string too long")
	}
	protocol.OptionalFunc(io, &pk.FilteredMessage, io.String)
	if filteredMessage, ok := pk.FilteredMessage.Value(); ok && len(filteredMessage) > 65536 {
		io.InvalidValue(filteredMessage, "filtered message", "string too long")
	}
}
