package minecraft

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDeferredInitialisationKeepsLoginAndTerrainFlowing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	listener, err := (ListenConfig{AuthenticationDisabled: true}).Listen("nethernet", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	serverDone := make(chan error, 1)
	go func() {
		server, err := listener.AcceptMinecraft()
		if err != nil {
			serverDone <- err
			return
		}
		defer server.Close()
		err = server.StartGameContextWithReady(ctx, GameData{WorldName: "CrabSMP ordering regression", EntityRuntimeID: 6668, EntityUniqueID: 6668}, func() {
			_ = server.WritePacket(&packet.Text{TextType: packet.TextTypeRaw, Message: "terrain continues before initialisation"})
			_ = server.Flush()
		})
		serverDone <- err
		<-ctx.Done()
	}()
	var initialisations atomic.Int32
	client, err := (Dialer{
		EarlyConnHandler: func(conn *Conn) { conn.SetPlayerInitialisationDeferred(true) },
		PacketFunc: func(h packet.Header, _ []byte, _, _ net.Addr) {
			if h.PacketID == packet.IDSetLocalPlayerAsInitialised {
				initialisations.Add(1)
			}
		},
	}).DialContext(ctx, "nethernet", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.DoSpawnContext(ctx); err != nil {
		t.Fatal(err)
	}
	if initialisations.Load() != 0 {
		t.Fatal("login prematurely initialised upstream player")
	}
	select {
	case err := <-serverDone:
		t.Fatalf("upstream considered player loaded before confirmation: %v", err)
	default:
	}
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		pk, err := client.ReadPacket()
		if err != nil {
			t.Fatal(err)
		}
		if text, ok := pk.(*packet.Text); ok && text.Message == "terrain continues before initialisation" {
			break
		}
	}
	if err := client.WritePacket(&packet.SetLocalPlayerAsInitialised{EntityRuntimeID: 6668}); err != nil {
		t.Fatal(err)
	}
	if err := client.Flush(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("upstream never accepted deferred initialisation")
	}
	if initialisations.Load() != 1 {
		t.Fatalf("sent %d initialisations, want one", initialisations.Load())
	}
}
