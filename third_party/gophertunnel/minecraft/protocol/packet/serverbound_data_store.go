package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerBoundDataStore struct {
	Update protocol.DataStoreUpdate
}

func (*ServerBoundDataStore) ID() uint32 {
	return IDServerBoundDataStore
}

func (pk *ServerBoundDataStore) Marshal(io protocol.IO) {
	protocol.Single(io, &pk.Update)
}
