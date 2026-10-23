package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PackResponseRefused = iota
	PackResponseSendPacks
	PackResponseAllPacksDownloaded
	PackResponseCompleted
)

type ResourcePackClientResponse struct {
	Response uint32

	PacksToDownload []string
}

func (*ResourcePackClientResponse) ID() uint32 {
	return IDResourcePackClientResponse
}

func (pk *ResourcePackClientResponse) Marshal(io protocol.IO) {
	io.Varuint32(&pk.Response)
	name, ok := resourcePackResponseToString(pk.Response)
	if !ok {
		io.UnknownEnumOption(pk.Response, "resource pack response")
		return
	}
	io.String(&name)

	if pk.Response == PackResponseSendPacks {
		protocol.FuncSlice(io, &pk.PacksToDownload, io.String)
	}
}

func resourcePackResponseToString(x uint32) (string, bool) {
	names := [...]string{"cancel", "downloading", "downloadingfinished", "resourcepackstackfinished"}
	if x >= uint32(len(names)) {
		return "", false
	}
	return names[x], true
}
