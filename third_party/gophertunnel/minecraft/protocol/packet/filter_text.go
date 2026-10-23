package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type FilterText struct {
	Text string

	FromServer bool
}

func (*FilterText) ID() uint32 {
	return IDFilterText
}

func (pk *FilterText) Marshal(io protocol.IO) {
	io.String(&pk.Text)
	io.Bool(&pk.FromServer)
}
