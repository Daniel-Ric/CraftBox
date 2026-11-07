package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/df-mc/go-xsapi/v2/mpsd"
	"github.com/google/uuid"
)




type World struct {



	Joinability string



	HostName string `json:"hostName"`

	OwnerID string `json:"ownerId"`




	RakNetGUID string `json:"rakNetGUID"`

	Version string `json:"version"`

	LevelID string `json:"levelId"`


	WorldName string `json:"worldName"`

	WorldType string `json:"worldType"`




	Protocol int `json:"protocol"`

	MemberCount *int `json:"MemberCount,omitempty"`



	MaxMemberCount *int `json:"MaxMemberCount,omitempty"`



	BroadcastSetting BroadcastSetting


	LANGame bool `json:"LanGame"`

	EditorWorld bool `json:"isEditorWorld"`

	Hardcore bool `json:"isHardcore"`


	TransportLayer int


	OnlineCrossPlatformGame bool


	CrossPlayDisabled bool


	TitleID int `json:"TitleId"`


	SupportedConnections []Connection





	Nonces map[string]string `json:"nonces"`






	ExperienceID uuid.UUID `json:"ExperienceId,omitzero"`



	ExperienceWorldID uuid.UUID `json:"ExperienceWorldId,omitzero"`


	FriendID string `json:"FriendId,omitempty"`


	handleID uuid.UUID


	client *Client
}



func (w World) clone() World {
	w2 := w
	w2.Nonces = maps.Clone(w.Nonces)
	w2.SupportedConnections = slices.Clone(w.SupportedConnections)
	return w2
}


func (w World) HandleID() uuid.UUID {
	return w.handleID
}


type Connection struct {



	Type int `json:"ConnectionType"`


	HostIPAddress string `json:"HostIpAddress"`


	HostPort uint16

	NetherNetID json.Number `json:"NetherNetId"`





	PlayerMessagingID uuid.UUID `json:"PmsgId,omitzero"`
}


func (c *Connection) UnmarshalJSON(b []byte) error {
	type Alias Connection
	if err := json.Unmarshal(b, (*Alias)(c)); err != nil {
		return err
	}
	switch c.Type {
	case ConnectionTypeSignalingOverJSONRPC:
		if c.PlayerMessagingID == uuid.Nil {
			return errors.New("minecraft/p2p: Connection.PlayerMessagingID cannot be uuid.Nil")
		}
		fallthrough
	case ConnectionTypeSignalingOverWebSocket:
		if c.NetherNetID == "" {
			return errors.New("minecraft/p2p: Connection.NetherNetID cannot be empty string")
		}
	default:
		return fmt.Errorf("minecraft/p2p: invalid Connection.Type: %d", c.Type)
	}
	return nil
}


func (c Connection) Address() string {
	switch c.Type {
	case ConnectionTypeSignalingOverWebSocket:
		return c.NetherNetID.String()
	case ConnectionTypeSignalingOverJSONRPC:
		return c.PlayerMessagingID.String()
	default:
		panic("unreachable")
	}
}

const (


	TransportLayerRakNet = iota << 1

	TransportLayerNetherNet
)

const (




	ConnectionTypeSignalingOverJSONRPC = 7





	ConnectionTypeSignalingOverWebSocket = 3




	ConnectionTypeSignalingOverLAN = 4
)

const (


	JoinabilityInviteOnly = "invite_only"

	JoinabilityFriends = "joinable_by_friends"
)

const (


	BroadcastSettingInviteOnly BroadcastSetting = iota + 1


	BroadcastSettingFriendsOnly


	BroadcastSettingFriendsOfFriends
)



type BroadcastSetting int




func (s BroadcastSetting) JoinRestriction() string {
	switch s {
	case BroadcastSettingInviteOnly:
		return mpsd.SessionRestrictionLocal
	case BroadcastSettingFriendsOnly, BroadcastSettingFriendsOfFriends:
		return mpsd.SessionRestrictionFollowed
	default:
		panic("unreachable")
	}
}




func (s BroadcastSetting) ReadRestriction() string {
	switch s {
	case BroadcastSettingInviteOnly, BroadcastSettingFriendsOnly, BroadcastSettingFriendsOfFriends:
		return mpsd.SessionRestrictionFollowed
	default:
		panic("unreachable")
	}
}


func (s BroadcastSetting) Valid() bool {
	switch s {
	case BroadcastSettingInviteOnly, BroadcastSettingFriendsOnly, BroadcastSettingFriendsOfFriends:
		return true
	default:
		return false
	}
}
