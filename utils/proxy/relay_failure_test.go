package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type relayFailureReader struct {
	*initialisationTestConn
	packet    packet.Packet
	readError error
}

func (c *relayFailureReader) ReadPacketWithTime() (packet.Packet, time.Time, error) {
	if c.packet != nil {
		pk := c.packet
		c.packet = nil
		return pk, time.Now(), nil
	}
	return nil, time.Time{}, c.readError
}

func TestRelayFailureLabelsActualReadOrWriteEndpoint(t *testing.T) {
	for _, toServer := range []bool{true, false} {
		for _, writeFailure := range []bool{true, false} {
			ctx := context.Background()
			cause := errors.New("endpoint failure")
			reader := &relayFailureReader{initialisationTestConn: &initialisationTestConn{ctx: ctx}, readError: cause}
			writer := &initialisationTestConn{ctx: ctx, writeError: cause}
			if writeFailure {
				reader.packet = &packet.SubChunkRequest{}
			}
			s := &Session{ctx: ctx}
			if toServer {
				s.Client = reader
				s.Server = writer
			} else {
				s.Server = reader
				s.Client = writer
			}
			err := s.proxyLoop(ctx, toServer)
			want := "client"
			if toServer == writeFailure {
				want = "server"
			}
			var failure *ConnectionFailure
			if !errors.As(err, &failure) || failure.Leg != want || !errors.Is(err, cause) {
				t.Fatalf("toServer=%t writeFailure=%t: got %v (%+v), want %s", toServer, writeFailure, err, failure, want)
			}
		}
	}
}

func TestRelayHandlerFailureDoesNotBlameNetworkEndpoint(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(context.Canceled)
	cause := errors.New("internal packet processing failed")
	reader := &relayFailureReader{initialisationTestConn: &initialisationTestConn{ctx: ctx}, packet: &packet.SubChunkRequest{}}
	s := &Session{ctx: ctx, cancelCtx: cancel, Server: reader, handlers: Handlers{&Handler{PacketControl: func(*Session, packet.Packet, bool) (packet.Packet, error) { return nil, cause }}}}
	err := s.runProxyLoops()
	var failure *ConnectionFailure
	if !errors.Is(err, cause) || errors.As(err, &failure) {
		t.Fatalf("internal failure incorrectly classified as network failure: %v", err)
	}
}
