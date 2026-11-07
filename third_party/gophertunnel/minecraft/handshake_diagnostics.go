package minecraft

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// HandshakeDiagnostics reports only queued packet IDs and counts. It never
// includes packet bodies, login data, content keys, or resource pack URLs.
func (conn *Conn) HandshakeDiagnostics() string {
	deferred := "locked"
	if conn.deferredPacketMu.TryLock() {
		ids := make(map[uint32]int)
		for _, pk := range conn.deferredPackets {
			ids[pk.h.PacketID]++
		}
		conn.deferredPacketMu.Unlock()
		deferred = handshakePacketCounts(ids)
	}
	buffered := "locked"
	if conn.sendMu.TryLock() {
		ids := make(map[uint32]int)
		for _, data := range conn.bufferedSend {
			header := &packet.Header{}
			if header.Read(bytes.NewBuffer(data)) == nil {
				ids[header.PacketID]++
			}
		}
		conn.sendMu.Unlock()
		buffered = handshakePacketCounts(ids)
	}
	return fmt.Sprintf("Buffered outbound packet IDs: %s\nDeferred inbound packet IDs: %s\n", buffered, deferred)
}

func handshakePacketCounts(counts map[uint32]int) string {
	ids := make([]uint32, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	var parts []string
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d x%d", id, counts[id]))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}
