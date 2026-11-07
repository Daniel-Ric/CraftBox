package worlds

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

type subchunkCaptureConn struct {
	minecraft.IConn
	written []packet.Packet
}

func (c *subchunkCaptureConn) WritePacket(pk packet.Packet) error {
	c.written = append(c.written, pk)
	return nil
}

// This announcement comes from the cache-disabled CrabSMP capture at
// 2026-11-05 08:20:47. It advertises streamed subchunks rather than full terrain.
func TestRecordedChunkAnnouncementUsesMinecraftRequests(t *testing.T) {
	for _, withClient := range []bool{true, false} {
		name := "proxy"
		if !withClient {
			name = "clientless"
		}
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/crabsmp-subchunk-announcement.json")
			if err != nil {
				t.Fatal(err)
			}
			var announcement packet.LevelChunk
			if err := json.Unmarshal(data, &announcement); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			state, err := worldstate.New(ctx, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			state.SetDimension(world.Overworld)
			server := &subchunkCaptureConn{}
			session := &proxy.Session{Server: server}
			if withClient {
				session.Client = &subchunkCaptureConn{}
			}
			blocks := world.NewBlockRegistry()
			blocks.Finalize()
			handler := &worldsHandler{worldState: state, session: session,
				serverState: serverState{blocks: blocks}, log: logrus.NewEntry(logrus.New())}
			if err := handler.handleLevelChunk(&announcement, time.Now()); err != nil {
				t.Fatal(err)
			}
			want := 1
			if withClient {
				want = 0
			}
			if len(server.written) != want {
				t.Fatalf("capture injected %d upstream packets, want %d", len(server.written), want)
			}
			if !withClient {
				request, ok := server.written[0].(*packet.SubChunkRequest)
				if !ok || request.Position[0] != announcement.Position.X() || request.Position[2] != announcement.Position.Z() || len(request.Offsets) == 0 {
					t.Fatal("clientless capture lost its terrain request")
				}
			}
			if _, found, err := state.LoadChunk(world.ChunkPos(announcement.Position)); err != nil || !found {
				t.Fatalf("chunk announcement was not retained: found=%t err=%v", found, err)
			}
		})
	}
}
