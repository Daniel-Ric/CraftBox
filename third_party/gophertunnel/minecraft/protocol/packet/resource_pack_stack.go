package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ResourcePackStack struct {
	TexturePackRequired bool

	TexturePacks []protocol.StackResourcePack

	BaseGameVersion string

	Experiments []protocol.ExperimentData

	ExperimentsPreviouslyToggled bool

	IncludeEditorPacks bool
}

func (*ResourcePackStack) ID() uint32 {
	return IDResourcePackStack
}

func (pk *ResourcePackStack) Marshal(io protocol.IO) {
	io.Bool(&pk.TexturePackRequired)
	protocol.Slice(io, &pk.TexturePacks)
	io.String(&pk.BaseGameVersion)
	protocol.SliceUint32Length(io, &pk.Experiments)
	io.Bool(&pk.ExperimentsPreviouslyToggled)
	io.Bool(&pk.IncludeEditorPacks)
}
