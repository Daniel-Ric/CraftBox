package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	BookActionReplacePage = iota
	BookActionAddPage
	BookActionDeletePage
	BookActionSwapPages
	BookActionSign
)

type BookEdit struct {
	InventorySlot int32

	ActionType uint32

	PageNumber int32

	SecondaryPageNumber int32

	Text string

	PhotoName string

	Title string

	Author string

	XUID string
}

func (*BookEdit) ID() uint32 {
	return IDBookEdit
}

func (pk *BookEdit) Marshal(io protocol.IO) {
	io.Varint32(&pk.InventorySlot)
	io.Varuint32(&pk.ActionType)
	switch pk.ActionType {
	case BookActionReplacePage, BookActionAddPage:
		io.Varint32(&pk.PageNumber)
		io.String(&pk.Text)
		io.String(&pk.PhotoName)
	case BookActionDeletePage:
		io.Varint32(&pk.PageNumber)
	case BookActionSwapPages:
		io.Varint32(&pk.PageNumber)
		io.Varint32(&pk.SecondaryPageNumber)
	case BookActionSign:
		io.String(&pk.Title)
		io.String(&pk.Author)
		io.String(&pk.XUID)
	default:
		io.UnknownEnumOption(pk.ActionType, "book edit action type")
	}
}
