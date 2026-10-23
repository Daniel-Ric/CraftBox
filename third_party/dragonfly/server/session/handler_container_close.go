package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type ContainerCloseHandler struct{}

func (h *ContainerCloseHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, c Controllable) error {
	pk := p.(*packet.ContainerClose)

	c.MoveItemsToInventory()

	var containerType byte
	switch pk.WindowID {
	case 0:

		s.invOpened = false
	case byte(s.openedWindowID.Load()):
		containerType = byte(s.openedContainerID.Load())
		s.closeCurrentContainer(tx, true)
	case 0xff:

		s.invOpened = false
		if s.containerOpened.Load() {
			s.closeCurrentContainer(tx, false)
		}
		return nil
	default:
		containerType = pk.ContainerType
	}
	s.writePacket(&packet.ContainerClose{
		WindowID:      pk.WindowID,
		ContainerType: containerType,
	})
	return nil
}
