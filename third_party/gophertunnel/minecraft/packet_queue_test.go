package minecraft

import (
	"bytes"
	"context"
	"errors"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"io"
	"log/slog"
	"net"
	"runtime"
	"testing"
	"time"
)

func TestConcurrentInboundPacketsKeepFIFOAndOwnTheirBuffers(t *testing.T) {
	for _, mode := range []string{"packets", "bytes", "reader"} {
		t.Run(mode, func(t *testing.T) {
			local, remote := net.Pipe()
			defer local.Close()
			defer remote.Close()
			c := newConn(local, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
			c.pool = proto{}.Packets(true)
			c.loggedIn = true
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
			const count = 3000
			sent := make(chan error, 1)
			go func() {
				var encoded bytes.Buffer
				for i := 0; i < count; i++ {
					encoded.Reset()
					header := packet.Header{PacketID: packet.IDNetworkStackLatency}
					_ = header.Write(&encoded)
					pk := packet.NetworkStackLatency{Timestamp: int64(i)}
					pk.Marshal(protocol.NewWriter(&encoded, 0))
					if err := c.receive(encoded.Bytes()); err != nil {
						sent <- err
						return
					}
				}
				sent <- nil
			}()
			for i := 0; i < count; i++ {
				var pk *packet.NetworkStackLatency
				if mode == "packets" {
					received, err := c.ReadPacket()
					if err != nil {
						t.Fatal(err)
					}
					pk = received.(*packet.NetworkStackLatency)
				} else {
					var raw []byte
					if mode == "bytes" {
						var err error
						raw, err = c.ReadBytes()
						if err != nil {
							t.Fatal(err)
						}
					} else {
						raw = make([]byte, 64)
						n, err := c.Read(raw)
						if err != nil {
							t.Fatal(err)
						}
						raw = raw[:n]
					}
					buffer := bytes.NewBuffer(raw)
					var header packet.Header
					if err := header.Read(buffer); err != nil {
						t.Fatal(err)
					}
					if header.PacketID != packet.IDNetworkStackLatency {
						t.Fatal("borrowed packet buffer changed")
					}
					pk = &packet.NetworkStackLatency{}
					pk.Marshal(protocol.NewReader(buffer, 0, true))
				}
				if pk.Timestamp != int64(i) {
					t.Fatalf("reordered inbound packet: got=%d want=%d", pk.Timestamp, i)
				}
			}
			if err := <-sent; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestQueuedPacketDoesNotBypassCanceledRead(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("test canceled connection")
	c := &Conn{ctx: ctx, packetReady: make(chan struct{}, 1), conn: local}
	c.deferPacket(&packetData{full: []byte{1}, payload: bytes.NewBuffer(nil), owned: true})
	cancel(cause)
	if _, err := c.nextPacketData("read"); !errors.Is(err, cause) {
		t.Fatalf("queued data bypassed cancellation: %v", err)
	}
}

func TestQueuedPacketDoesNotBypassExpiredReadDeadline(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	deadline := make(chan time.Time, 1)
	deadline <- time.Now()
	c := &Conn{ctx: context.Background(), packetReady: make(chan struct{}, 1), readDeadline: deadline, conn: local}
	c.deferPacket(&packetData{full: []byte{1}, payload: bytes.NewBuffer(nil), owned: true})
	if _, err := c.nextPacketData("read"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued data bypassed deadline: %v", err)
	}
}

func TestCoalescedReadinessWakesBothWaitingReaders(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(net.ErrClosed)
	c := &Conn{ctx: ctx, packetReady: make(chan struct{}, 1)}
	started := make(chan struct{}, 2)
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { started <- struct{}{}; _, err := c.ReadBytes(); done <- err }()
	}
	<-started
	<-started
	// Both readers enter their wait before a burst coalesces into one wake-up.
	time.Sleep(time.Millisecond)
	for i := 0; i < 2; i++ {
		c.deferPacket(&packetData{full: []byte{byte(i)}, payload: bytes.NewBuffer(nil), owned: true})
	}
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("reader stranded despite queued packet")
		}
	}
}
