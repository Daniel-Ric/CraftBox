package realms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

type Client struct {
	tokenSrc   oauth2.TokenSource
	xblToken   *auth.XBLToken
	httpClient *http.Client

	negotiateMu sync.Mutex

	versionMu        sync.Mutex
	preferredVersion string
	acceptedVersion  string
	searchFailedAt   time.Time
}

const (
	realmsBaseURL      = "https://bedrock.frontendlegacy.realms.minecraft-services.net"
	realmsRelyingParty = "https://pocket.realms.minecraft.net/"
)

var (
	ErrPlayerNotInRealm = errors.New("player not in realm")
	ErrRealmNotFound    = errors.New("realm not found")
)

func NewClient(src oauth2.TokenSource, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		tokenSrc:   src,
		httpClient: httpClient,
	}
}

type Player struct {
	UUID       string `json:"uuid"`
	Name       string `json:"Name"`
	Operator   bool   `json:"operator"`
	Accepted   bool   `json:"accepted"`
	Online     bool   `json:"online"`
	Permission string `json:"permission"`
}

type Realm struct {
	ID int `json:"id"`

	RemoteSubscriptionID string `json:"remoteSubscriptionID"`

	Owner string `json:"owner"`

	OwnerUUID string `json:"ownerUUID"`

	Name string `json:"name"`

	MOTD string `json:"motd"`

	DefaultPermission string `json:"defaultPermission"`

	State string `json:"state"`

	DaysLeft int `json:"daysLeft"`

	Expired bool `json:"expired"`

	ExpiredTrial bool `json:"expiredTrial"`

	GracePeriod bool `json:"gracePeriod"`

	WorldType string `json:"worldType"`

	Players []Player `json:"players"`

	MaxPlayers int `json:"maxPlayers"`

	MinigameName string `json:"minigameName"`

	MinigameID string `json:"minigameId"`

	MinigameImage string `json:"minigameImage"`

	ActiveSlot int `json:"activeSlot"`

	Slots []struct{} `json:"slots"`

	Member bool `json:"member"`

	ClubID int64 `json:"clubId"`

	SubscriptionRefreshStatus struct{} `json:"subscriptionRefreshStatus"`

	client *Client
}

type StoryOptIn string

const (
	StoryOptInNone   StoryOptIn = "NONE"
	StoryOptInOptIn  StoryOptIn = "OPT_IN"
	StoryOptInOptOut StoryOptIn = "OPT_OUT"
)

type StorySettings struct {
	AutoStories   bool       `json:"autostories"`
	Coordinates   bool       `json:"coordinates"`
	Notifications bool       `json:"notifications"`
	PlayerOptIn   StoryOptIn `json:"playerOptIn"`
	RealmOptIn    StoryOptIn `json:"realmOptIn"`
	Timeline      bool       `json:"timeline"`
}

type RealmAddress struct {
	Address           string          `json:"address"`
	NetworkProtocol   NetworkProtocol `json:"networkProtocol"`
	PendingUpdate     bool            `json:"pendingUpdate"`
	SessionRegionData struct {
		RegionName     string `json:"regionName"`
		ServiceQuality int    `json:"serviceQuality"`
	} `json:"sessionRegionData"`
}

func (r *Realm) Address(ctx context.Context) (RealmAddress, error) {
	if r.client == nil {
		return RealmAddress{}, fmt.Errorf("realm client is nil")
	}
	return r.client.RealmAddress(ctx, r.ID)
}

func (r *Client) RealmAddress(ctx context.Context, realmID int) (RealmAddress, error) {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return RealmAddress{}, ctx.Err()
		case <-ticker.C:
			body, status, err := r.request(ctx, fmt.Sprintf("/worlds/%d/join", realmID))
			if err != nil {
				switch status {
				case 503:
					continue
				case 404:
					return RealmAddress{}, ErrRealmNotFound
				case 403:
					return RealmAddress{}, ErrPlayerNotInRealm
				}
				return RealmAddress{}, err
			}

			var address RealmAddress
			if err := json.Unmarshal(body, &address); err != nil {
				return RealmAddress{}, err
			}
			return address, nil
		}
	}
}

func (r *Realm) OnlinePlayers(ctx context.Context) (players []Player, err error) {
	body, status, err := r.client.request(ctx, fmt.Sprintf("/worlds/%d", r.ID))
	if err != nil {
		switch status {
		case 403:
			return nil, ErrPlayerNotInRealm
		case 404:
			return nil, ErrRealmNotFound
		}
		return nil, err
	}

	var response Realm
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Players, nil
}

func (r *Realm) UpdateStorySettings(ctx context.Context, settings StorySettings) error {
	if r.client == nil {
		return fmt.Errorf("realm client is nil")
	}
	return r.client.UpdateStorySettings(ctx, r.ID, settings)
}

func (r *Realm) StorySettings(ctx context.Context) (StorySettings, error) {
	if r.client == nil {
		return StorySettings{}, fmt.Errorf("realm client is nil")
	}
	return r.client.StorySettings(ctx, r.ID)
}

func (r *Realm) OptInToStoryTimeline(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("realm client is nil")
	}
	return r.client.OptInToStoryTimeline(ctx, r.ID)
}

func (r *Client) StorySettings(ctx context.Context, realmID int) (StorySettings, error) {
	body, _, err := r.request(ctx, fmt.Sprintf("/worlds/%d/stories/settings", realmID))
	if err != nil {
		return StorySettings{}, err
	}
	var settings StorySettings
	if err := json.Unmarshal(body, &settings); err != nil {
		return StorySettings{}, err
	}
	return settings, nil
}

func (r *Client) UpdateStorySettings(ctx context.Context, realmID int, settings StorySettings) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, _, err = r.requestWithMethod(
		ctx,
		http.MethodPost,
		realmsBaseURL,
		fmt.Sprintf("/worlds/%d/stories/settings", realmID),
		body,
	)
	return err
}

func (r *Client) OptInToStoryTimeline(ctx context.Context, realmID int) error {
	settings, err := r.StorySettings(ctx, realmID)
	if err != nil {
		return err
	}
	if settings.PlayerOptIn == StoryOptInOptIn {
		return nil
	}
	settings.PlayerOptIn = StoryOptInOptIn
	return r.UpdateStorySettings(ctx, realmID, settings)
}

func (r *Client) xboxToken(ctx context.Context) (*auth.XBLToken, error) {
	if r.xblToken != nil && r.xblToken.Valid() {
		return r.xblToken, nil
	}
	if r.tokenSrc == nil {
		return nil, fmt.Errorf("token source is nil")
	}

	t, err := r.tokenSrc.Token()
	if err != nil {
		return nil, err
	}

	r.xblToken, err = auth.RequestXBLToken(ctx, t, realmsRelyingParty)
	return r.xblToken, err
}

func (r *Client) request(ctx context.Context, path string) (body []byte, status int, err error) {
	return r.requestWithMethod(ctx, http.MethodGet, realmsBaseURL, path, nil)
}

func (r *Client) requestWithMethod(ctx context.Context, method, baseURL, path string, requestBody []byte) (body []byte, status int, err error) {
	sent := r.clientVersion()
	body, status, err = r.sendWithMethod(ctx, method, baseURL, path, requestBody, sent)
	if !unknownClientVersion(status, body) {
		return body, status, err
	}
	version, retry := r.negotiateClientVersion(ctx, sent)
	if !retry {
		return body, status, err
	}
	return r.sendWithMethod(ctx, method, baseURL, path, requestBody, version)
}

func (r *Client) sendWithMethod(ctx context.Context, method, baseURL, path string, requestBody []byte, clientVersion string) (body []byte, status int, err error) {
	if path == "" {
		return nil, 0, fmt.Errorf("path is empty")
	}
	if path[0] != '/' {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(requestBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "MCPE/UWP")
	req.Header.Set("Client-Version", clientVersion)
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	xbl, err := r.xboxToken(ctx)
	if err != nil {
		return nil, 0, err
	}
	xbl.SetAuthHeader(req)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("HTTP Error: %d", resp.StatusCode)
	}

	return body, resp.StatusCode, nil
}

func (c *Client) Realm(ctx context.Context, code string) (Realm, error) {
	body, _, err := c.request(ctx, fmt.Sprintf("/worlds/v1/link/%s", code))
	if err != nil {
		return Realm{}, err
	}

	var realm Realm
	if err := json.Unmarshal(body, &realm); err != nil {
		return Realm{}, err
	}
	realm.client = c

	return realm, nil
}

func (c *Client) Realms(ctx context.Context) ([]Realm, error) {
	body, _, err := c.request(ctx, "/worlds")
	if err != nil {
		return nil, err
	}

	var response struct {
		Servers []Realm `json:"servers"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	realms := response.Servers
	for i := range realms {
		realms[i].client = c
	}

	return realms, nil
}
