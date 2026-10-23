package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	PlayStatusLoginSuccess int32 = iota
	PlayStatusLoginFailedClient
	PlayStatusLoginFailedServer
	PlayStatusPlayerSpawn
	PlayStatusLoginFailedInvalidTenant
	PlayStatusLoginFailedVanillaEdu
	PlayStatusLoginFailedEduVanilla
	PlayStatusLoginFailedServerFull
	PlayStatusLoginFailedEditorVanilla
	PlayStatusLoginFailedVanillaEditor
)

type PlayStatus struct {
	Status int32
}

func (*PlayStatus) ID() uint32 {
	return IDPlayStatus
}

func (pk *PlayStatus) Marshal(io protocol.IO) {
	io.BEInt32(&pk.Status)
}
