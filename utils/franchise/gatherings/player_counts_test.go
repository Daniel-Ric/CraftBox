package gatherings

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/service"
	native "github.com/sandertv/gophertunnel/minecraft/service/gatherings"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestPlayerCountsCacheCoalescesAndProtectsResults(t *testing.T) {
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	id := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		close(entered)
		<-release
		fmt.Fprintf(w, `{"result":[{"experienceId":%q,"playerCount":27}]}`, id.String())
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	g := &Service{CatalogClient: (&native.Environment{ServiceURI: endpoint}).New(catalogToken{&service.Token{}})}
	type result struct {
		counts map[uuid.UUID]int
		err    error
	}
	results := make(chan result, 2)
	go func() { c, e := g.GetPlayerCounts(context.Background(), nil); results <- result{c, e} }()
	<-entered
	go func() { c, e := g.GetPlayerCounts(context.Background(), nil); results <- result{c, e} }()
	close(release)
	for range 2 {
		r := <-results
		if r.err != nil || r.counts[id] != 27 {
			t.Fatalf("%v, %v", r.counts, r.err)
		}
		r.counts[id] = 999
	}
	c, e := g.GetPlayerCounts(context.Background(), nil)
	if e != nil || c[id] != 27 || calls.Load() != 1 {
		t.Fatalf("cached counts=%v err=%v requests=%d", c, e, calls.Load())
	}
}

func TestPlayerCountsRateLimitCooldown(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(429)
		fmt.Fprint(w, `{"code":"TooManyRequests","message":"Too many requests."}`)
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	g := &Service{CatalogClient: (&native.Environment{ServiceURI: endpoint}).New(catalogToken{&service.Token{}})}
	for range 2 {
		if _, err := g.GetPlayerCounts(context.Background(), nil); err == nil {
			t.Fatal("expected rate limit")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("sent %d requests while rate limited", calls.Load())
	}
	if time.Until(g.playerCounts.nextFetch) < 110*time.Second {
		t.Fatal("Retry-After was ignored")
	}
}

func TestPlayerCountsRefreshAfterExpiryAndPreserveZero(t *testing.T) {
	var calls atomic.Int32
	id := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"result":[{"experienceId":%q,"playerCount":%d}]}`, id.String(), calls.Add(1)-1)
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	g := &Service{CatalogClient: (&native.Environment{ServiceURI: endpoint}).New(catalogToken{&service.Token{}})}
	counts, err := g.GetPlayerCounts(context.Background(), nil)
	if n, exists := counts[id]; err != nil || !exists || n != 0 {
		t.Fatalf("zero lost: %v %v", counts, err)
	}
	g.playerCounts.mu.Lock()
	g.playerCounts.nextFetch = time.Now().Add(-time.Second)
	g.playerCounts.mu.Unlock()
	counts, err = g.GetPlayerCounts(context.Background(), nil)
	if err != nil || counts[id] != 1 || calls.Load() != 2 {
		t.Fatalf("did not refresh: %v %v", counts, err)
	}
}

func TestPlayerCountWaiterCanCancel(t *testing.T) {
	c := playerCountCache{pending: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.get(ctx, func() (map[uuid.UUID]int, error) { t.Fatal("canceled caller fetched"); return nil, nil }); err != context.Canceled {
		t.Fatal(err)
	}
}

func TestPlayerCountsUseDiscoveredServiceInsteadOfCatalogHost(t *testing.T) {
	id := uuid.New()
	var catalogCalls atomic.Int32
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		catalogCalls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"code":"TooManyRequests","message":"Too many requests."}`)
	}))
	defer catalog.Close()
	discovered := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/dataquery/playercounts" || r.Header.Get("Authorization") != "test-count-token" {
			t.Errorf("wrong count request: %s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"result":[{"experienceId":%q,"playerCount":42}]}`, id.String())
	}))
	defer discovered.Close()
	catalogURL, _ := url.Parse(catalog.URL)
	discoveredURL, _ := url.Parse(discovered.URL)
	g := &Service{ServiceURI: discoveredURL, CatalogClient: (&native.Environment{ServiceURI: catalogURL}).New(catalogToken{&service.Token{}})}
	counts, err := g.GetPlayerCounts(context.Background(), &service.Token{AuthorizationHeader: "test-count-token"})
	if err != nil || counts[id] != 42 || catalogCalls.Load() != 0 {
		t.Fatalf("counts=%v error=%v catalogRequests=%d", counts, err, catalogCalls.Load())
	}
}
