package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

const (
	GameTestRequestRotation0 = iota
	GameTestRequestRotation90
	GameTestRequestRotation180
	GameTestRequestRotation270
	GameTestRequestRotation360
)

type GameTestRequest struct {
	Name string

	Rotation uint8

	Repetitions int32

	Position protocol.BlockPos

	StopOnError bool

	TestsPerRow int32

	MaxTestsPerBatch int32
}

func (pk *GameTestRequest) ID() uint32 {
	return IDGameTestRequest
}

func (pk *GameTestRequest) Marshal(io protocol.IO) {
	io.Varint32(&pk.MaxTestsPerBatch)
	io.Varint32(&pk.Repetitions)
	io.Uint8(&pk.Rotation)
	io.Bool(&pk.StopOnError)
	io.BlockPos(&pk.Position)
	io.Varint32(&pk.TestsPerRow)
	io.String(&pk.Name)
}
