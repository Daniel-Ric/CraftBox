package minecraft

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestTerrainReadyWaitsForClientChunkRadius(t *testing.T) {
	c := newConn(&spawnSinkConn{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started, ready, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var mu sync.Mutex
	var writes []uint32
	c.packetFunc = func(h packet.Header, _ []byte, _, _ net.Addr) {
		mu.Lock()
		writes = append(writes, h.PacketID)
		mu.Unlock()
		if h.PacketID == packet.IDItemRegistry {
			close(started)
		}
	}
	go func() {
		done <- c.StartGameContextWithReady(ctx, GameData{WorldName: "Hoplite terrain barrier", EntityRuntimeID: 3, EntityUniqueID: 3}, func() {
			_ = c.WritePacket(&packet.LevelChunk{})
			_ = c.Flush()
			close(ready)
		})
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("StartGame was not sent")
	}
	select {
	case <-ready:
		t.Fatal("terrain released before Minecraft requested chunks")
	case <-time.After(30 * time.Millisecond):
	}
	if err := c.handleRequestChunkRadius(&packet.RequestChunkRadius{ChunkRadius: 96, MaxChunkRadius: 28}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("terrain still blocked after radius/spawn response")
	}
	mu.Lock()
	order := append([]uint32(nil), writes...)
	mu.Unlock()
	index := func(id uint32) int {
		for i, v := range order {
			if v == id {
				return i
			}
		}
		return -1
	}
	if index(packet.IDChunkRadiusUpdated) < 0 || index(packet.IDPlayStatus) < index(packet.IDChunkRadiusUpdated) || index(packet.IDLevelChunk) < index(packet.IDPlayStatus) {
		t.Fatalf("terrain preceded readiness packets: %v", order)
	}
	select {
	case err := <-done:
		t.Fatalf("full initialisation finished too early: %v", err)
	default:
	}
	if err := c.handleSetLocalPlayerAsInitialised(&packet.SetLocalPlayerAsInitialised{EntityRuntimeID: 3}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("client initialisation never completed")
	}
}

func TestTerrainReadyLoopbackPreservesSpawnBeforeBootstrapChunk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
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
		serverDone <- server.StartGameContextWithReady(ctx, GameData{WorldName: "Single bootstrap chunk", EntityRuntimeID: 3, EntityUniqueID: 3}, func() {
			_ = server.WritePacket(&packet.LevelChunk{})
			_ = server.Flush()
		})
	}()
	var spawnSeen, terrainTooEarly atomic.Bool
	client, err := (Dialer{EarlyConnHandler: func(c *Conn) { c.SetPlayerInitialisationDeferred(true) }, PacketFunc: func(h packet.Header, body []byte, _, _ net.Addr) {
		if h.PacketID == packet.IDPlayStatus && len(body) >= 4 && binary.BigEndian.Uint32(body) == uint32(packet.PlayStatusPlayerSpawn) {
			spawnSeen.Store(true)
		}
		if h.PacketID == packet.IDLevelChunk && !spawnSeen.Load() {
			terrainTooEarly.Store(true)
		}
	}}).DialContext(ctx, "nethernet", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		pk, err := client.ReadPacket()
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := pk.(*packet.LevelChunk); ok {
			break
		}
	}
	if terrainTooEarly.Load() {
		t.Fatal("bootstrap chunk reached the client before its spawn response")
	}
	select {
	case err := <-serverDone:
		t.Fatalf("premature initialisation: %v", err)
	default:
	}
	if err := client.WritePacket(&packet.SetLocalPlayerAsInitialised{EntityRuntimeID: 3}); err != nil {
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
		t.Fatal("final client initialisation did not complete")
	}
}

type terrainGateConn struct {
	spawnSinkConn
	block                             atomic.Bool
	first, entered, release           chan struct{}
	firstOnce, enterOnce, releaseOnce sync.Once
}

func (c *terrainGateConn) Write(data []byte) (int, error) {
	if c.block.Load() {
		c.enterOnce.Do(func() { close(c.entered) })
		<-c.release
	}
	n, err := c.spawnSinkConn.Write(data)
	c.firstOnce.Do(func() { close(c.first) })
	return n, err
}
func (c *terrainGateConn) Close() error { c.releaseOnce.Do(func() { close(c.release) }); return nil }
func TestTerrainReadyDoesNotEscapeCancellationDuringFlush(t *testing.T) {
	sink := &terrainGateConn{first: make(chan struct{}), entered: make(chan struct{}), release: make(chan struct{})}
	c := newConn(sink, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
	defer func() { _ = sink.Close(); _ = c.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done, handled := make(chan error, 1), make(chan error, 1)
	var called atomic.Bool
	go func() {
		done <- c.StartGameContextWithReady(ctx, GameData{WorldName: "Flush race", EntityRuntimeID: 3}, func() { called.Store(true) })
	}()
	select {
	case <-sink.first:
	case <-time.After(time.Second):
		t.Fatal("StartGame never flushed")
	}
	sink.block.Store(true)
	go func() { handled <- c.handleRequestChunkRadius(&packet.RequestChunkRadius{ChunkRadius: 16}) }()
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("readiness Flush did not block")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("StartGame cancellation was blocked by Flush")
	}
	_ = sink.Close()
	select {
	case err := <-handled:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("readiness handler did not return")
	}
	if called.Load() {
		t.Fatal("terrain callback escaped the cancelled StartGame during Flush")
	}
}

func TestTerrainReadyClearedOnCancelledStartGame(t *testing.T) {
	c := newConn(&spawnSinkConn{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, done := make(chan struct{}), make(chan error, 1)
	var called atomic.Bool
	c.packetFunc = func(h packet.Header, _ []byte, _, _ net.Addr) {
		if h.PacketID == packet.IDItemRegistry {
			close(started)
		}
	}
	go func() {
		done <- c.StartGameContextWithReady(ctx, GameData{WorldName: "Cancelled", EntityRuntimeID: 3}, func() { called.Store(true) })
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("StartGame not sent")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancelled StartGame")
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation ignored")
	}
	if err := c.handleRequestChunkRadius(&packet.RequestChunkRadius{ChunkRadius: 16}); err != nil {
		t.Fatal(err)
	}
	if called.Load() {
		t.Fatal("cancelled terrain callback was invoked")
	}
}
