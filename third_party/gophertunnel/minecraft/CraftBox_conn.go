package minecraft

import (
	"context"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"net"
	"time"
)

type IConn interface {
	Authenticated() bool
	ChunkRadius() int
	ClientCacheEnabled() bool
	ClientData() login.ClientData
	Close() error
	DoSpawn() error
	DoSpawnContext(ctx context.Context) error
	DoSpawnTimeout(timeout time.Duration) error
	Flush() error
	GameData() GameData
	IdentityData() login.IdentityData
	Latency() time.Duration
	LocalAddr() net.Addr
	ReadPacket() (pk packet.Packet, err error)
	ReadPacketWithTime() (pk packet.Packet, receivedAt time.Time, err error)
	RemoteAddr() net.Addr
	ResourcePacks() []*resource.Pack
	SetDeadline(t time.Time) error
	SetGameData(data GameData)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(time.Time) error
	StartGame(data GameData) error
	StartGameContext(ctx context.Context, data GameData) error
	StartGameTimeout(data GameData, timeout time.Duration) error
	WritePacket(pk packet.Packet) error
	Expect(...uint32)
	Context() context.Context
	SetLoggedIn()
	ShieldID() int32
}
