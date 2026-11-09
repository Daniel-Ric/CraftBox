package messages

import (
	"image"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

type UIState uint8

type EventIngameFeedback struct{ Enabled bool }

const (
	UIStateMain UIState = iota + 1
	UIStateFinished
)

type ConnectState uint8

const (
	ConnectStateBegin ConnectState = iota + 1
	ConnectStateListening
	ConnectStateServerConnecting
	ConnectStateReceivingResources
	ConnectStateEstablished
	ConnectStateDone
)

type EventSetValue struct {
	Name  string
	Value string
}

type EventSetUIState struct {
	State UIState
}

type EventConnectStateUpdate struct {
	State      ConnectState
	ListenAddr string
}

type EventMinecraftJoinAddress struct {
	Expected        string
	Observed        string
	HostnameMatches bool
}

type EventUpdateAvailable struct {
	Version string
}

type EventUpdateDownloadProgress struct {
	Progress int
}

type EventUpdateDoInstall struct {
	Filepath string
}

type EventError struct {
	Error error
}

type EventWorldOpened struct {
	Folder                         string
	WorldID, WorldName, ServerName string
	HeightRange                    [2]int
}

// EventWorldAbandoned releases a provisional connection that received no terrain.
// Unlike deleting a capture, it must never remove files from disk.
type EventWorldAbandoned struct{ WorldID string }

type EventProcessingWorldUpdate struct {
	WorldID   string
	WorldName string
	State     string
}

type EventFinishedSavingWorld struct {
	WorldID   string
	ImagePath string
	WorldName string
	Filepath  string
	Chunks    int
	Entities  int
}

type EventInitialPacksInfo struct {
	Packs []protocol.TexturePackInfo
}

type EventProcessingPack struct {
	ID string
}

type EventPackDownloadProgress struct {
	ID         string
	BytesAdded int
}

type EventFinishedPack struct {
	ID       string
	Name     string
	Version  string
	Filepath string
	Icon     *image.RGBA
	Error    error
}
type EventResourcePackAvailable struct{ Pack *resource.Pack }
type EventCapturedBlockDefinitions struct {
	Packs       []*resource.Pack
	Definitions [][]byte
}

type EventDisplayAuthCode struct {
	AuthCode string
	URI      string
}

type EventAuthFinished struct {
	Error error
}

type MapTile struct {
	Pos protocol.ChunkPos
	Img image.RGBA
}

type EventMapTiles struct {
	WorldID string
	Tiles   []MapTile
}

type EventResetMap struct{ WorldID string }

type EventPlayerPosition struct {
	WorldID  string
	Position mgl32.Vec3
}

type EventPlayerSkin struct {
	PlayerName string
	Skin       protocol.Skin
	PlayerUUID string
	XUID       string
}

type EventPlayerEquipment struct {
	PlayerName string
	PlayerUUID string
	Equipment  map[string]string
}

type EventHandler interface {
	HandleEvent(event any) error
}

var eventHandler func(event any) error

func SetEventHandler(f func(event any) error) {
	eventHandler = f
}

func SendEvent(event any) {
	if eventHandler == nil {
		return
	}

	err := eventHandler(event)
	if err != nil {
		logrus.Errorf("event handler errored %s", err)
	}
}

type AuthHandler struct{}

type AuthCodeHandler interface {
	DisplayCode(verificationURI, userCode string)
	Successful()
}

var _ AuthCodeHandler = (*AuthHandler)(nil)

func (a *AuthHandler) DisplayCode(VerificationURI string, UserCode string) {
	SendEvent(&EventDisplayAuthCode{
		URI:      VerificationURI,
		AuthCode: UserCode,
	})
}

func (a *AuthHandler) Successful() {
	SendEvent(&EventAuthFinished{})
}
