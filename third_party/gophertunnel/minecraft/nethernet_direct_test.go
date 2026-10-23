package minecraft

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDirectNetherNetMinecraftLogin(t *testing.T) {
	clientData := make(chan login.ClientData, 1)
	textReceived := make(chan struct{}, 1)
	cfg := ListenConfig{AuthenticationDisabled: true, StatusProvider: NewStatusProvider("NetherNet proxy", "CraftBox"), OnClientData: func(c *Conn) { clientData <- c.ClientData() }}
	l, err := cfg.Listen("nethernet", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	accepted := make(chan *Conn, 1)
	spawned := make(chan error, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		accepted <- c.(*Conn)
		spawned <- c.(*Conn).StartGameContext(ctx, GameData{WorldName: "NetherNet test", EntityUniqueID: 1, EntityRuntimeID: 1})
	}()
	client, err := (Dialer{
		GetClientData: func() login.ClientData { return login.ClientData{LanguageCode: "de_DE"} },
		PacketFunc: func(h packet.Header, _ []byte, _, _ net.Addr) {
			if h.PacketID == packet.IDText {
				textReceived <- struct{}{}
			}
		},
	}).DialContext(ctx, "nethernet", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var server *Conn
	select {
	case server = <-accepted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	defer server.Close()
	select {
	case data := <-clientData:
		if data.LanguageCode != "de_DE" {
			t.Fatalf("proxy client language = %q", data.LanguageCode)
		}
	case <-ctx.Done():
		t.Fatal("client-data callback did not run")
	}
	if err := <-spawned; err != nil {
		t.Fatal(err)
	}
	if client.IdentityData().Identity != server.IdentityData().Identity {
		t.Fatal("login identity changed")
	}
	if err := server.WritePacket(&packet.Text{TextType: packet.TextTypeRaw, Message: "nethernet-login-ok"}); err != nil {
		t.Fatal(err)
	}
	if err := server.Flush(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-textReceived:
	case <-ctx.Done():
		t.Fatal("text never reached transport callback")
	}
	readStarted := time.Now()
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		pk, receivedAt, err := client.ReadPacketWithTime()
		if err != nil {
			t.Fatal(err)
		}
		if text, ok := pk.(*packet.Text); ok {
			if receivedAt.IsZero() || receivedAt.After(readStarted) {
				t.Fatal("capture timestamp was taken at read rather than receipt")
			}
			if text.Message != "nethernet-login-ok" {
				t.Fatalf("received %#v", text)
			}
			break
		}
	}
}
