package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundDataStore struct {
	Updates []protocol.DataStoreChangeEntry
}

func (*ClientBoundDataStore) ID() uint32 {
	return IDClientBoundDataStore
}

func (pk *ClientBoundDataStore) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Updates)
}
