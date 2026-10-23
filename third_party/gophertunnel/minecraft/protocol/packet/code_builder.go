package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type CodeBuilder struct {
	URL string

	ShouldOpenCodeBuilder bool
}

func (*CodeBuilder) ID() uint32 {
	return IDCodeBuilder
}

func (pk *CodeBuilder) Marshal(io protocol.IO) {
	io.String(&pk.URL)
	io.Bool(&pk.ShouldOpenCodeBuilder)
}
