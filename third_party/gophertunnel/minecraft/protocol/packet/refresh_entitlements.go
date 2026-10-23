package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type RefreshEntitlements struct{}

func (*RefreshEntitlements) ID() uint32 {
	return IDRefreshEntitlements
}

func (*RefreshEntitlements) Marshal(protocol.IO) {}
