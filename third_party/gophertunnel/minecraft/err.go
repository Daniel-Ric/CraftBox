package minecraft

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var errBufferTooSmall = errors.New("a message sent was larger than the buffer used to receive the message into")

func (conn *Conn) wrap(err error, op string) error {
	if err == nil {
		return nil
	}
	return &net.OpError{
		Op:     op,
		Net:    "minecraft",
		Source: conn.LocalAddr(),
		Addr:   conn.RemoteAddr(),
		Err:    err,
	}
}

type DisconnectError string

func (d DisconnectError) Error() string {
	return string(d)
}

func disconnectPacketError(pk *packet.Disconnect) DisconnectError {
	message := strings.TrimSpace(pk.Message)
	if message == "" {
		message = strings.TrimSpace(pk.FilteredMessage)
	}
	if message == "" {
		message = "The server closed the connection without providing a reason"
		if pk.Reason == packet.DisconnectReasonKicked {
			message = "The server removed you from the world. No reason was provided"
		}
	}
	return DisconnectError(fmt.Sprintf("%s (disconnect code %d)", message, pk.Reason))
}
