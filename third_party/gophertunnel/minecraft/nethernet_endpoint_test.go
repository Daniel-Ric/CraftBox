package minecraft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDirectNetherNetSelectsHTTPS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/join" {
			t.Errorf("wrong endpoint %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ServerName":"TLS server"}`))
	}))
	defer server.Close()
	n := DirectNetherNet{HTTPClient: server.Client()}
	got, err := n.resolveEndpoint(context.Background(), strings.TrimPrefix(server.URL, "https://"))
	if err != nil || got != server.URL {
		t.Fatalf("endpoint = %q, %v; want %q", got, err, server.URL)
	}
}

func TestDirectNetherNetSelectsLocalHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ServerName":"Local server"}`))
	}))
	defer server.Close()
	got, err := (DirectNetherNet{}).resolveEndpoint(context.Background(), strings.TrimPrefix(server.URL, "http://"))
	if err != nil || got != server.URL {
		t.Fatalf("endpoint = %q, %v; want %q", got, err, server.URL)
	}
}

type failingEndpointTransport struct {
	cause    error
	requests int
}

func (f *failingEndpointTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.requests++
	return nil, f.cause
}
func TestDirectNetherNetUnavailableEndpointExplainsFailure(t *testing.T) {
	cause := fmt.Errorf("connection reset: %w", io.EOF)
	transport := &failingEndpointTransport{cause: cause}
	_, err := (DirectNetherNet{HTTPClient: &http.Client{Transport: transport}}).resolveEndpoint(context.Background(), "server.example:19132")
	if err == nil || !strings.Contains(err.Error(), "No reachable NetherNet endpoint") || !errors.Is(err, cause) || transport.requests != 2 {
		t.Fatalf("error=%v, attempts=%d", err, transport.requests)
	}
}
