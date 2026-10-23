package minecraft

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type ServerStatusProvider interface {
	ServerStatus(playerCount, maxPlayers int) ServerStatus
}

type ServerStatus struct {
	ServerName string

	ServerSubName string

	PlayerCount int

	MaxPlayers int

	GameType int
}

type ListenerStatusProvider struct {
	name string

	subName string
}

func NewStatusProvider(serverName, serverSubName string) ListenerStatusProvider {
	return ListenerStatusProvider{name: serverName, subName: serverSubName}
}

func (l ListenerStatusProvider) ServerStatus(playerCount, maxPlayers int) ServerStatus {
	return ServerStatus{
		ServerName:    l.name,
		ServerSubName: l.subName,
		PlayerCount:   playerCount,
		MaxPlayers:    maxPlayers,
	}
}

type ForeignStatusProvider struct {
	addr string

	mu     sync.Mutex
	status ServerStatus

	once   sync.Once
	closed chan struct{}
}

func NewForeignStatusProvider(addr string) (*ForeignStatusProvider, error) {
	if _, err := net.ResolveUDPAddr("udp", addr); err != nil {
		return nil, err
	}
	f := &ForeignStatusProvider{addr: addr, closed: make(chan struct{})}
	go f.update()
	return f, nil
}

func (f *ForeignStatusProvider) ServerStatus(int, int) ServerStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *ForeignStatusProvider) Close() error {
	f.once.Do(func() {
		close(f.closed)
	})
	return nil
}

func (f *ForeignStatusProvider) update() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			data, err := (DirectNetherNet{}).PingContext(ctx, f.addr)
			cancel()
			if err != nil {
				continue
			}
			f.mu.Lock()
			f.status = ParsePongData(data)
			f.mu.Unlock()
		case <-f.closed:
			return
		}
	}
}

func ParsePongData(pong []byte) ServerStatus {
	frag := splitPong(string(pong))
	if len(frag) < 8 {
		return ServerStatus{ServerName: "Invalid pong data"}
	}
	serverName := frag[1]
	serverSubName := frag[7]
	online, err := strconv.Atoi(frag[4])
	if err != nil {
		return ServerStatus{ServerName: "Invalid player count"}
	}
	max, err := strconv.Atoi(frag[5])
	if err != nil {
		return ServerStatus{ServerName: "Invalid max player count"}
	}
	status := ServerStatus{
		ServerName:    serverName,
		ServerSubName: serverSubName,
		PlayerCount:   online,
		MaxPlayers:    max,
	}

	if len(frag) > 8 {
		status.GameType = parseGameType(frag[8])
	}
	return status
}

func gameTypeName(gameType int) string {
	switch gameType {
	case packet.GameTypeCreative:
		return "Creative"
	case packet.GameTypeAdventure:
		return "Adventure"
	case packet.GameTypeSpectator:
		return "Spectator"
	}
	return "Survival"
}

func parseGameType(v string) int {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "creative":
		return packet.GameTypeCreative
	case "adventure":
		return packet.GameTypeAdventure
	case "spectator":
		return packet.GameTypeSpectator
	}
	return packet.GameTypeSurvival
}
