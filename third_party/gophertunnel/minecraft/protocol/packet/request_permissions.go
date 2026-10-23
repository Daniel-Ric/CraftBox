package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RequestPermissions struct {
	EntityUniqueID int64

	PermissionLevel int32

	RequestedPermissions uint16
}

func (*RequestPermissions) ID() uint32 {
	return IDRequestPermissions
}

func (pk *RequestPermissions) Marshal(io protocol.IO) {
	io.ActorUniqueIDInt64(&pk.EntityUniqueID)
	io.Varint32(&pk.PermissionLevel)
	io.Uint16(&pk.RequestedPermissions)
}
