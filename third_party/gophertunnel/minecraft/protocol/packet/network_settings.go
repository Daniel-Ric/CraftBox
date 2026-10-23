package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CompressionAlgorithmFlate = iota
	CompressionAlgorithmSnappy
	CompressionAlgorithmNone = 0xffff
)

type NetworkSettings struct {
	CompressionThreshold uint16

	CompressionAlgorithm uint16

	ClientThrottle bool

	ClientThrottleThreshold uint8

	ClientThrottleScalar float32
}

func (*NetworkSettings) ID() uint32 {
	return IDNetworkSettings
}

func (pk *NetworkSettings) Marshal(io protocol.IO) {
	io.Uint16(&pk.CompressionThreshold)
	io.Uint16(&pk.CompressionAlgorithm)
	io.Bool(&pk.ClientThrottle)
	io.Uint8(&pk.ClientThrottleThreshold)
	io.Float32(&pk.ClientThrottleScalar)
}
