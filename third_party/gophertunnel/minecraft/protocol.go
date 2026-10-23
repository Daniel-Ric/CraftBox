package minecraft

import (
	"io"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Protocol interface {
	ID() int32

	Ver() string

	Packets(listener bool) packet.Pool

	NewReader(r ByteReader, shieldID int32, enableLimits bool) protocol.IO

	NewWriter(w ByteWriter, shieldID int32) protocol.IO

	ConvertToLatest(pk packet.Packet, conn *Conn) []packet.Packet

	ConvertFromLatest(pk packet.Packet, conn *Conn) []packet.Packet
}

type ByteReader interface {
	io.Reader
	io.ByteReader
}

type ByteWriter interface {
	io.Writer
	io.ByteWriter
}

type proto struct{}

func (proto) ID() int32     { return protocol.CurrentProtocol }
func (p proto) Ver() string { return protocol.CurrentVersion }
func (p proto) Packets(listener bool) packet.Pool {
	if listener {
		return packet.NewClientPool()
	}
	return packet.NewServerPool()
}
func (p proto) NewReader(r ByteReader, shieldID int32, enableLimits bool) protocol.IO {
	return protocol.NewReader(r, shieldID, enableLimits)
}
func (p proto) NewWriter(w ByteWriter, shieldID int32) protocol.IO {
	return protocol.NewWriter(w, shieldID)
}
func (p proto) ConvertToLatest(pk packet.Packet, _ *Conn) []packet.Packet { return []packet.Packet{pk} }
func (p proto) ConvertFromLatest(pk packet.Packet, _ *Conn) []packet.Packet {
	return []packet.Packet{pk}
}

var DefaultProtocol = proto{}
