package proxy

import (
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	nethernet "github.com/df-mc/go-nethernet"
	"github.com/pion/webrtc/v4"
	"strings"
	"testing"
)

func TestConnectionNetherNetCandidate(t *testing.T) {
	address := &nethernet.Addr{NetworkID: "private-peer-id", ConnectionID: 42, SelectedCandidate: &webrtc.ICECandidate{Address: "2001:db8::1", Port: 19132, Protocol: webrtc.ICEProtocolUDP, Foundation: "private-foundation"}}
	if got := connectionAddress(address); got != "[2001:db8::1]:19132 (ICE)" {
		t.Fatal(got)
	}
}

func TestConnectionDetailsRedactsCredentials(t *testing.T) {
	s := &Session{connectInfo: &connectinfo.ConnectInfo{Value: "https://example.org:19132"}}
	text := s.ConnectionDetailsText() + connectionEndpoint("https://user:secret@example.org:19132/private?token=hidden#secret")
	for _, secret := range []string{"user", "secret", "private", "hidden", "token="} {
		if strings.Contains(text, secret) {
			t.Fatalf("credential leaked: %s", text)
		}
	}
	if !strings.Contains(text, "example.org:19132") || !strings.Contains(text, "Waiting") {
		t.Fatal(text)
	}
}

func TestConnectionDetailsWithoutServer(t *testing.T) {
	text := (&Session{}).ConnectionDetailsText()
	if !strings.Contains(text, "Waiting") || !strings.Contains(text, "Protocol") {
		t.Fatal(text)
	}
}
