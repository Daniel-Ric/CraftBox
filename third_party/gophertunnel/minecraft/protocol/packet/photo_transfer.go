package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PhotoTypePortfolio uint8 = iota
	PhotoTypePhotoItem
	PhotoTypeBook
)

type PhotoTransfer struct {
	PhotoName string

	PhotoData []byte

	BookID string

	PhotoType byte

	SourceType byte

	OwnerEntityUniqueID int64

	NewPhotoName string
}

func (*PhotoTransfer) ID() uint32 {
	return IDPhotoTransfer
}

func (pk *PhotoTransfer) Marshal(io protocol.IO) {
	io.String(&pk.PhotoName)
	io.ByteSlice(&pk.PhotoData)
	io.String(&pk.BookID)
	io.Uint8(&pk.PhotoType)
	io.Uint8(&pk.SourceType)
	io.ActorUniqueIDInt64(&pk.OwnerEntityUniqueID)
	io.String(&pk.NewPhotoName)
}
