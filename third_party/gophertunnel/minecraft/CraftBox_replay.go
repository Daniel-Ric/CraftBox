package minecraft

import (
	"bytes"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type CapturedPacketData struct {
	Header  *packet.Header
	Payload *bytes.Buffer
}

func ParseData(data []byte) (*CapturedPacketData, error) {
	payload := bytes.NewBuffer(bytes.Clone(data))
	header := &packet.Header{}
	if err := header.Read(payload); err != nil {
		return nil, fmt.Errorf("read captured packet header: %w", err)
	}
	return &CapturedPacketData{Header: header, Payload: payload}, nil
}

func (p *CapturedPacketData) Decode(pool packet.Pool, proto Protocol, close func() error, rejectUnknown, rejectInvalid bool, shieldID int32) ([]packet.Packet, error) {
	if proto == nil {
		proto = DefaultProtocol
	}
	if _, known := pool[p.Header.PacketID]; !known && rejectUnknown {
		if close != nil {
			_ = close()
		}
		return nil, unknownPacketError{id: p.Header.PacketID}
	}
	conn := &Conn{proto: proto, pool: pool}
	conn.shieldID.Store(shieldID)
	data := &packetData{h: p.Header, payload: p.Payload, owned: true}
	decoded, err := data.decode(conn)
	if err != nil && rejectInvalid {
		if close != nil {
			_ = close()
		}
		return nil, err
	}
	return decoded, err
}
