package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ContainerRegistryCleanup struct {
	RemovedContainers []protocol.FullContainerName
}

func (*ContainerRegistryCleanup) ID() uint32 {
	return IDContainerRegistryCleanup
}

func (pk *ContainerRegistryCleanup) Marshal(io protocol.IO) {
	protocol.Slice(io, &pk.RemovedContainers)
}
