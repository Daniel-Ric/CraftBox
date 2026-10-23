package gatherings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/df-mc/go-playfab/v2"
	"github.com/df-mc/go-playfab/v2/catalog"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/service"
	"github.com/sandertv/gophertunnel/minecraft/service/internal"
	"golang.org/x/text/language"
)

type Environment struct {
	ServiceURI *url.URL `json:"serviceUri"`

	HTTPClient *http.Client `json:"-"`
}

var DefaultEnvironment = &Environment{
	ServiceURI: &url.URL{
		Scheme: "https",
		Host:   "gatherings-secondary.franchise.minecraft-services.net",
	},
}

func NewClient(src service.TokenSource) *Client {
	return DefaultEnvironment.New(src)
}

func (e *Environment) New(src service.TokenSource) *Client {
	return &Client{
		src:    src,
		client: e.httpClient(),
		env:    e,
	}
}

func (e *Environment) httpClient() *http.Client {
	if e.HTTPClient != nil {
		return e.HTTPClient
	}
	return http.DefaultClient
}

type Client struct {
	src    service.TokenSource
	client *http.Client
	env    *Environment
}

func (c *Client) ParseExperience(item catalog.Item) (*Experience, error) {
	var exp Experience
	if err := json.Unmarshal(item.DisplayProperties, &exp); err != nil {
		return nil, fmt.Errorf("decode display properties for %s: %w", item.ID, err)
	}
	exp.Item, exp.client = item, c
	return &exp, nil
}

func (c *Client) ParseFeaturedServer(item catalog.Item) (*FeaturedServer, error) {
	var server FeaturedServer
	if err := json.Unmarshal(item.DisplayProperties, &server); err != nil {
		return nil, fmt.Errorf("decode display properties for %s: %w", item.ID, err)
	}
	server.Item = item
	return &server, nil
}

func (c *Client) SearchItems(ctx context.Context, filter catalog.SearchFilter, opts ...playfab.RequestOption) (*catalog.SearchResult, error) {
	buf := &bytes.Buffer{}
	defer buf.Reset()
	if err := json.NewEncoder(buf).Encode(filter); err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}

	requestURL := c.env.ServiceURI.JoinPath("/api/v2.0/discovery/blob/client").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	opts = append(opts, playfab.AcceptLanguage([]language.Tag{language.AmericanEnglish}))
	for _, opt := range opts {
		if err := opt(req); err != nil {
			return nil, fmt.Errorf("apply request option: %w", err)
		}
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:

		var result struct {
			Data *catalog.SearchResult `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode response body: %w", err)
		}
		if result.Data == nil {
			return nil, errors.New("service/gatherings: invalid search result")
		}
		return result.Data, nil
	default:
		return nil, internal.Err(resp)
	}
}

func (c *Client) Experiences(ctx context.Context) ([]*Experience, error) {
	result, err := c.SearchItems(ctx, catalog.SearchFilter{

		Filter:  "(ContentType eq '3PP_V2.0') and (DisplayProperties/experienceId ne '')",
		OrderBy: "startDate desc",
		Select:  "images",
	})
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("service/gatherings: empty search result")
	}
	experiences := make([]*Experience, 0, len(result.Items))
	for _, item := range result.Items {
		var exp Experience
		if err := json.Unmarshal(item.DisplayProperties, &exp); err != nil {
			return nil, fmt.Errorf("service/gatherings: decode experience (%s): %w", item.ID, err)
		}
		if !exp.Valid() {
			continue
		}
		exp.Item, exp.client = item, c
		experiences = append(experiences, &exp)
	}
	return experiences, nil
}

func (c *Client) FeaturedServers(ctx context.Context) ([]*FeaturedServer, error) {
	result, err := c.SearchItems(ctx, catalog.SearchFilter{

		Filter:  "(ContentType eq '3PP_V2.0') and (DisplayProperties/experienceId eq '')",
		OrderBy: "startDate desc",
		Select:  "images",
	})
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("service/gatherings: empty search result")
	}
	servers := make([]*FeaturedServer, 0, len(result.Items))
	for _, item := range result.Items {
		server, err := c.ParseFeaturedServer(item)
		if err != nil {
			return nil, err
		}
		if !server.Valid() {
			continue
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func (c *Client) JoinExperience(ctx context.Context, id uuid.UUID) (*Address, error) {
	if id == uuid.Nil {
		return nil, errors.New("service/gatherings: experience ID is nil")
	}

	buf := &bytes.Buffer{}
	defer buf.Reset()
	if err := json.NewEncoder(buf).Encode(map[string]any{
		"experienceId": id,
	}); err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	requestURL := c.env.ServiceURI.JoinPath("/api/v2.0/join/experience").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var result internal.Result[*Address]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode response body: %w", err)
		}
		if result.Data == nil {
			return nil, errors.New("service/gatherings: invalid join result")
		}
		return result.Data, nil
	default:
		return nil, internal.Err(resp)
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	token, err := c.src.ServiceToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("request service token: %w", err)
	}
	token.SetAuthHeader(req)

	req.Header.Set("Client-Version", protocol.CurrentVersion)
	req.Header.Set("User-Agent", "libhttpclient/1.0.0.0")
	return c.client.Do(req)
}

func (c *Client) JoinFriend(ctx context.Context, experienceID, worldID uuid.UUID, friendID string) (*Address, error) {
	if experienceID == uuid.Nil {
		return nil, fmt.Errorf("service/gatherings: experience ID must not be nil")
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(joinFriendRequest{
		ExperienceID: experienceID,
		WorldID:      worldID,
	}); err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}

	requestURL := c.env.ServiceURI.JoinPath("/api/v2.0/join/world/client")
	if friendID != "" {
		q := requestURL.Query()
		q.Set("friendId", friendID)
		requestURL.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var result internal.Result[*Address]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode response body: %w", err)
		}
		if result.Data == nil {
			return nil, errors.New("service/gatherings: invalid JoinFriend response")
		}
		return result.Data, nil
	default:
		return nil, internal.Err(resp)
	}
}

func (c *Client) PlayerCounts(ctx context.Context) (map[uuid.UUID]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.env.ServiceURI.JoinPath("/api/v2.0/dataquery/playercounts").String(), nil)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var result internal.Result[[]playerCountRecord]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode response body: %w", err)
		}
		m := make(map[uuid.UUID]int, len(result.Data))
		for _, entry := range result.Data {
			if entry.ExperienceID == uuid.Nil {
				return nil, errors.New("gatherings: invalid player counts response")
			}
			m[entry.ExperienceID] = entry.PlayerCount
		}
		return m, nil
	default:
		return nil, internal.Err(resp)
	}
}

type playerCountRecord struct {
	ExperienceID uuid.UUID `json:"experienceId"`

	PlayerCount int `json:"playerCount"`
}

type joinFriendRequest struct {
	ExperienceID uuid.UUID `json:"experienceId"`

	WorldID uuid.UUID `json:"worldId,omitzero"`
}
