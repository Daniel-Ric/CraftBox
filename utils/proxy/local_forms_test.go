package proxy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestLocalFormRepliesStayLocalAndRunOnce(t *testing.T) {
	client := &initialisationTestConn{}
	s := &Session{Client: client}
	calls := 0
	var received string
	if err := s.ShowLocalForm(map[string]any{"type": "form", "title": "CraftBox"}, func(data json.RawMessage) {
		calls++
		received = string(data)
		// Opening a submenu from a response must not deadlock.
		if err := s.ShowLocalForm(map[string]any{"type": "form"}, nil); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	local := client.writes[0].(*packet.ModalFormRequest)
	// Even a server choosing the local form's ID must not collide on the wire.
	server, err := s.commandHandlerPacketCB(&packet.ModalFormRequest{FormID: local.FormID}, false, time.Time{}, false)
	if err != nil || server.(*packet.ModalFormRequest).FormID == local.FormID {
		t.Fatal("server/local collision", err)
	}
	for range 2 {
		forward, err := s.commandHandlerPacketCB(&packet.ModalFormResponse{FormID: local.FormID, ResponseData: protocol.Option([]byte("3"))}, true, time.Time{}, false)
		if err != nil || forward != nil {
			t.Fatal("local response reached upstream", err)
		}
	}
	if calls != 1 || received != "3" {
		t.Fatalf("callback: %d calls, data %q", calls, received)
	}
	serverReply, err := s.commandHandlerPacketCB(&packet.ModalFormResponse{FormID: server.(*packet.ModalFormRequest).FormID, CancelReason: protocol.Option(uint8(packet.ModalFormCancelReasonUserClosed))}, true, time.Time{}, false)
	if err != nil || serverReply.(*packet.ModalFormResponse).FormID != local.FormID {
		t.Fatal("server response was swallowed", err)
	}
}

func TestCancelledMalformedAndReplacedFormsDoNotRun(t *testing.T) {
	for _, tc := range []struct {
		name   string
		data   []byte
		cancel bool
	}{
		{"closed", nil, true},
		{"busy", []byte("0"), true},
		{"null", []byte(" null\n"), false},
		{"invalid JSON", []byte("{"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &initialisationTestConn{}
			s := &Session{Client: client}
			calls := 0
			s.ShowLocalForm(map[string]any{"type": "form"}, func(json.RawMessage) { calls++ })
			id := client.writes[0].(*packet.ModalFormRequest).FormID
			reply := &packet.ModalFormResponse{FormID: id, ResponseData: protocol.Option(tc.data)}
			if tc.cancel {
				reply.CancelReason = protocol.Option(uint8(packet.ModalFormCancelReasonUserBusy))
			}
			forward, err := s.commandHandlerPacketCB(reply, true, time.Time{}, false)
			if err != nil || forward != nil || calls != 0 {
				t.Fatal("invalid/cancelled form executed or leaked")
			}
		})
	}
	client := &initialisationTestConn{}
	s := &Session{Client: client}
	calls := 0
	s.ShowLocalForm(map[string]any{"type": "form"}, func(json.RawMessage) { calls++ })
	oldID := client.writes[0].(*packet.ModalFormRequest).FormID
	s.ShowLocalForm(map[string]any{"type": "form"}, nil)
	forward, err := s.commandHandlerPacketCB(&packet.ModalFormResponse{FormID: oldID, ResponseData: protocol.Option([]byte("0"))}, true, time.Time{}, false)
	if err != nil || forward != nil || calls != 0 {
		t.Fatal("replaced menu executed or leaked")
	}
}

func TestServerFormIDsAreRemappedAndRestored(t *testing.T) {
	s := &Session{}
	var ids []uint32
	for _, original := range []uint32{0, 1, 0xffffffff} {
		pk, err := s.commandHandlerPacketCB(&packet.ModalFormRequest{FormID: original, FormData: []byte(`{"type":"form"}`)}, false, time.Time{}, false)
		if err != nil {
			t.Fatal(err)
		}
		id := pk.(*packet.ModalFormRequest).FormID
		for _, previous := range ids {
			if id == previous {
				t.Fatal("wire form IDs collided")
			}
		}
		ids = append(ids, id)
		response, err := s.commandHandlerPacketCB(&packet.ModalFormResponse{FormID: id, ResponseData: protocol.Option([]byte("2"))}, true, time.Time{}, false)
		if err != nil || response.(*packet.ModalFormResponse).FormID != original {
			t.Fatalf("server form ID not restored: %v, %v", response, err)
		}
		duplicate, err := s.commandHandlerPacketCB(&packet.ModalFormResponse{FormID: id}, true, time.Time{}, false)
		if err != nil || duplicate != nil {
			t.Fatal("duplicate response was forwarded")
		}
	}
}
