package minecraft

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"strings"
	"testing"
)

func TestDisconnectWithoutMessagePreservesReason(t *testing.T) {
	err := disconnectPacketError(&packet.Disconnect{Reason: packet.DisconnectReasonKicked})
	if !strings.Contains(err.Error(), "disconnect code 55") || !strings.Contains(err.Error(), "removed you") {
		t.Fatalf("reason lost: %q", err)
	}
}

func TestDisconnectPreservesServerMessage(t *testing.T) {
	err := disconnectPacketError(&packet.Disconnect{Reason: packet.DisconnectReasonKicked, Message: "Host closed this session"})
	if !strings.Contains(err.Error(), "Host closed this session") || !strings.Contains(err.Error(), "55") {
		t.Fatal(err)
	}
}
