package sourceprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type Result struct {
	Discovery       string `json:"discovery,omitempty"`
	NetworkProtocol string `json:"networkProtocol,omitempty"`
	Endpoint        string `json:"endpoint"`
	Reachable       bool   `json:"reachable"`
	Name            string `json:"name"`
	Protocol        *int   `json:"protocol,omitempty"`
	Version         string `json:"version,omitempty"`
	Level           string `json:"level,omitempty"`
	Players         *int   `json:"players,omitempty"`
	MaxPlayers      *int   `json:"maxPlayers,omitempty"`
	GameType        *int   `json:"gameType,omitempty"`
	OnlineAuth      *bool  `json:"onlineAuth,omitempty"`
	SelfSignedAuth  *bool  `json:"selfSignedAuth,omitempty"`
	Editor          *bool  `json:"editor,omitempty"`
	Hardcore        *bool  `json:"hardcore,omitempty"`
	TransportLayer  *int   `json:"transportLayer,omitempty"`
	Connection      *int   `json:"connection,omitempty"`
	DataVersion     *int   `json:"dataVersion,omitempty"`
	Compatible      *bool  `json:"compatible,omitempty"`
	LatencyMS       int64  `json:"latencyMs"`
	Transport       string `json:"transport"`
	TransportError  string `json:"transportError,omitempty"`
	Error           string `json:"error,omitempty"`
	CheckedAt       string `json:"checkedAt"`
}

func Candidates(address string) ([]string, error) {
	address = strings.TrimSpace(address)
	explicit := strings.Contains(address, "://")
	if !explicit {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil {
		return nil, errors.New("Invalid server address")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(u.Hostname(), " \t\r\n") {
		return nil, errors.New("Enter a hostname or IP with an optional port, or an HTTP/HTTPS endpoint")
	}
	port := u.Port()
	if port == "" {
		port = "19132"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return nil, errors.New("Port must be between 1 and 65535")
	}
	u.Host = net.JoinHostPort(u.Hostname(), port)
	u.Path = ""
	if explicit {
		return []string{u.String()}, nil
	}
	u.Scheme = "https"
	secure := u.String()
	u.Scheme = "http"
	return []string{secure, u.String()}, nil
}

func Inspect(ctx context.Context, address string, connect func(context.Context, string, string) error) Result {
	result := Result{Transport: "not-tested", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	candidates, err := Candidates(address)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	client := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var failures []string
	networkID := strconv.FormatUint(rand.Uint64(), 10)
	for _, candidate := range candidates {
		started := time.Now()
		status, err := fetch(ctx, client, candidate, networkID)
		if err != nil {
			failures = append(failures, candidate+": "+err.Error())
			if ctx.Err() != nil {
				break
			}
			continue
		}
		result = status
		result.Discovery = "http"
		result.Endpoint = candidate
		result.Reachable = true
		result.Transport = "not-tested"
		result.LatencyMS = time.Since(started).Milliseconds()
		result.CheckedAt = time.Now().UTC().Format(time.RFC3339)
		if result.Protocol != nil {
			compatible := *result.Protocol == protocol.CurrentProtocol
			result.Compatible = &compatible
		}
		if connect != nil {
			result.Transport = "connected"
			if err := connect(ctx, candidate, networkID); err != nil {
				result.Transport = "failed"
				result.TransportError = err.Error()
			}
		}
		return result
	}
	result.Error = strings.Join(failures, "; ")
	return result
}

func fetch(ctx context.Context, client *http.Client, address, networkID string) (Result, error) {
	u, _ := url.Parse(address)
	u.Path = "/v1/join"
	q := u.Query()
	q.Set("version", protocol.CurrentVersion)
	q.Set("protocol", strconv.Itoa(protocol.CurrentProtocol))
	q.Set("platform", "7")
	q.Set("id", networkID)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "libhttpclient/1.0.0.0")
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("NetherNet status: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024+1))
	if err != nil {
		return Result{}, err
	}
	if len(data) > 64*1024 {
		return Result{}, errors.New("Status response is too large")
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return r, errors.New("Response is not valid NetherNet status JSON")
	}
	if r.Name == "" && r.Protocol == nil && r.Version == "" {
		return r, errors.New("Response contains no NetherNet server metadata")
	}
	for _, n := range []*int{r.Players, r.MaxPlayers, r.Protocol, r.GameType} {
		if n != nil && *n < 0 {
			return r, errors.New("Response contains invalid negative server metadata")
		}
	}

	r.Reachable = false
	r.Compatible = nil
	r.Error = ""
	r.TransportError = ""
	r.Name = plainText(r.Name)
	r.Level = plainText(r.Level)
	return r, nil
}

func plainText(s string) string {
	var b strings.Builder
	skip := false
	for _, r := range s {
		if skip {
			skip = false
			continue
		}
		if r == '§' {
			skip = true
			continue
		}
		if r >= 32 && r != 127 {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
