package minecraft

import (
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type PrePlayPacketHandler func(*Conn, packet.Packet, time.Time) (bool, error)

type ResourcePackHandler interface {
	OnResourcePacksInfo(*packet.ResourcePacksInfo) error
	OnResourcePackClientResponse(*packet.ResourcePackClientResponse) error
	OnResourcePackDataInfo(*packet.ResourcePackDataInfo) error
	OnResourcePackChunkRequest(*packet.ResourcePackChunkRequest) error
	OnResourcePackChunkData(*packet.ResourcePackChunkData) error
	OnResourcePackStack(*packet.ResourcePackStack) error
	GetResourcePacksInfo(bool) *packet.ResourcePacksInfo
	ResourcePacks() []*resource.Pack
}

func (conn *Conn) Expect(ids ...uint32)      { conn.expect(ids...) }
func (conn *Conn) SetLoggedIn()              { conn.loggedIn = true }
func (conn *Conn) ShieldID() int32           { return conn.shieldID.Load() }
func (conn *Conn) SetGameData(data GameData) { conn.gameData = data }
func (conn *Conn) SetPrePlayPacketHandler(handler PrePlayPacketHandler) {
	conn.prePlayPacketHandler = handler
}
func (conn *Conn) ReadPacketWithTime() (packet.Packet, time.Time, error) {
	pk, err := conn.ReadPacket()
	return pk, conn.lastReadTime, err
}

func (listener *Listener) AcceptMinecraft() (*Conn, error) {
	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}
	return conn.(*Conn), nil
}
