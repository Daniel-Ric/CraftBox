package minecraft

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestInvalidClientPacketRetainsDecodeCause(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	c := newConn(local, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
	c.pool = proto{}.Packets(true)
	c.disconnectOnInvalidPacket = true
	var payload bytes.Buffer
	h := packet.Header{PacketID: packet.IDText}
	_ = h.Write(&payload)
	parsed, err := parseData(payload.Bytes(), c)
	if err != nil {
		t.Fatal(err)
	}
	c.deferPacket(parsed)
	_, err = c.ReadPacket()
	if err == nil || !strings.Contains(err.Error(), "decode packet *packet.Text") {
		t.Fatalf("lost original malformed-packet cause: %v", err)
	}
}

func TestBatchFailureRetainsDecodeCause(t *testing.T) {
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	c := newConn(local, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, false)
	done := make(chan struct{})
	go func() { listenConn(c, make(chan struct{}), make(chan struct{}), func(error) {}); close(done) }()
	if _, err := remote.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("decoder did not terminate")
	}
	cause := context.Cause(c.Context())
	if cause == nil || !strings.Contains(cause.Error(), "invalid header") {
		t.Fatalf("lost original batch cause: %v", cause)
	}
}
