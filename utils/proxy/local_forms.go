package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Each client-visible form gets a session-unique ID. Server IDs are restored
// before forwarding replies; local replies and retired IDs never reach upstream.
type localFormState struct {
	mu      sync.Mutex
	next    uint32
	server  map[uint32]uint32
	localID uint32
	local   func(json.RawMessage)
}

func (f *localFormState) registerServer(original uint32) (uint32, error) {
	if len(f.server) >= 128 {
		return 0, fmt.Errorf("too many unanswered server forms")
	}
	id, err := f.allocate()
	if err != nil {
		return 0, err
	}
	if f.server == nil {
		f.server = make(map[uint32]uint32)
	}
	f.server[id] = original
	return id, nil
}

func (f *localFormState) allocate() (uint32, error) {
	if f.next == ^uint32(0) {
		return 0, fmt.Errorf("form ID space exhausted")
	}
	f.next++
	return f.next, nil
}

// ShowLocalForm replaces any outstanding CraftBox form without affecting server
// forms. The callback runs once, on the client packet loop, after releasing mu.
func (s *Session) ShowLocalForm(form any, submit func(json.RawMessage)) error {
	if s.Client == nil {
		return fmt.Errorf("the menu requires a connected Minecraft client")
	}
	data, err := json.Marshal(form)
	if err != nil {
		return err
	}
	f := &s.forms
	f.mu.Lock()
	id, err := f.allocate()
	if err == nil {
		f.localID, f.local = id, submit
	}
	f.mu.Unlock()
	if err != nil {
		return err
	}
	if err = s.ClientWritePacket(&packet.ModalFormRequest{FormID: id, FormData: data}); err != nil {
		f.mu.Lock()
		if f.localID == id {
			f.localID, f.local = 0, nil
		}
		f.mu.Unlock()
	}
	return err
}

func (s *Session) formPacketCB(pk packet.Packet, toServer bool) (packet.Packet, error) {
	f := &s.forms
	switch p := pk.(type) {
	case *packet.ModalFormRequest:
		if toServer {
			return pk, nil
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		id, err := f.registerServer(p.FormID)
		if err != nil {
			return nil, err
		}
		forward := *p
		forward.FormID = id
		return &forward, nil
	case *packet.ServerSettingsResponse:
		if toServer {
			return pk, nil
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		id, err := f.registerServer(p.FormID)
		if err != nil {
			return nil, err
		}
		forward := *p
		forward.FormID = id
		return &forward, nil
	case *packet.ModalFormResponse:
		if !toServer {
			return pk, nil
		}
		f.mu.Lock()
		if original, ok := f.server[p.FormID]; ok {
			delete(f.server, p.FormID)
			f.mu.Unlock()
			forward := *p
			forward.FormID = original
			return &forward, nil
		}
		if p.FormID != 0 && p.FormID == f.localID {
			submit := f.local
			f.localID, f.local = 0, nil
			f.mu.Unlock()
			data, present := p.ResponseData.Value()
			_, cancelled := p.CancelReason.Value()
			if present && !cancelled && submit != nil && len(data) <= 16384 && json.Valid(data) && !bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
				submit(json.RawMessage(data))
			}
			return nil, nil
		}
		issued := p.FormID != 0 && p.FormID <= f.next
		f.mu.Unlock()
		if issued {
			return nil, nil
		}
	}
	return pk, nil
}
