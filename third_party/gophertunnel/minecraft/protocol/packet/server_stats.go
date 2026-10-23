package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type ServerStats struct {
	ServerTime float32

	NetworkTime float32
}

func (pk *ServerStats) ID() uint32 {
	return IDServerStats
}

func (pk *ServerStats) Marshal(io protocol.IO) {
	io.Float32(&pk.ServerTime)
	io.Float32(&pk.NetworkTime)
}
