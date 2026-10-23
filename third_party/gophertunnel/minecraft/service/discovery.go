package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/service/internal"
)

type Environment interface {
	ServiceName() string
}

type Discovery struct {
	ServiceEnvironments map[string]map[string]json.RawMessage `json:"serviceEnvironments"`

	SupportedEnvironments map[string][]string `json:"supportedEnvironments,omitempty"`
}

const (
	ApplicationTypeDedicatedServer = "MinecraftDedicatedServer"

	ApplicationTypeMinecraftPE = "MinecraftPE"
)

var (
	discoveryURL = &url.URL{
		Scheme: "https",
		Host:   "client.discovery.minecraft-services.net",
	}

	discoveryCache   = make(map[string]*Discovery)
	discoveryCacheMu sync.Mutex
)

func (d *Discovery) Environment(env Environment) error {
	m, ok := d.ServiceEnvironments[env.ServiceName()]
	if !ok {
		return fmt.Errorf("minecraft/service: %q is not present in ServiceEnvironments", env.ServiceName())
	}
	m2, ok := m["prod"]
	if !ok {
		return fmt.Errorf("minecraft/service: %q is not present on %q in ServiceEnvironments", "prod", env.ServiceName())
	}
	return json.Unmarshal(m2, env)
}

func Default(ctx context.Context) (*Discovery, error) {
	return Discover(ctx, ApplicationTypeMinecraftPE, protocol.CurrentVersion)
}

func Discover(ctx context.Context, appType, version string) (*Discovery, error) {
	discoveryCacheMu.Lock()
	defer discoveryCacheMu.Unlock()

	requestURL := discoveryURL.JoinPath("/api/v1.0/discovery", appType, "builds", version).String()
	if d, ok := discoveryCache[requestURL]; ok {
		return d, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", internal.UserAgent)

	resp, err := auth.ContextClient(ctx).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, internal.Err(resp)
	}

	var result internal.Result[Discovery]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	d := &result.Data
	discoveryCache[requestURL] = d
	return d, nil
}
