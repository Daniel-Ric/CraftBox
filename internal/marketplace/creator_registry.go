package marketplace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ParseCreatorRegistry uses only exact, unambiguous names published by the
// Minecraft store configuration. Ambiguous display names never select a creator.
func ParseCreatorRegistry(raw []byte) (map[string]string, error) {
	m, e := decode(raw)
	if e != nil {
		return nil, e
	}
	filters, ok := field(object(field(m, "result")), "storeFilters").([]any)
	if !ok {
		return nil, errors.New("Creator registry has no store filters")
	}
	out := map[string]string{}
	ambiguous := map[string]bool{}
	for _, f := range filters {
		filter := object(f)
		if !strings.EqualFold(str(filter, "filterType"), "creator") {
			continue
		}
		for _, v := range list(field(filter, "toggles")) {
			toggle := object(v)
			name, id := str(toggle, "filterName"), str(toggle, "filterId")
			parts := strings.Split(id, "!")
			if name == "" || len(parts) != 2 || parts[1] == "" || (parts[0] != "master_player_account" && parts[0] != "title_player_account") {
				continue
			}
			if old := out[name]; old != "" && old != id {
				ambiguous[name] = true
			}
			out[name] = id
		}
	}
	for name := range ambiguous {
		delete(out, name)
	}
	return out, nil
}

// FetchCreatorRegistry reads the same public creator mapping as the official
// store. Its Minecraft authorization header is never exposed to the viewer.
func FetchCreatorRegistry(ctx context.Context, token string) (map[string]string, error) {
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return fetchCreatorRegistry(ctx, token, "https://store.mktpl.minecraft-services.net/api/v1.0/session/config", client)
}
func fetchCreatorRegistry(ctx context.Context, token, endpoint string, client *http.Client) (map[string]string, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US")
	resp, e := client.Do(req)
	if e != nil {
		return nil, errors.New("Creator registry connection failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Creator registry failed (HTTP %d)", resp.StatusCode)
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
	if e != nil {
		return nil, errors.New("Creator registry response failed")
	}
	if len(raw) > 4<<20 {
		return nil, errors.New("Creator registry response exceeds 4 MiB")
	}
	return ParseCreatorRegistry(raw)
}
