package proxy

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type subchunkRequestClient struct {
	*initialisationTestConn
	request *packet.SubChunkRequest
}

func (c *subchunkRequestClient) ReadPacketWithTime() (packet.Packet, time.Time, error) {
	if c.request == nil {
		return nil, time.Time{}, net.ErrClosed
	}
	pk := c.request
	c.request = nil
	return pk, time.Now(), nil
}

func TestMinecraftSubChunkRequestsAreForwardedUnchanged(t *testing.T) {
	ctx := context.Background()
	request := &packet.SubChunkRequest{Dimension: 0, Position: protocol.SubChunkPos{-6245, 6, -6245}, Offsets: []protocol.SubChunkOffset{{0, 0, 0}, {1, 0, 0}}}
	server := &initialisationTestConn{ctx: ctx}
	client := &subchunkRequestClient{initialisationTestConn: &initialisationTestConn{ctx: ctx}, request: request}
	s := &Session{ctx: ctx, Client: client, Server: server}
	if err := s.proxyLoop(ctx, true); !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
	if len(server.writes) != 1 || server.writes[0] != request {
		t.Fatal("Minecraft terrain request was dropped, duplicated or modified")
	}
}
