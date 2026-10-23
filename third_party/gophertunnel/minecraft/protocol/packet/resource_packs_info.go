package packet

import (
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ResourcePacksInfo struct {
	TexturePackRequired bool

	HasAddons bool

	HasScripts bool

	ForceDisableVibrantVisuals bool

	WorldTemplateUUID uuid.UUID

	WorldTemplateVersion string

	TexturePacks []protocol.TexturePackInfo
}

func (*ResourcePacksInfo) ID() uint32 {
	return IDResourcePacksInfo
}

func (pk *ResourcePacksInfo) Marshal(io protocol.IO) {
	io.Bool(&pk.TexturePackRequired)
	io.Bool(&pk.HasAddons)
	io.Bool(&pk.HasScripts)
	io.Bool(&pk.ForceDisableVibrantVisuals)
	io.UUID(&pk.WorldTemplateUUID)
	io.String(&pk.WorldTemplateVersion)
	protocol.Slice(io, &pk.TexturePacks)
}
