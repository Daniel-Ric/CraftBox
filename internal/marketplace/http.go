package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var titlePattern = regexp.MustCompile(`^[a-fA-F0-9]{3,16}$`)
var requestSlots = make(chan struct{}, 8)

type CatalogError struct {
	Operation string
	Status    int
}

func (e *CatalogError) Error() string {
	return fmt.Sprintf("Catalog %s failed (HTTP %d)", e.Operation, e.Status)
}

type API interface {
	Call(context.Context, string, string, any) (json.RawMessage, error)
}
type HTTPAPI struct {
	Client  *http.Client
	Token   func(context.Context, string) (string, error)
	BaseURL func(string) string
}

func (a HTTPAPI) Call(ctx context.Context, title, operation string, body any) (json.RawMessage, error) {
	select {
	case requestSlots <- struct{}{}:
		defer func() { <-requestSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if !titlePattern.MatchString(title) {
		return nil, errors.New("Invalid catalog title")
	}
	if operation != "SearchItems" && operation != "GetItem" && operation != "GetItems" {
		return nil, errors.New("Unsupported catalog operation")
	}
	raw, e := json.Marshal(body)
	if e != nil {
		return nil, e
	}
	if len(raw) > 128<<10 {
		return nil, errors.New("Catalog request is too large")
	}
	token, e := a.Token(ctx, title)
	if e != nil {
		return nil, e
	}
	if token == "" {
		return nil, errors.New("PlayFab sign-in is unavailable")
	}
	base := "https://" + strings.ToLower(title) + ".playfabapi.com"
	if a.BaseURL != nil {
		base = a.BaseURL(title)
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, base+"/Catalog/"+operation, bytes.NewReader(raw))
		if e != nil {
			return nil, e
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Language", "en-US")
		req.Header.Set("X-EntityToken", token)
		req.Header.Set("Cache-Control", "no-cache")
		resp, e := client.Do(req)
		if e != nil {
			return nil, fmt.Errorf("Catalog connection failed: %w", e)
		}
		content, e := io.ReadAll(io.LimitReader(resp.Body, 16<<20+1))
		resp.Body.Close()
		if e != nil {
			return nil, e
		}
		if len(content) > 16<<20 {
			return nil, errors.New("Catalog response exceeds 16 MiB")
		}
		if (resp.StatusCode == 429 || resp.StatusCode == 503) && attempt == 0 {
			delay := time.Second
			if h := resp.Header.Get("Retry-After"); h != "" {
				if d, e := time.ParseDuration(h + "s"); e == nil && d >= 0 && d <= 3*time.Second {
					delay = d
				} else {
					return nil, fmt.Errorf("Catalog temporarily unavailable (HTTP %d). Retry later", resp.StatusCode)
				}
			}
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if resp.StatusCode != 200 {
			return nil, &CatalogError{Operation: operation, Status: resp.StatusCode}
		}
		var envelope struct {
			Code  int             `json:"code"`
			Error string          `json:"error"`
			Data  json.RawMessage `json:"data"`
		}
		if json.Unmarshal(content, &envelope) != nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return nil, errors.New("Catalog returned an invalid response")
		}
		if envelope.Error != "" || envelope.Code >= 400 {
			return nil, errors.New("Catalog rejected the request")
		}
		return envelope.Data, nil
	}
	return nil, errors.New("Catalog is temporarily unavailable")
}
