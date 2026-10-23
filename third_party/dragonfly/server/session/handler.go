package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type packetHandler interface {
	Handle(p packet.Packet, s *Session, tx *world.Tx, c Controllable) error
}
