package proxy

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type preplayFailureConn struct{ minecraft.IConn }

func (*preplayFailureConn) LocalAddr() net.Addr { return &net.UDPAddr{Port: 2} }
func (*preplayFailureConn) ShieldID() int32     { return 0 }

func TestPrePlayHandlerFailureCancelsSession(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	failure := errors.New("initialise world block registry: unresolved trait minecraft:future")
	s := &Session{ctx: ctx, cancelCtx: cancel, Server: &preplayFailureConn{}, clientAddr: &net.UDPAddr{Port: 1}, handlers: Handlers{&Handler{
		PacketCallback: func(_ *Session, _ packet.Packet, _ bool, _ time.Time, _ bool) (packet.Packet, error) {
			return nil, failure
		},
	}}}
	var payload bytes.Buffer
	(&packet.GameRulesChanged{}).Marshal(protocol.NewWriter(&payload, 0))
	s.packetFunc(packet.Header{PacketID: packet.IDGameRulesChanged}, payload.Bytes(), s.Server.LocalAddr(), s.clientAddr, time.Now())
	if !errors.Is(context.Cause(ctx), failure) {
		t.Fatalf("session continued after failed initialisation: %v", context.Cause(ctx))
	}
}
