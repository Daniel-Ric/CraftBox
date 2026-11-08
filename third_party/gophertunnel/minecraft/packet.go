package minecraft

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"time"
)

type packetData struct {
	decoded    []packet.Packet
	receivedAt time.Time
	h          *packet.Header
	full       []byte
	payload    *bytes.Buffer

	owned bool
}

func parseData(data []byte, conn *Conn) (*packetData, error) {
	receivedAt := time.Now()
	buf := bytes.NewBuffer(data)
	header := &packet.Header{}
	if err := header.Read(buf); err != nil {

		return nil, fmt.Errorf("read packet header: %w", err)
	}
	if conn.packetFunc != nil {

		conn.packetFunc(*header, bytes.Clone(buf.Bytes()), conn.RemoteAddr(), conn.LocalAddr())
	}
	return &packetData{h: header, full: data, payload: buf, receivedAt: receivedAt}, nil
}

func (p *packetData) ensureOwned() *packetData {
	if p.owned {
		return p
	}
	full := bytes.Clone(p.full)
	payloadOffset := len(p.full) - p.payload.Len()
	var payload []byte
	if payloadOffset < 0 {
		payload = bytes.Clone(p.payload.Bytes())
	} else {
		payload = full[payloadOffset:]
	}
	return &packetData{
		decoded:    p.decoded,
		receivedAt: p.receivedAt,
		h:          p.h,
		full:       full,
		payload:    bytes.NewBuffer(payload),
		owned:      true,
	}
}

type unknownPacketError struct {
	id uint32
}

func (err unknownPacketError) Error() string {
	return fmt.Sprintf("unexpected packet (ID=%v)", err.id)
}

func (p *packetData) decode(conn *Conn) (pks []packet.Packet, err error) {
	if p.decoded != nil {
		return p.decoded, nil
	}

	pkFunc, ok := conn.pool[p.h.PacketID]
	var pk packet.Packet
	if !ok {

		pk = &packet.Unknown{PacketID: p.h.PacketID}
		if conn.disconnectOnUnknownPacket {
			cause := unknownPacketError{id: p.h.PacketID}
			_ = conn.close(cause)
			return nil, cause
		}
	} else {
		pk = pkFunc()
	}

	defer func() {
		if recoveredErr := recover(); recoveredErr != nil {
			err = fmt.Errorf("decode packet %T: %w", pk, recoveredErr.(error))
		}
		if err != nil && !errors.Is(err, unknownPacketError{}) && conn.disconnectOnInvalidPacket {
			_ = conn.close(err)
		}
	}()

	payloadBuf := p.payload.Bytes()
	payload := bytes.NewReader(payloadBuf)
	r := conn.proto.NewReader(payload, conn.shieldID.Load(), conn.readerLimits)
	pk.Marshal(r)
	if payload.Len() != 0 {
		left := payloadBuf[len(payloadBuf)-payload.Len():]
		err = fmt.Errorf("decode packet %T: %v unread bytes left: 0x%x", pk, len(left), left)
	}
	if conn.disconnectOnInvalidPacket && err != nil {
		return nil, err
	}
	return conn.proto.ConvertToLatest(pk, conn), err
}
