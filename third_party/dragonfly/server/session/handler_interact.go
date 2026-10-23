package session

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type InteractHandler struct{}

func (h *InteractHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.Interact)
	pos := c.Position()

	switch pk.ActionType {
	case packet.InteractActionMouseOverEntity:

	case packet.InteractActionOpenInventory:
		if s.invOpened {

			return nil
		}
		s.invOpened = true
		s.writePacket(&packet.ContainerOpen{
			WindowID:                0,
			ContainerType:           0xff,
			ContainerEntityUniqueID: -1,
			ContainerPosition: protocol.BlockPos{
				int32(pos[0]),
				int32(pos[1]),
				int32(pos[2]),
			},
		})
	default:
		return fmt.Errorf("unexpected interact packet action %v", pk.ActionType)
	}
	return nil
}
