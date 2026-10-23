package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ViolationTypeMalformed = iota
)

const (
	ViolationSeverityWarning = iota
	ViolationSeverityFinalWarning
	ViolationSeverityTerminatingConnection
)

type PacketViolationWarning struct {
	Type int32

	Severity int32

	PacketID int32

	ViolationContext string
}

func (*PacketViolationWarning) ID() uint32 {
	return IDPacketViolationWarning
}

func (pk *PacketViolationWarning) Marshal(io protocol.IO) {
	io.Varint32(&pk.Type)
	io.Varint32(&pk.Severity)
	io.Varint32(&pk.PacketID)
	io.String(&pk.ViolationContext)
}
