package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type EditorNetwork struct {
	RouteToManager bool

	Payload map[string]any
}

func (*EditorNetwork) ID() uint32 {
	return IDEditorNetwork
}

func (pk *EditorNetwork) Marshal(io protocol.IO) {
	io.Bool(&pk.RouteToManager)
	io.NBT(&pk.Payload, nbt.NetworkLittleEndian)
}
