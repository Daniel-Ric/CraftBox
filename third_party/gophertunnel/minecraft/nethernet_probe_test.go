package minecraft

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

func TestDirectNetherNetProbeMatchesOffer(t *testing.T) {
	for _, identity := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "identity"}[identity], func(t *testing.T) {
			var mu sync.Mutex
			var probeID, offerID, offer string
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method == http.MethodGet {
					query := r.URL.Query()
					probeID = query.Get("id")
					if query.Get("version") != protocol.CurrentVersion || query.Get("protocol") != "2225" || query.Get("platform") != "1" || probeID == "" {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusOK)
					return // A valid probe need not have a JSON body.
				}
				offerID = strings.TrimPrefix(r.URL.Path, "/v1/join/")
				raw, _ := io.ReadAll(r.Body)
				offer = string(raw)
				w.WriteHeader(http.StatusForbidden) // Stop before ICE, after observing the real offer.
			}))
			defer server.Close()
			n := DirectNetherNet{HTTPClient: server.Client()}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			address := strings.TrimPrefix(server.URL, "https://")
			if identity {
				key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = n.DialContextIdentity(ctx, address, "", key)
			} else {
				_, _ = n.DialContext(ctx, address)
			}
			mu.Lock()
			defer mu.Unlock()
			if probeID == "" || offerID != probeID {
				t.Fatalf("probe ID %q does not match offer ID %q", probeID, offerID)
			}
			if identity && !strings.Contains(offer, "a=identity:") {
				t.Fatal("authenticated offer lost its identity assertion")
			}
		})
	}
}

func TestDirectProbeUsesDialerClientMetadata(t *testing.T) {
	var mu sync.Mutex
	var version, platform string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodGet {
			version = r.URL.Query().Get("version")
			platform = r.URL.Query().Get("platform")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := Dialer{DisablePortFollowing: true, ClientData: login.ClientData{GameVersion: "1.26.62", DeviceOS: protocol.DeviceWin10}}
	_, _ = dialer.DialContextNetwork(ctx, DirectNetherNet{}, server.URL)
	mu.Lock()
	defer mu.Unlock()
	if version != "1.26.62" || platform != "7" {
		t.Fatalf("probe advertised %q platform %q, expected dialer's 1.26.62/Windows metadata", version, platform)
	}
}

func TestDirectExplicitEndpointKeepsPOSTOnlyCompatibility(t *testing.T) {
	var offered atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		offered.Store(true)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = (DirectNetherNet{}).DialContext(ctx, server.URL)
	if !offered.Load() {
		t.Fatal("optional probe blocked an explicitly selected POST-only server")
	}
}
