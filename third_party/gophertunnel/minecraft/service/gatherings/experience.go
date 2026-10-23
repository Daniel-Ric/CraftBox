package gatherings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/df-mc/go-playfab/v2/catalog"
	"github.com/google/uuid"
)

type Experience struct {
	experience

	ID uuid.UUID `json:"experienceId"`

	client *Client
}

func (e *Experience) Valid() bool {
	return e != nil && e.ID != uuid.Nil
}

type experience struct {
	Item catalog.Item `json:"-"`

	AvailableGames []AvailableGame `json:"availableGames"`

	CreatorName string `json:"creatorName"`

	MaxClientVersion string `json:"maxClientVersion"`

	MinClientVersion string `json:"minClientVersion"`

	News string `json:"news"`

	NewsTitle string `json:"newsTitle"`

	OriginalCreatorID string `json:"originalCreatorId"`

	RequireXBL string `json:"requireXBL"`

	StorePageID string `json:"storePageId"`

	WhiteListURL string `json:"whitelistUrl"`

	AllowListURL string `json:"allowListUrl"`

	Rank int `json:"rank"`

	RequiredExperiments []string `json:"requiredExperiments"`
}

func (e *Experience) UnmarshalJSON(b []byte) (err error) {
	type Alias Experience
	data := struct {
		*Alias
		ID string `json:"experienceId"`
	}{Alias: (*Alias)(e)}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if data.ID != "" {
		e.ID, err = uuid.Parse(data.ID)
		if err != nil {
			return fmt.Errorf("service/gatherings: parse Experience.ID: %w", err)
		}
	}
	return nil
}

func (e *Experience) Join(ctx context.Context) (*Address, error) {
	return e.client.JoinExperience(ctx, e.ID)
}

type Address struct {
	NetworkProtocol string `json:"networkProtocol"`

	IPv4Address string `json:"ipV4Address"`

	Port uint16 `json:"port"`

	DestinationInfo DestinationInfo `json:"destinationInfo"`
}

func (a Address) String() string {
	return net.JoinHostPort(a.IPv4Address, strconv.Itoa(int(a.Port)))
}

const (
	NetworkProtocolDefault = "Default"
)

type DestinationInfo struct {
	CreatorID string `json:"creatorId"`

	ExperienceID uuid.UUID `json:"experienceId"`

	ExperienceName string `json:"experienceName"`

	ScenarioID uuid.UUID `json:"scenarioId"`

	MPSASScenarioID string `json:"mpsasScenarioId"`

	ServerID string `json:"serverId"`

	TargetID string `json:"targetId"`

	WorldID uuid.UUID `json:"worldId"`

	WorldName string `json:"worldName"`
}

type AvailableGame struct {
	Description string `json:"description"`

	ImageTag string `json:"imageTag"`

	Subtitle string `json:"subtitle"`

	Title string `json:"title"`
}

type FeaturedServer struct {
	experience

	Port uint16 `json:"port"`

	Host string `json:"url"`
}

func (s FeaturedServer) Valid() bool {
	return s.Host != "" && s.Port != 0
}

func (s FeaturedServer) Address() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(int(s.Port)))
}
