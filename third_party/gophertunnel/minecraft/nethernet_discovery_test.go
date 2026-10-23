package minecraft

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"github.com/df-mc/go-nethernet"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/df-mc/go-nethernet/discovery"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

func freeDiscoveryAddress(t *testing.T) string {
	t.Helper()
	c, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := c.LocalAddr().String()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func TestDirectNetherNetLANAndHTTPMinecraftJoin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var identities atomic.Int32
	issuer := func(context.Context) (*nethernet.Identity, error) {
		identities.Add(1)
		return nethernet.GenerateServerIdentity(key, "self")
	}
	address := freeDiscoveryAddress(t)
	l, err := (ListenConfig{AuthenticationDisabled: true, ErrorLog: logger, StatusProvider: NewStatusProvider("LAN proxy", "Discovery world")}).ListenNetwork(DirectNetherNet{DiscoveryAddress: address, Log: logger, IssueServerIdentity: issuer}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	target, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (discovery.ListenConfig{BroadcastAddress: target}).Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var id uint64
	var data discovery.ServerData
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
search:
	for {
		select {
		case <-ctx.Done():
			t.Fatal("LAN server was not discovered:", ctx.Err())
		case <-ticker.C:
			for networkID, response := range d.Responses() {
				if err := data.UnmarshalBinary(response); err != nil {
					t.Fatal(err)
				}
				id = networkID
				break search
			}
		}
	}
	if data.Protocol != protocol.CurrentProtocol || data.Version != protocol.CurrentVersion || data.ServerName != "LAN proxy" || data.LevelName != "Discovery world" || data.ConnectionType != 4 || !data.AcceptsSelfSignedAuth || data.Nonce == "" {
		t.Fatalf("invalid LAN announcement: %#v", data)
	}
	spawned := make(chan error, 2)
	go func() {
		for range 2 {
			c, err := l.AcceptMinecraft()
			if err != nil {
				spawned <- err
				return
			}
			go func() {
				defer c.Close()
				err := c.StartGameContext(ctx, GameData{WorldName: "Dual transport world", EntityUniqueID: 1, EntityRuntimeID: 1})
				spawned <- err
				<-ctx.Done()
			}()
		}
	}()
	lan, err := (Dialer{ClientData: login.ClientData{Nonce: data.Nonce}}).DialContextNetwork(ctx, NetherNet{Signaling: d}, strconv.FormatUint(id, 10))
	if err != nil {
		t.Fatal("LAN join:", err)
	}
	defer lan.Close()
	httpClient, err := (Dialer{}).DialContext(ctx, "nethernet", "http://"+l.Addr().String())
	if err != nil {
		t.Fatal("HTTP join:", err)
	}
	defer httpClient.Close()
	for _, c := range []*Conn{lan, httpClient} {
		if c.GameData().WorldName != "Dual transport world" {
			t.Fatalf("world = %q", c.GameData().WorldName)
		}
		if err := <-spawned; err != nil {
			t.Fatal(err)
		}
	}
	if identities.Load() != 2 {
		t.Fatalf("shared identity issuer called %d times, want both LAN and HTTP", identities.Load())
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	socket, err := net.ListenPacket("udp4", address)
	if err != nil {
		t.Fatal("LAN socket leaked:", err)
	}
	socket.Close()
}

func TestDirectNetherNetExplicitDiscoveryBindFailure(t *testing.T) {
	socket, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	l, err := (DirectNetherNet{DiscoveryAddress: socket.LocalAddr().String()}).Listen("127.0.0.1:0")
	if err == nil {
		l.Close()
		t.Fatal("explicit occupied LAN address was ignored")
	}
	if errors.Is(err, net.ErrClosed) {
		t.Fatalf("wrong bind error: %v", err)
	}
}
