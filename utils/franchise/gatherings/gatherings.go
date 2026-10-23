package gatherings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/franchise/internal"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/service"
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
	ServiceURI *url.URL `json:"serviceUri"`
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

func (g *Service) JoinExperience(ctx context.Context, mcToken *service.Token, id uuid.UUID) (string, error) {
	type Join struct {
		NetworkProtocol string `json:"networkProtocol"`
		IPV4Address     string `json:"ipV4Address"`
		Port            int    `json:"port"`
	}
	resp, err := internal.DoRequest[internal.Result[Join]](
		ctx, http.DefaultClient, "POST",
		g.ServiceURI.JoinPath("/api/v2.0/join/experience").String(),
		map[string]any{"experienceId": id},
		mcToken.SetAuthHeader,
	)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", resp.Data.IPV4Address, resp.Data.Port), nil
}

type FeaturedServer struct {
	Name         string
	Address      string
	ExperienceId string
}

func (g *Service) GetFeaturedServers(ctx context.Context, mcToken *service.Token) ([]FeaturedServer, error) {
	type Translated struct {
		Neutral string `json:"NEUTRAL"`
	}
	type Images struct {
		Tag  string `json:"Tag"`
		ID   string `json:"Id"`
		Type string `json:"Type"`
		URL  string `json:"Url"`
	}
	type AvailableGames struct {
		Description string `json:"description"`
		ImageTag    string `json:"imageTag"`
		Subtitle    string `json:"subtitle"`
		Title       string `json:"title"`
	}
	type DisplayProperties struct {
		AvailableGames    []AvailableGames `json:"availableGames"`
		CreatorName       string           `json:"creatorName"`
		MaxClientVersion  string           `json:"maxClientVersion"`
		MinClientVersion  string           `json:"minClientVersion"`
		News              string           `json:"news"`
		NewsTitle         string           `json:"newsTitle"`
		OriginalCreatorID string           `json:"originalCreatorId"`
		Port              int              `json:"port"`
		RequireXBL        string           `json:"requireXBL"`
		StorePageID       string           `json:"storePageId"`
		URL               string           `json:"url"`
		WhitelistURL      string           `json:"whitelistUrl"`
		AllowListURL      string           `json:"allowListUrl"`
		ExperienceID      string           `json:"experienceId"`
		IsTop             bool             `json:"isTop"`
	}
	type EntityKey struct {
		ID         string `json:"Id"`
		Type       string `json:"Type"`
		TypeString string `json:"TypeString"`
	}
	type Items struct {
		ID                string            `json:"Id"`
		Type              string            `json:"Type"`
		AlternateIds      []any             `json:"AlternateIds"`
		Title             Translated        `json:"Title"`
		Description       Translated        `json:"Description,omitempty"`
		ContentType       string            `json:"ContentType"`
		Platforms         []string          `json:"Platforms"`
		Tags              []string          `json:"Tags"`
		CreationDate      time.Time         `json:"CreationDate"`
		LastModifiedDate  time.Time         `json:"LastModifiedDate"`
		StartDate         time.Time         `json:"StartDate"`
		Contents          []any             `json:"Contents"`
		Images            []Images          `json:"Images"`
		ItemReferences    []any             `json:"ItemReferences"`
		DisplayProperties DisplayProperties `json:"DisplayProperties,omitempty"`
		IsStackable       bool              `json:"IsStackable"`
		CreatorEntityKey  EntityKey         `json:"CreatorEntityKey"`
		IsHydrated        bool              `json:"IsHydrated"`
		Keywords          Translated        `json:"Keywords"`
		CreatorEntity     EntityKey         `json:"CreatorEntity,omitempty"`
	}
	type Data struct {
		Count             int     `json:"Count"`
		Items             []Items `json:"Items"`
		ConfigurationName string  `json:"ConfigurationName"`
	}

	resp, err := internal.DoRequest[internal.Data[Data]](
		ctx, http.DefaultClient, "POST",
		g.ServiceURI.JoinPath("/api/v2.0/discovery/blob/client").String(),
		nil, mcToken.SetAuthHeader)
	if err != nil {
		return nil, err
	}

	var out []FeaturedServer
	for _, item := range resp.Data.Items {
		address := ""
		if item.DisplayProperties.URL != "" {
			address = fmt.Sprintf("%s:%d", item.DisplayProperties.URL, item.DisplayProperties.Port)
		}
		out = append(out, FeaturedServer{
			Name:         item.Title.Neutral,
			Address:      address,
			ExperienceId: item.DisplayProperties.ExperienceID,
		})
	}
	return out, nil
}
