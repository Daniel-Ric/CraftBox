package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type SubChunkRequestHandler struct{}

func (*SubChunkRequestHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, _ Controllable) error {
	pk := p.(*packet.SubChunkRequest)
	if dimID, _ := world.DimensionID(tx.World().Dimension()); pk.Dimension != int32(dimID) {

		s.writePacket(&packet.SubChunk{
			Dimension:       pk.Dimension,
			Position:        pk.Position,
			CacheEnabled:    s.conn.ClientCacheEnabled(),
			SubChunkEntries: []protocol.SubChunkEntry{},
		})
		return nil
	}
	s.ViewSubChunks(world.SubChunkPos(pk.Position), pk.Offsets, tx)
	return nil
}
