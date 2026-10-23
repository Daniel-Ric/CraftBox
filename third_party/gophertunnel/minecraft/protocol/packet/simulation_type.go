package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

const (
	SimulationTypeGame byte = iota
	SimulationTypeEditor
	SimulationTypeTest
	SimulationTypeInvalid
)

type SimulationType struct {
	SimulationType byte
}

func (*SimulationType) ID() uint32 {
	return IDSimulationType
}

func (pk *SimulationType) Marshal(io protocol.IO) {
	io.Uint8(&pk.SimulationType)
}
