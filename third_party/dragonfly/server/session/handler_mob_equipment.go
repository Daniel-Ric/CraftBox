package session

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type MobEquipmentHandler struct{}

func (*MobEquipmentHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, c Controllable) error {
	pk := p.(*packet.MobEquipment)

	if pk.EntityRuntimeID != selfEntityRuntimeID {
		return errSelfRuntimeID
	}
	switch pk.WindowID {
	case protocol.WindowIDOffHand:

		return nil
	case protocol.WindowIDInventory:
		return s.VerifyAndSetHeldSlot(int(pk.InventorySlot), stackToItem(s.br, pk.NewItem.Stack), c)
	default:
		return fmt.Errorf("only main inventory should be involved in slot change, got window ID %v", pk.WindowID)
	}
}
