package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type FeatureRegistry struct {
	Features []protocol.GenerationFeature
}

func (pk *FeatureRegistry) ID() uint32 {
	return IDFeatureRegistry
}

func (pk *FeatureRegistry) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.Features)
}
