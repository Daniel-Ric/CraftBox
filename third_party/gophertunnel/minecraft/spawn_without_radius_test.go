package minecraft

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Exercise the normal receive/expect/deferred-queue path with Hoplite's actual
// observed ordering, without storing account/login data from the recording.
type spawnSinkConn struct{ bytes.Buffer }

func (*spawnSinkConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (*spawnSinkConn) Close() error                     { return nil }
func (*spawnSinkConn) LocalAddr() net.Addr              { return &net.UDPAddr{Port: 1} }
func (*spawnSinkConn) RemoteAddr() net.Addr             { return &net.UDPAddr{Port: 2} }
func (*spawnSinkConn) SetDeadline(time.Time) error      { return nil }
func (*spawnSinkConn) SetReadDeadline(time.Time) error  { return nil }
func (*spawnSinkConn) SetWriteDeadline(time.Time) error { return nil }

func TestHopliteSpawnWireOrdering(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(map[bool]string{false: "standalone", true: "proxy"}[deferred], func(t *testing.T) {
			c := newConn(&spawnSinkConn{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
			c.pool = proto{}.Packets(false)
			c.deferPlayerInitialisation = deferred
			defer c.Close()
			c.expect(packet.IDStartGame)
			var initialised int
			c.packetFunc = func(h packet.Header, _ []byte, _, _ net.Addr) {
				if h.PacketID == packet.IDSetLocalPlayerAsInitialised {
					initialised++
				}
			}
			receive := func(pk packet.Packet) {
				t.Helper()
				var b bytes.Buffer
				h := packet.Header{PacketID: pk.ID()}
				_ = h.Write(&b)
				pk.Marshal(protocol.NewWriter(&b, 0))
				if err := c.receive(b.Bytes()); err != nil {
					t.Fatal(err)
				}
			}
			receive(&packet.StartGame{EntityRuntimeID: 6668, EntityUniqueID: 6668, WorldName: "Hoplite ordering", PropertyData: map[string]any{}})
			receive(&packet.ItemRegistry{})
			receive(&packet.LevelChunk{SubChunkCount: 0, CacheEnabled: false, RawPayload: []byte{}})
			receive(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn})
			if !c.loggedIn {
				t.Fatal("wire login blocked despite PlayerSpawn")
			}
			if (deferred && initialised != 0) || (!deferred && initialised != 1) {
				t.Fatalf("initialisation count=%d deferred=%v", initialised, deferred)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := c.DoSpawnContext(ctx); err != nil {
				t.Fatal(err)
			}
			if err := c.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			pk, err := c.ReadPacket()
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := pk.(*packet.LevelChunk); !ok {
				t.Fatalf("early terrain lost/reordered: got %T", pk)
			}
			// A late radius response remains available to the ordered proxy relay.
			receive(&packet.ChunkRadiusUpdated{ChunkRadius: 8})
			pk, err = c.ReadPacket()
			if err != nil {
				t.Fatal(err)
			}
			if radius, ok := pk.(*packet.ChunkRadiusUpdated); !ok || radius.ChunkRadius != 8 {
				t.Fatalf("late server radius lost: %T", pk)
			}
		})
	}
}

func radiusSpawnConn(t *testing.T, deferred bool) *Conn {
	t.Helper()
	local, remote := net.Pipe()
	c := newConn(local, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), proto{}, 0, true)
	c.gameData.EntityRuntimeID = 6668
	c.deferPlayerInitialisation = deferred
	t.Cleanup(func() { c.bufferedSend = nil; _ = remote.Close(); _ = c.Close() })
	return c
}
func TestClientSpawnWithoutChunkRadiusUpdated(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(map[bool]string{false: "standalone", true: "proxy"}[deferred], func(t *testing.T) {
			c := radiusSpawnConn(t, deferred)
			if err := c.handleItemRegistry(&packet.ItemRegistry{}); err != nil {
				t.Fatal(err)
			}
			select {
			case <-c.spawn:
				t.Fatal("spawn finished before server PlayerSpawn")
			default:
			}
			// Hoplite/Geyser: ItemRegistry -> our radius request -> LevelChunk and
			// PlayerSpawn, without a separate ChunkRadiusUpdated response.
			if err := c.handlePlayStatus(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn}); err != nil {
				t.Fatal(err)
			}
			if !c.loggedIn {
				t.Fatal("server approved spawn but login still waits for optional radius acknowledgement")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := c.DoSpawnContext(ctx); err != nil {
				t.Fatal(err)
			}
			if c.ChunkRadius() != 16 {
				t.Fatalf("fallback radius %d, want requested 16", c.ChunkRadius())
			}
			want := 1
			if !deferred {
				want++
			}
			if len(c.bufferedSend) != want {
				t.Fatalf("wrote %d packets, want %d (deferred=%v)", len(c.bufferedSend), want, deferred)
			}
		})
	}
}
func TestClientSpawnStillUsesAcknowledgedRadiusAndRequiresPlayerSpawn(t *testing.T) {
	c := radiusSpawnConn(t, true)
	if err := c.handleItemRegistry(&packet.ItemRegistry{}); err != nil {
		t.Fatal(err)
	}
	if err := c.handleChunkRadiusUpdated(&packet.ChunkRadiusUpdated{ChunkRadius: 8}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.spawn:
		t.Fatal("radius ack alone completed spawn")
	default:
	}
	if err := c.handlePlayStatus(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn}); err != nil {
		t.Fatal(err)
	}
	if !c.loggedIn || c.ChunkRadius() != 8 {
		t.Fatalf("login=%v radius=%d; expected login with server radius8", c.loggedIn, c.ChunkRadius())
	}
}
func TestPlayerSpawnAloneDoesNotBypassGameData(t *testing.T) {
	c := radiusSpawnConn(t, true)
	if err := c.handlePlayStatus(&packet.PlayStatus{Status: packet.PlayStatusPlayerSpawn}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.spawn:
		t.Fatal("spawn accepted before world/item data")
	default:
	}
}
