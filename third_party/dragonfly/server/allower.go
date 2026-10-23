package server

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"net"
)

type Allower interface {
	Allow(addr net.Addr, d login.IdentityData, c login.ClientData) (string, bool)
}

type allower struct{}

func (allower) Allow(net.Addr, login.IdentityData, login.ClientData) (string, bool) {
	return "", true
}
