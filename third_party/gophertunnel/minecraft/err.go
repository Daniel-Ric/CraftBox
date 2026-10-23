package minecraft

import (
	"errors"
	"net"
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
