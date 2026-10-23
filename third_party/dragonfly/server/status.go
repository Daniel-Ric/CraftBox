package server

import (
	"github.com/sandertv/gophertunnel/minecraft"
)

type statusProvider struct {
	name string
}

func (s statusProvider) ServerStatus(playerCount, maxPlayers int) minecraft.ServerStatus {
	return minecraft.ServerStatus{
		ServerName:  s.name,
		PlayerCount: playerCount,
		MaxPlayers:  maxPlayers,
	}
}
