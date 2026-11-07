package proxy

import (
	"context"
	"errors"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"net"
	"sync"
	"testing"
	"time"
)

type initialisationTestConn struct {
	minecraft.IConn
	ctx                    context.Context
	started, release       chan struct{}
	mu                     sync.Mutex
	writes                 []packet.Packet
	writeError, spawnError error
}

func (c *initialisationTestConn) Context() context.Context             { return c.ctx }
func (c *initialisationTestConn) Close() error                         { return nil }
func (c *initialisationTestConn) DoSpawnContext(context.Context) error { return nil }
func (c *initialisationTestConn) ReadPacketWithTime() (packet.Packet, time.Time, error) {
	<-c.ctx.Done()
	return nil, time.Time{}, net.ErrClosed
}
func (c *initialisationTestConn) StartGameContextWithReady(ctx context.Context, _ minecraft.GameData, ready func()) error {
	ready()
	close(c.started)
	select {
	case <-c.release:
		return c.spawnError
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
func (c *initialisationTestConn) WritePacket(pk packet.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.writeError != nil {
		return c.writeError
	}
	c.writes = append(c.writes, pk)
	return nil
}
func (c *initialisationTestConn) Flush() error { return nil }
func (c *initialisationTestConn) initialisations() []uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ids []uint64
	for _, pk := range c.writes {
		if p, ok := pk.(*packet.SetLocalPlayerAsInitialised); ok {
			ids = append(ids, p.EntityRuntimeID)
		}
	}
	return ids
}

func TestUpstreamInitialisationWaitsForMinecraft(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sessionCtx, stop := context.WithCancelCause(ctx)
	defer stop(context.Canceled)
	server := &initialisationTestConn{ctx: sessionCtx}
	client := &initialisationTestConn{ctx: sessionCtx, started: make(chan struct{}), release: make(chan struct{})}
	s := &Session{ctx: sessionCtx, cancelCtx: stop, Server: server, Client: client, withClient: true,
		handlers: Handlers{&Handler{OnConnect: func(*Session) bool { return true }}}}
	done := make(chan error, 1)
	go func() { done <- s.runGame(minecraft.GameData{EntityRuntimeID: 6668}) }()
	select {
	case <-client.started:
	case <-ctx.Done():
		t.Fatal("downstream StartGame did not start")
	}
	if len(server.initialisations()) != 0 {
		t.Fatal("upstream initialised before Minecraft finished loading")
	}
	close(client.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("proxy did not complete")
	}
	ids := server.initialisations()
	if len(ids) != 1 || ids[0] != 6668 {
		t.Fatalf("upstream initialisations %v; want exactly runtime ID 6668 after downstream completion", ids)
	}
}

func TestUpstreamInitialisationPreservesFailure(t *testing.T) {
	for _, failure := range []string{"client spawn", "upstream write"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			sessionCtx, stop := context.WithCancelCause(ctx)
			defer stop(context.Canceled)
			want := errors.New(failure + " failed")
			server := &initialisationTestConn{ctx: sessionCtx}
			client := &initialisationTestConn{ctx: sessionCtx, started: make(chan struct{}), release: make(chan struct{})}
			if failure == "client spawn" {
				client.spawnError = want
			} else {
				server.writeError = want
			}
			close(client.release)
			s := &Session{ctx: sessionCtx, cancelCtx: stop, Server: server, Client: client, withClient: true}
			err := s.runGame(minecraft.GameData{EntityRuntimeID: 6668})
			if !errors.Is(err, want) {
				t.Fatalf("got %v; want %v", err, want)
			}
			if len(server.initialisations()) != 0 {
				t.Fatal("failed startup sent initialisation")
			}
		})
	}
}

func TestClientlessCaptureDoesNotDuplicateInitialisation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sessionCtx, stop := context.WithCancelCause(ctx)
	defer stop(context.Canceled)
	server := &initialisationTestConn{ctx: sessionCtx}
	s := &Session{ctx: sessionCtx, cancelCtx: stop, Server: server,
		handlers: Handlers{&Handler{OnConnect: func(*Session) bool { return true }}}}
	if err := s.runGame(minecraft.GameData{EntityRuntimeID: 6668}); err != nil {
		t.Fatal(err)
	}
	if len(server.initialisations()) != 0 {
		t.Fatal("clientless capture duplicated the connection's automatic initialisation")
	}
}
