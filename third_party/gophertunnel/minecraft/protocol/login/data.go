package login

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"golang.org/x/text/language"
)

type IdentityData struct {
	XUID string

	Identity string `json:"identity"`

	DisplayName string `json:"displayName"`

	TitleID string `json:"titleId,omitempty"`

	PlayFabTitleID string `json:"-"`

	PlayFabID string `json:"-"`
}

var checkOfflineUsername = regexp.MustCompile(`[ \p{L}]`).MatchString

var checkOnlineUsername = regexp.MustCompile("[A-Za-z0-9 ]").MatchString

func (data IdentityData) Validate() error {
	if _, err := strconv.ParseInt(data.XUID, 10, 64); err != nil && len(data.XUID) != 0 {
		return fmt.Errorf("XUID must be parseable as an int64, but got %v", data.XUID)
	}
	if id, err := uuid.Parse(data.Identity); err != nil || id == uuid.Nil {
		return fmt.Errorf("UUID must be parseable as a valid UUID, but got %v", data.Identity)
	}
	nameLimit := 15
	if data.XUID == "" {

		nameLimit = 16
	}
	if len(data.DisplayName) == 0 || len(data.DisplayName) > nameLimit {
		return fmt.Errorf("DisplayName must not be empty or longer than %d characters, but got %v characters", nameLimit, len(data.DisplayName))
	}
	if data.DisplayName[0] == ' ' || data.DisplayName[len(data.DisplayName)-1] == ' ' {
		return fmt.Errorf("DisplayName may not have a space as first/last character, but got %v", data.DisplayName)
	}
	if data.DisplayName[0] >= '0' && data.DisplayName[0] <= '9' {
		return fmt.Errorf("DisplayName may not have a number as first character, but got %v", data.DisplayName)
	}
	if data.XUID != "" {
		if !checkOnlineUsername(data.DisplayName) {
			return fmt.Errorf("DisplayName for authorized client must only contain numbers, Latin letters and spaces, but got %v", data.DisplayName)
		}
	} else {
		if !checkOfflineUsername(data.DisplayName) {
			return fmt.Errorf("DisplayName for unauthorized client must only contain numbers, letters and spaces, but got %v", data.DisplayName)
		}
	}

	if strings.Contains(data.DisplayName, "  ") {
		return fmt.Errorf("DisplayName must only have single spaces, but got %v", data.DisplayName)
	}
	return nil
}

type ClientData struct {
	AnimatedImageData []SkinAnimation

	CapeData string

	CapeID string `json:"CapeId"`

	CapeImageHeight, CapeImageWidth int

	CapeOnClassicSkin bool

	ClientRandomID int64 `json:"ClientRandomId"`

	CurrentInputMode int

	DefaultInputMode int

	DeviceModel string

	DeviceOS protocol.DeviceOS

	DeviceID DeviceID `json:"DeviceId"`

	GameVersion string

	GUIScale int `json:"GuiScale"`

	FilterProfanity bool

	ClientEditorConnectionIntent int

	ClientIsEditorCapable bool

	LanguageCode string

	PersonaSkin bool

	PlatformOfflineID string `json:"PlatformOfflineId"`

	PlatformOnlineID string `json:"PlatformOnlineId"`

	PlatformUserID string `json:"PlatformUserId,omitempty"`

	PremiumSkin bool

	SelfSignedID string `json:"SelfSignedId"`

	ServerAddress string

	SkinAnimationData string

	SkinData string

	SkinGeometry string `json:"SkinGeometryData"`

	SkinGeometryVersion string `json:"SkinGeometryDataEngineVersion"`

	SkinID string `json:"SkinId"`

	PlayFabID string `json:"PlayFabId"`

	SkinImageHeight, SkinImageWidth int

	SkinResourcePatch string

	SkinColour string `json:"SkinColor"`

	ArmSize string

	PersonaPieces []PersonaPiece

	PieceTintColours []PersonaPieceTintColour `json:"PieceTintColors"`

	ThirdPartyName string

	ThirdPartyNameOnly *bool `json:"ThirdPartyNameOnly,omitempty"`

	UIProfile int

	TrustedSkin bool

	OverrideSkin bool

	CompatibleWithClientSideChunkGen bool

	MaxViewDistance int

	MemoryTier int

	PlatformType int

	GraphicsMode int

	PartyID string `json:"PartyId"`

	PartyLeader bool `json:"IsPartyLeader"`

	ProfileHash string `json:"ProfileHash"`

	Nonce string `json:",omitempty"`
}

type PersonaPiece struct {
	Default bool `json:"IsDefault"`

	PackID string `json:"PackId"`

	PieceID string `json:"PieceId"`

	PieceType string

	ProductID string `json:"ProductId"`
}

type PersonaPieceTintColour struct {
	Colours [4]string `json:"Colors"`

	PieceType string
}

type SkinAnimation struct {
	Frames float64

	Image string

	ImageHeight, ImageWidth int

	Type int

	AnimationExpression int
}

var checkVersion = regexp.MustCompile("[0-9.]").MatchString

func (data ClientData) Validate() error {
	if data.DeviceOS <= 0 || data.DeviceOS > 15 {
		return fmt.Errorf("DeviceOS must carry a value between 1 and 15, but got %v", data.DeviceOS)
	}
	if !checkVersion(data.GameVersion) {
		return fmt.Errorf("GameVersion must only contain dots and numbers, but got %v", data.GameVersion)
	}
	if _, err := language.Parse(strings.Replace(data.LanguageCode, "_", "-", 1)); err != nil {
		return fmt.Errorf("LanguageCode must be a valid BCP-47 ISO language code, but got %v", data.LanguageCode)
	}
	if _, err := uuid.Parse(data.PlatformOfflineID); err != nil && len(data.PlatformOfflineID) != 0 {
		return fmt.Errorf("PlatformOfflineID must be parseable as a valid UUID or empty, but got %v", data.PlatformOfflineID)
	}
	if _, err := strconv.ParseUint(data.PlatformOnlineID, 10, 64); err != nil && len(data.PlatformOnlineID) != 0 {
		return fmt.Errorf("PlatformOnlineID must be parseable as an int64 or empty, but got %v", data.PlatformOnlineID)
	}
	if _, err := uuid.Parse(data.SelfSignedID); data.SelfSignedID != "" && err != nil {
		return fmt.Errorf("SelfSignedID must be parseable as a valid UUID, but got %v", data.SelfSignedID)
	}
	if strings.Contains(data.ServerAddress, "://") {

		ind := strings.LastIndex(data.ServerAddress, ":")
		u, err := url.Parse(data.ServerAddress[:ind])
		if err != nil {
			return fmt.Errorf("ServerAddress must be a URL, but got %v", data.ServerAddress)
		}
		if u.Host == "" || u.Port() == "" || u.Port() != data.ServerAddress[ind+1:] ||
			(u.Scheme != "https" && u.Scheme != "http") {
			return fmt.Errorf("ServerAddress is invalid: %v", data.ServerAddress)
		}
	} else if _, err := net.ResolveUDPAddr("udp", data.ServerAddress); err != nil {

		if _, idErr := strconv.ParseUint(data.ServerAddress, 10, 64); idErr != nil {
			return fmt.Errorf("ServerAddress must be a UDP address, URL or NetherNet network ID, but got %v", data.ServerAddress)
		}
	}
	if err := base64DecLength(data.SkinData, data.SkinImageHeight*data.SkinImageWidth*4); err != nil {
		return fmt.Errorf("SkinData is invalid: %w", err)
	}
	if err := base64DecLength(data.CapeData, data.CapeImageHeight*data.CapeImageWidth*4); err != nil {
		return fmt.Errorf("CapeData is invalid: %w", err)
	}
	for _, s := range data.PlayFabID {
		if (s < '0' || s > '9') && (s < 'a' || s > 'f') {
			return fmt.Errorf("PlayFabID must consist of hex characters, got %v", data.PlayFabID)
		}
	}
	for _, anim := range data.AnimatedImageData {
		if err := base64DecLength(anim.Image, anim.ImageHeight*anim.ImageWidth*4); err != nil {
			return fmt.Errorf("invalid animated image data: %w", err)
		}
		if anim.Type < 0 || anim.Type > 3 {
			return fmt.Errorf("invalid animation type: %v", anim.Type)
		}
	}
	if geomData, err := base64.StdEncoding.DecodeString(data.SkinGeometry); err != nil {
		return fmt.Errorf("SkinGeometry was not a valid base64 string: %w", err)
	} else if len(geomData) != 0 {
		m := make(map[string]any)
		if err := json.Unmarshal(geomData, &m); err != nil {
			return fmt.Errorf("SkinGeometry base64 decoded was not a valid JSON string: %w", err)
		}
	}
	b, err := base64.StdEncoding.DecodeString(data.SkinResourcePatch)
	if err != nil {
		return fmt.Errorf("SkinResourcePatch was not a valid base64 string: %w", err)
	}
	m := make(map[string]any)
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("SkinResourcePatch base64 decoded was not a valid JSON string: %w", err)
	}
	if data.SkinID == "" {
		return fmt.Errorf("SkinID must not be an empty string")
	}
	if data.UIProfile < 0 || data.UIProfile > 2 {
		return fmt.Errorf("UIProfile must be between 0-2, but got %v", data.UIProfile)
	}
	return nil
}

func base64DecLength(base64Data string, validLengths ...int) error {
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("decode base64 data: %w", err)
	}
	actualLength := len(data)
	for _, length := range validLengths {
		if length == actualLength {
			return nil
		}
	}
	return fmt.Errorf("invalid size: got %v, expected one of %v", actualLength, validLengths)
}

type DeviceID string

type DeviceIDFormat uint8

const (
	DeviceIDFormatUpperHexString DeviceIDFormat = iota
	DeviceIDFormatLowerHexString
	DeviceIDFormatBase64
	DeviceIDFormatUUID
	DeviceIDFormatInvalid
)

var lowerMatch = regexp.MustCompile(`^[a-z0-9]+$`)

func (dID DeviceID) Format() DeviceIDFormat {
	deviceID := string(dID)

	data, err := hex.DecodeString(deviceID)
	if err == nil {
		if len(data) != 16 {
			return DeviceIDFormatInvalid
		}

		if lowerMatch.MatchString(deviceID) {
			return DeviceIDFormatLowerHexString
		}
		return DeviceIDFormatUpperHexString
	}

	_, err = uuid.Parse(deviceID)
	if err == nil {
		return DeviceIDFormatUUID
	}

	data, err = base64.StdEncoding.DecodeString(deviceID)
	if err == nil && len(data) == 32 {
		return DeviceIDFormatBase64
	}

	return DeviceIDFormatInvalid
}

func (data ClientData) ExpectedDeviceIDFormat() DeviceIDFormat {
	switch data.DeviceOS {
	case protocol.DeviceAndroid:
		return DeviceIDFormatLowerHexString
	case protocol.DeviceIOS:
		return DeviceIDFormatUpperHexString
	case protocol.DeviceWin32:
		return DeviceIDFormatLowerHexString
	case protocol.DeviceOrbis:
		return DeviceIDFormatUUID
	case protocol.DeviceXBOX:
		return DeviceIDFormatBase64
	}
	return DeviceIDFormatInvalid
}

func (f DeviceIDFormat) Generate() DeviceID {
	switch f {
	case DeviceIDFormatUpperHexString:
		return DeviceID(strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")))
	case DeviceIDFormatUUID:
		return DeviceID(uuid.NewString())
	case DeviceIDFormatBase64:
		b := make([]byte, 32)
		_, _ = cryptorand.Read(b)
		return DeviceID(base64.StdEncoding.EncodeToString(b))
	default:
		return DeviceID(strings.ReplaceAll(uuid.NewString(), "-", ""))
	}
}
