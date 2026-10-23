package session

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"sync/atomic"
)

type ServerBoundLoadingScreenHandler struct {
	currentID  atomic.Uint32
	expectedID atomic.Uint32
}

func (h *ServerBoundLoadingScreenHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, _ Controllable) error {
	pk := p.(*packet.ServerBoundLoadingScreen)
	v, ok := pk.LoadingScreenID.Value()
	expected := h.expectedID.Load()

	switch {
	case !ok || expected == 0:
		return nil
	case v != expected:
		return fmt.Errorf("expected loading screen ID %d, got %d", expected, v)
	case pk.Type == packet.LoadingScreenTypeEnd:
		s.changingDimension.Store(false)
		h.expectedID.Store(0)
	}

	return nil
}
