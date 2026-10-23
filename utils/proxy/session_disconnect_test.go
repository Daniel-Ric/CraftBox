package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type disconnectConn struct {
	minecraft.IConn
	readErr error
	closed  chan struct{}
	once    sync.Once
}

func (c *disconnectConn) ReadPacketWithTime() (packet.Packet, time.Time, error) {
	if c.readErr != nil {
		return nil, time.Time{}, c.readErr
	}
	<-c.closed
	return nil, time.Time{}, net.ErrClosed
}
func (c *disconnectConn) Close() error { c.once.Do(func() { close(c.closed) }); return nil }

func TestProxySessionRemoteChannelClosure(t *testing.T) {
	for _, label := range []string{"ReliableDataChannel", "UnreliableDataChannel"} {
		for _, fromClient := range []bool{true, false} {
			name := label + "/server"
			if fromClient {
				name = label + "/client"
			}
			t.Run(name, func(t *testing.T) {
				remoteErr := &net.OpError{Op: "read packet", Net: "minecraft", Err: errors.New("nethernet: data channel \"" + label + "\" closed by remote peer")}
				client := &disconnectConn{closed: make(chan struct{})}
				server := &disconnectConn{closed: make(chan struct{})}
				if fromClient {
					client.readErr = remoteErr
				} else {
					server.readErr = remoteErr
				}
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				s := NewSession(ctx, ProxySettings{}, nil, nil, true)
				s.Client, s.Server = client, server
				got := s.runProxyLoops()
				if fromClient && got != nil {
					t.Fatalf("client closure returned error: %v", got)
				}
				if !fromClient && !errors.Is(got, remoteErr) {
					t.Fatalf("server error hidden: %v", got)
				}
				for _, c := range []*disconnectConn{client, server} {
					select {
					case <-c.closed:
					default:
						t.Fatal("connection not closed")
					}
				}
			})
		}
	}
}

func TestProxySessionClientFailureRemainsVisible(t *testing.T) {
	failure := errors.New("nethernet: handle message: invalid segment")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s := NewSession(ctx, ProxySettings{}, nil, nil, true)
	s.Client = &disconnectConn{readErr: failure, closed: make(chan struct{})}
	s.Server = &disconnectConn{closed: make(chan struct{})}
	if got := s.runProxyLoops(); !errors.Is(got, failure) {
		t.Fatalf("client failure hidden: %v", got)
	}
}

func TestProxySessionCancellationUnblocksReads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewSession(ctx, ProxySettings{}, nil, nil, true)
	s.Client = &disconnectConn{closed: make(chan struct{})}
	s.Server = &disconnectConn{closed: make(chan struct{})}
	result := make(chan error, 1)
	go func() { result <- s.runProxyLoops() }()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled proxy left blocked reads")
	}
}

func TestClientSCTPClosureIsNormalButUpstreamClosureRemainsError(t *testing.T) {
	err := fmt.Errorf("read packet: %w", errors.New("nethernet: SCTP transport closed"))
	if !errors.Is(proxyEndpointError(err, true), net.ErrClosed) {
		t.Fatal("normal client quit still reported as an error")
	}
	if errors.Is(proxyEndpointError(err, false), net.ErrClosed) {
		t.Fatal("upstream failure was hidden")
	}
	if errors.Is(proxyEndpointError(errors.New("nethernet: SCTP transport failed"), true), net.ErrClosed) {
		t.Fatal("genuine transport failure was hidden")
	}
}

type writeDisconnectConn struct {
	minecraft.IConn
	err error
}

func (c *writeDisconnectConn) WritePacket(packet.Packet) error { return c.err }

func TestClientMapWritesTreatSCTPQuitAsClosed(t *testing.T) {
	s := &Session{Client: &writeDisconnectConn{err: errors.New("nethernet: SCTP transport closed")}}
	if err := s.ClientWritePacket(&packet.ClientBoundMapItemData{}); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("map update reported a normal client quit: %v", err)
	}
	s.Client = &writeDisconnectConn{}
	if err := s.ClientWritePacket(&packet.ClientBoundMapItemData{}); err != nil {
		t.Fatalf("successful write: %v", err)
	}
}
