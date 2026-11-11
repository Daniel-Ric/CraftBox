package gatherings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/franchise/internal"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/service"
	native "github.com/sandertv/gophertunnel/minecraft/service/gatherings"
)

type Segment struct {
	SegmentType  string    `json:"segmentType"`
	StartTimeUtc time.Time `json:"startTimeUtc"`
	EndTimeUtc   time.Time `json:"endTimeUtc"`
	UI           struct {
		CaptionText              string `json:"captionText"`
		CaptionForegroundColor   string `json:"captionForegroundColor"`
		CaptionBackgroundColor   string `json:"captionBackgroundColor"`
		StartScreenButtonText    string `json:"startScreenButtonText"`
		BadgeImage               string `json:"badgeImage"`
		CaptionIncludesCountdown bool   `json:"captionIncludesCountdown"`
		ActionButtonText         string `json:"actionButtonText"`
		InfoButtonText           string `json:"infoButtonText"`
		HeaderText               string `json:"headerText"`
		TitleText                string `json:"titleText"`
		BodyText                 string `json:"bodyText"`
		EventImage               string `json:"eventImage"`
		BodyImage                string `json:"bodyImage"`
		ActionButtonURL          string `json:"actionButtonUrl"`
		InfoButtonURL            string `json:"infoButtonUrl"`
	} `json:"ui"`
}

type Gathering struct {
	service *Service

	GatheringID   string         `json:"gatheringId"`
	StartTimeUtc  time.Time      `json:"startTimeUtc"`
	EndTimeUtc    time.Time      `json:"endTimeUtc"`
	Segments      []Segment      `json:"segments"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	IsEnabled     bool           `json:"isEnabled"`
	IsPrivate     bool           `json:"isPrivate"`
	GatheringType string         `json:"gatheringType"`
	AdditionalLoc map[string]any `json:"additionalLoc"`
}

func (g *Gathering) Address(ctx context.Context, mcToken *service.Token) (string, error) {
	type Venue struct {
		Venue struct {
			ServerIpAddress string `json:"serverIpAddress"`
			ServerPort      int    `json:"serverPort"`
		} `json:"venue"`
	}
	accessUrl := g.service.ServiceURI.JoinPath("/api/v1.0/access")
	accessUrl.RawQuery = url.Values{
		"lang":              []string{"en-US"},
		"clientVersion":     []string{protocol.CurrentVersion},
		"clientPlatform":    []string{"Windows10"},
		"clientSubPlatform": []string{"Windows10"},
	}.Encode()
	accessResp, err := internal.DoRequest[any](ctx, http.DefaultClient, "GET", accessUrl.String(), nil, mcToken.SetAuthHeader)
	if err != nil {
		return "", err
	}
	_ = accessResp

	resp, err := internal.DoRequest[internal.Result[Venue]](
		ctx, http.DefaultClient, "GET",
		g.service.ServiceURI.JoinPath("/api/v1.0/venue/", g.GatheringID).String(),
		nil, mcToken.SetAuthHeader,
	)
	if err != nil {
		return "", err
	}

	if resp.Data.Venue.ServerIpAddress == "" {
		return "", errors.New("didnt get a server address")
	}

	return fmt.Sprintf("%s:%d", resp.Data.Venue.ServerIpAddress, resp.Data.Venue.ServerPort), nil
}

type Service struct {
	ServiceURI    *url.URL       `json:"serviceUri"`
	CatalogClient *native.Client `json:"-"`
	playerCounts  playerCountCache
}

func (a *Service) ServiceName() string {
	return "gatherings"
}

func (e *Service) UnmarshalJSON(b []byte) error {
	type Alias Service
	data := struct {
		*Alias
		ServiceURI string `json:"serviceUri"`
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	if data.ServiceURI == "" {
		return errors.New("service: Service.ServiceURI cannot be empty string")
	}
	var err error
	e.ServiceURI, err = url.Parse(data.ServiceURI)
	if err != nil {
		return fmt.Errorf("parse ServiceURI: %w", err)
	}
	return nil
}

func (g *Service) GetGatherings(ctx context.Context, mcToken *service.Token) ([]*Gathering, error) {
	var configUrl = g.ServiceURI.JoinPath("/api/v1.0/config/public")
	configUrl.RawQuery = url.Values{
		"lang":              []string{"en-US"},
		"clientVersion":     []string{protocol.CurrentVersion},
		"clientPlatform":    []string{"Windows10"},
		"clientSubPlatform": []string{"Windows10"},
	}.Encode()
	configResp, err := internal.DoRequest[internal.Result[[]Gathering]](ctx, http.DefaultClient, "GET", configUrl.String(), nil, mcToken.SetAuthHeader)
	if err != nil {
		return nil, err
	}

	var gatherings []*Gathering
	for _, gathering := range configResp.Data {
		gathering.service = g
		gatherings = append(gatherings, &gathering)
	}
	return gatherings, nil
}

type ExperienceAssignment struct {
	NetworkProtocol string `json:"networkProtocol"`
	IPV4Address     string `json:"ipV4Address"`
	Port            int    `json:"port"`
	NetherNetID     string `json:"netherNetId"`
	DestinationInfo struct {
		ExperienceName string `json:"experienceName"`
		WorldName      string `json:"worldName"`
	} `json:"destinationInfo"`
}

func (a ExperienceAssignment) Address() (string, error) {
	switch strings.ToLower(strings.TrimSpace(a.NetworkProtocol)) {
	case "nethernet":
		id := strings.TrimSpace(a.NetherNetID)
		if id == "" || strings.ContainsAny(id, " /\\\t\r\n") {
			return "", errors.New("Experience allocation did not provide a valid NetherNet signaling ID")
		}
		return "nethernet-signaling:" + id, nil
	case "nethernet_jsonrpc":
		id, err := uuid.Parse(a.NetherNetID)
		if err != nil || id == uuid.Nil {
			return "", errors.New("Experience allocation did not provide a valid NetherNet messaging ID")
		}
		return "nethernet-jsonrpc:" + id.String(), nil
	case "default", "nethernet_http", "":
		if strings.TrimSpace(a.IPV4Address) == "" || a.Port < 1 || a.Port > 65535 {
			return "", errors.New("Experience allocation did not provide a server address and port")
		}
		return net.JoinHostPort(a.IPV4Address, strconv.Itoa(a.Port)), nil
	default:
		return "", fmt.Errorf("Experience uses unsupported transport %q", a.NetworkProtocol)
	}
}

func (g *Service) ResolveExperience(ctx context.Context, mcToken *service.Token, id uuid.UUID) (*ExperienceAssignment, error) {
	resp, err := internal.DoRequest[internal.Result[ExperienceAssignment]](
		ctx, http.DefaultClient, "POST",
		g.ServiceURI.JoinPath("/api/v2.0/join/experience").String(),
		map[string]any{"experienceId": id},
		mcToken.SetAuthHeader,
	)
	if err != nil {
		return nil, err
	}
	if _, err := resp.Data.Address(); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (g *Service) JoinExperience(ctx context.Context, mcToken *service.Token, id uuid.UUID) (string, error) {
	assignment, err := g.ResolveExperience(ctx, mcToken, id)
	if err != nil {
		return "", err
	}
	return assignment.Address()
}

type FeaturedServer struct {
	Name         string
	Address      string
	ExperienceId string
	Description  string
}
