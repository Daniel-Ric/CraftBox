package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RecordStarted struct {
	Position protocol.BlockPos

	Handle uint64
}

func (*RecordStarted) ID() uint32 {
	return IDRecordStarted
}

func (pk *RecordStarted) Marshal(io protocol.IO) {
	io.BlockPos(&pk.Position)
	io.Uint64(&pk.Handle)
}
