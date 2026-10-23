package session

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"sync"
	"sync/atomic"
)

type ModalFormResponseHandler struct {
	mu        sync.Mutex
	forms     map[uint32]form.Form
	currentID atomic.Uint32
}

func (h *ModalFormResponseHandler) Handle(p packet.Packet, _ *Session, tx *world.Tx, c Controllable) error {
	pk := p.(*packet.ModalFormResponse)

	h.mu.Lock()
	f, ok := h.forms[pk.FormID]
	delete(h.forms, pk.FormID)
	h.mu.Unlock()

	resp, exists := pk.ResponseData.Value()
	if !ok && !exists {

		return nil
	}
	if !exists || len(resp) == 0 {

		resp = nil
	}
	if !ok {
		return fmt.Errorf("no form with ID %v currently opened", pk.FormID)
	}
	if err := f.SubmitJSON(resp, c, tx); err != nil {
		return fmt.Errorf("error submitting form data: %w", err)
	}
	return nil
}
