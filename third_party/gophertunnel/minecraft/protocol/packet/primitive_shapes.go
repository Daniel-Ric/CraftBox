package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type PrimitiveShapes struct {
	Shapes []protocol.PrimitiveShape
}

func (pk *PrimitiveShapes) ID() uint32 {
	return IDPrimitiveShapes
}

func (pk *PrimitiveShapes) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Shapes)
}
