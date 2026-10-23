package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type CreatePhoto struct {
	EntityUniqueID int64

	PhotoName string

	ItemName string
}

func (*CreatePhoto) ID() uint32 {
	return IDCreatePhoto
}

func (pk *CreatePhoto) Marshal(io protocol.IO) {
	io.ActorUniqueIDInt64(&pk.EntityUniqueID)
	io.String(&pk.PhotoName)
	io.String(&pk.ItemName)
}
