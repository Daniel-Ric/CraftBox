package proxy

import (
	"errors"
	"net"
)

func (s *Session) endpointError(err error, client bool) error {
	if err != nil && client && s.Client != nil {
		if ctx := s.Client.Context(); ctx != nil && ctx.Err() != nil {
			return errors.Join(net.ErrClosed, err)
		}
	}
	return proxyEndpointError(err, client)
}
