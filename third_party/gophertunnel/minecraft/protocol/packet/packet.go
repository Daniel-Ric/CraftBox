package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"io"
)

type Packet interface {
	ID() uint32

	Marshal(io protocol.IO)
}

type Header struct {
	PacketID        uint32
	SenderSubClient byte
	TargetSubClient byte
}

func (header *Header) Write(w io.ByteWriter) error {
	return protocol.WriteVaruint32(w, header.PacketID|(uint32(header.SenderSubClient)<<10)|(uint32(header.TargetSubClient)<<12))
}

func (header *Header) Read(r io.ByteReader) error {
	var value uint32
	if err := protocol.Varuint32(r, &value); err != nil {
		return err
	}
	header.PacketID = value & 0x3FF
	header.SenderSubClient = byte((value >> 10) & 0x3)
	header.TargetSubClient = byte((value >> 12) & 0x3)
	return nil
}
