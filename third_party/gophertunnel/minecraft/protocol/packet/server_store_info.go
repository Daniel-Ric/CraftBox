package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ServerStoreInfo struct {
	StoreInfo protocol.Optional[protocol.StoreEntryPointInfo]
}

func (*ServerStoreInfo) ID() uint32 {
	return IDServerStoreInfo
}

func (pk *ServerStoreInfo) Marshal(io protocol.IO) {
	protocol.OptionalMarshaler(io, &pk.StoreInfo)
}
