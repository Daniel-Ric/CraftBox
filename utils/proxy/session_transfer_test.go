package proxy

import (
	"context"
	"errors"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/df-mc/go-nethernet/discovery"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func transferTestPeer(t *testing.T, ctx context.Context, lan bool) (*minecraft.Listener, *minecraft.Conn, *minecraft.Conn) {
	t.Helper()
	network := minecraft.DirectNetherNet{DisableLANDiscovery: !lan}
	if lan {
		probe, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		network.DiscoveryAddress = probe.LocalAddr().String()
		probe.Close()
	}
	l, err := (minecraft.ListenConfig{AuthenticationDisabled: true}).ListenNetwork(network, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	accepted := make(chan *minecraft.Conn, 1)
	spawned := make(chan error, 1)
	go func() {
		c, err := l.AcceptMinecraft()
		if err != nil {
			spawned <- err
			return
		}
		accepted <- c
		spawned <- c.StartGameContext(ctx, minecraft.GameData{WorldName: "Transfer regression", EntityUniqueID: 1, EntityRuntimeID: 1})
	}()
	var c *minecraft.Conn
	if !lan {
		c, err = (minecraft.Dialer{}).DialContext(ctx, "nethernet", "http://"+l.Addr().String())
	} else {
		address, _ := net.ResolveUDPAddr("udp4", network.DiscoveryAddress)
		d, listenErr := (discovery.ListenConfig{BroadcastAddress: address}).Listen("127.0.0.1:0")
		if listenErr != nil {
			t.Fatal(listenErr)
		}
		t.Cleanup(func() { d.Close() })
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
	found:
		for {
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-ticker.C:
				for id, response := range d.Responses() {
					var data discovery.ServerData
					if parseErr := data.UnmarshalBinary(response); parseErr != nil {
						t.Fatal(parseErr)
					}
					c, err = (minecraft.Dialer{ClientData: login.ClientData{Nonce: data.Nonce}}).DialContextNetwork(ctx, minecraft.NetherNet{Signaling: d}, strconv.FormatUint(id, 10))
					break found
				}
			}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	var remote *minecraft.Conn
	select {
	case remote = <-accepted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	t.Cleanup(func() { remote.Close() })
	if err := <-spawned; err != nil {
		t.Fatal(err)
	}
	return l, c, remote
}

func TestNetherNetTransferReachesClientAndStopsSession(t *testing.T) {
	for _, fast := range []bool{false, true} {
		label := "reload"
		if fast {
			label = "fast"
		}
		t.Run(label+" HTTP repeated-port login address", func(t *testing.T) { testNetherNetTransfer(t, false, !fast) })
		t.Run(label+" LAN network-ID login address", func(t *testing.T) { testNetherNetTransfer(t, true, !fast) })
	}
}
func testNetherNetTransfer(t *testing.T, lan, reload bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	frontend, client, frontConn := transferTestPeer(t, ctx, lan)
	_, upstream, server := transferTestPeer(t, ctx, false)
	s := NewSession(ctx, ProxySettings{}, nil, nil, true)
	s.listener = frontend
	s.Client, s.Server = frontConn, upstream
	done := make(chan error, 1)
	go func() { done <- s.runProxyLoops() }()
	target := &packet.Transfer{Address: "upstream.example", Port: 19133, ReloadWorld: reload}
	if err := server.WritePacket(target); err != nil {
		t.Fatal(err)
	}
	if err := server.Flush(); err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		pk, err := client.ReadPacket()
		if err != nil {
			t.Fatal("transfer not delivered:", err)
		}
		if transfer, ok := pk.(*packet.Transfer); ok {
			_, port, err := net.SplitHostPort(frontend.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			address := net.JoinHostPort(transfer.Address, port)
			if transfer.Port != uint16(frontend.Addr().(*net.TCPAddr).Port) || net.ParseIP(transfer.Address) == nil || transfer.ReloadWorld != reload {
				t.Fatalf("invalid proxy transfer: %#v (%s)", transfer, address)
			}
			break
		}
	}
	_ = client.Close()
	select {
	case err := <-done:
		var transfer *errTransfer
		if !errors.As(err, &transfer) || transfer.transfer.Address != target.Address || transfer.transfer.Port != target.Port {
			t.Fatalf("upstream reconnect lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("transfer left forwarding goroutines blocked")
	}
}

func TestSessionEarlyFailureReleasesListener(t *testing.T) {
	t.Chdir(t.TempDir())
	messages.SetEventHandler(func(any) error { return nil })
	defer messages.SetEventHandler(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	listener, _, server := transferTestPeer(t, ctx, false)
	address := listener.Addr().String()
	s := NewSession(ctx, ProxySettings{}, nil, &connectinfo.ConnectInfo{Value: filepath.Join(t.TempDir(), "missing.pcap2")}, true)
	s.Client, s.listener = server, listener
	if err := s.Run(); err == nil {
		t.Fatal("missing replay unexpectedly succeeded")
	}
	replacement, err := (minecraft.DirectNetherNet{DisableLANDiscovery: true}).Listen(address)
	if err != nil {
		t.Fatal("failed join leaked listener:", err)
	}
	replacement.Close()
	select {
	case <-server.Context().Done():
	default:
		t.Fatal("failed join leaked client")
	}
}
