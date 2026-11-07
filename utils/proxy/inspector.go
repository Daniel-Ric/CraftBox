package proxy

import (
	"context"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"net"
	"time"
)

type extraHandlersKey struct{}

func WithExtraHandlers(ctx context.Context, handlers ...func() *Handler) context.Context {
	return context.WithValue(ctx, extraHandlersKey{}, handlers)
}
func (s *Session) SessionContext() context.Context { return s.ctx }
func (s *Session) observeWire(header packet.Header, payload []byte, src, dst net.Addr, clientLeg bool, at time.Time) {
	if src == nil {
		return
	}
	local := ""
	if clientLeg {
		if s.Client != nil {
			local = s.Client.LocalAddr().String()
		} else if s.listener != nil {
			local = s.listener.Addr().String()
		}
	} else if s.Server != nil {
		local = s.Server.LocalAddr().String()
	}
	outgoing := local != "" && src.String() == local
	toServer := outgoing
	if clientLeg {
		toServer = !outgoing
	}
	stage := "received"
	if outgoing {
		stage = "sent"
	}
	shield := int32(0)
	if s.Server != nil {
		shield = s.Server.ShieldID()
	}
	sourceAddress, destinationAddress, connectionLeg := src.String(), "", "server"
	if dst != nil {
		destinationAddress = dst.String()
	}
	if clientLeg {
		connectionLeg = "client"
	}
	for _, h := range s.handlers {
		if h.PacketWire != nil {
			h.PacketWire(s, header, payload, toServer, stage, at, shield, sourceAddress, destinationAddress, connectionLeg)
		}
	}
}
func (h Handlers) Intercept(s *Session, pk packet.Packet, toServer bool) (packet.Packet, error) {
	for _, handler := range h {
		if handler.PacketControl != nil {
			var e error
			pk, e = handler.PacketControl(s, pk, toServer)
			if e != nil || pk == nil {
				return pk, e
			}
		}
	}
	return pk, nil
}
