package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mp "github.com/Daniel-Ric/CraftBox/internal/marketplace"
	"github.com/Daniel-Ric/CraftBox/internal/sourceprobe"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/df-mc/go-playfab/v2"
	"github.com/df-mc/go-playfab/v2/title"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sirupsen/logrus"
)

type marketplaceSession struct {
	environment string
	account     *auth.Account
	service     *mp.Service
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	clients     map[string]*playfab.Client
}

func (a *App) marketplace(environments ...string) (*marketplaceSession, error) {
	environment := "production"
	if len(environments) > 0 && environments[0] != "" {
		environment = environments[0]
	}
	if environment != "production" && environment != "staging" {
		return nil, errors.New("Select Production or Staging")
	}
	account := auth.Auth.Account()
	if account == nil {
		return nil, errors.New("Sign in with your Microsoft account to browse Marketplace")
	}
	a.marketMu.Lock()
	defer a.marketMu.Unlock()
	if a.marketSessions == nil {
		a.marketSessions = map[string]*marketplaceSession{}
	}
	for key, old := range a.marketSessions {
		if old.account != account {
			old.cancel()
			old.mu.Lock()
			for _, client := range old.clients {
				client.Close()
			}
			old.mu.Unlock()
			delete(a.marketSessions, key)
		}
	}
	if existing := a.marketSessions[environment]; existing != nil {
		return existing, nil
	}
	ctx, cancel := context.WithCancel(a.ctx)
	session := &marketplaceSession{environment: environment, account: account, ctx: ctx, cancel: cancel, clients: map[string]*playfab.Client{}}
	api := mp.HTTPAPI{Token: func(ctx context.Context, id string) (string, error) {
		if auth.Auth.Account() != account {
			return "", errors.New("Account changed. Refresh Marketplace")
		}
		session.mu.Lock()
		defer session.mu.Unlock()
		client := session.clients[id]
		if client == nil {
			if len(session.clients) >= 64 {
				return "", errors.New("Too many discovery catalog titles")
			}
			xbl, e := account.XBLClient(ctx)
			if e != nil {
				return "", e
			}
			client, e = playfab.LoginWithXbox(ctx, title.Title(id), xbl, playfab.ClientConfig{CreateAccount: true})
			if e != nil {
				return "", e
			}
			session.clients[id] = client
		}
		token, e := client.MasterPlayerAccount().EntityToken(ctx)
		if e != nil {
			return "", e
		}
		return token.Token, nil
	}}
	session.service = mp.New(api, func(ctx context.Context) ([]json.RawMessage, error) {
		client, e := account.Gatherings(ctx)
		if e != nil {
			return nil, e
		}
		token, e := account.MCToken(ctx)
		if e != nil {
			return nil, e
		}
		items, e := client.GetCatalogItems(ctx, token)
		if e != nil {
			return nil, e
		}
		out := []json.RawMessage{}
		for _, item := range items {
			raw, e := json.Marshal(item)
			if e != nil {
				return nil, e
			}
			out = append(out, raw)
		}
		return out, nil
	}, protocol.CurrentVersion)
	session.service.CreatorRegistry = func(ctx context.Context) (map[string]string, error) {
		token, e := account.MCToken(ctx)
		if e != nil {
			return nil, errors.New("Minecraft sign-in is unavailable for creator registry")
		}
		return mp.FetchCreatorRegistry(ctx, token.AuthorizationHeader)
	}
	session.service.Observe = func(event string, fields map[string]any) {
		logrus.WithFields(logrus.Fields(fields)).WithField("part", "Marketplace").WithField("event", event).Info("Marketplace catalog diagnostic")
	}
	session.service.TitleID = "20CA2"
	if environment == "staging" {
		session.service.TitleID = "EE7B"
	}
	a.marketSessions[environment] = session
	return session, nil
}
func (s *marketplaceSession) check() error {
	if auth.Auth.Account() != s.account {
		return errors.New("Account changed. Refresh Marketplace")
	}
	return s.ctx.Err()
}
func (a *App) ListMarketplaceSources(environment string) (mp.Discovery, error) {
	session, e := a.marketplace(environment)
	if e != nil {
		return mp.Discovery{}, e
	}
	ctx, cancel := context.WithTimeout(session.ctx, 30*time.Second)
	defer cancel()
	result, e := session.service.Sources(ctx)
	if e == nil {
		e = session.check()
	}
	return result, e
}
func (a *App) SearchMarketplace(q mp.Query) (mp.Page, error) {
	session, e := a.marketplace(q.Environment)
	if e != nil {
		return mp.Page{}, e
	}
	ctx, cancel := context.WithTimeout(session.ctx, 30*time.Second)
	defer cancel()
	result, e := session.service.Search(ctx, q)
	if e == nil {
		e = session.check()
	}
	return result, e
}

// Counts load separately so status probes never delay the catalog itself.
func (a *App) GetMarketplacePlayerCounts(environment string) (map[string]int, error) {
	session, err := a.marketplace(environment)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(session.ctx, 12*time.Second)
	defer cancel()
	sources := session.service.CurrentSources()
	counts := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, 4)
	store := func(id string, n int) {
		if n >= 0 {
			mu.Lock()
			counts[id] = n
			mu.Unlock()
		}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		client, e := session.account.Gatherings(ctx)
		if e != nil {
			return
		}
		token, e := session.account.MCToken(ctx)
		if e != nil {
			return
		}
		values, e := client.GetPlayerCounts(ctx, token)
		if e != nil {
			return
		}
		for _, source := range sources {
			for id, n := range values {
				if strings.EqualFold(source.ExperienceID, id.String()) {
					store(source.ID, n)
				}
			}
		}
	}()
	for _, source := range sources {
		if source.Kind != "featured" || source.Address == "" {
			continue
		}
		wg.Add(1)
		go func(source mp.Source) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				return
			}
			probeCtx, stop := context.WithTimeout(ctx, 4*time.Second)
			defer stop()
			result := sourceprobe.Inspect(probeCtx, source.Address, nil)
			if result.Players != nil && result.Error == "" {
				store(source.ID, *result.Players)
			}
		}(source)
	}
	wg.Wait()
	if err = session.check(); err != nil {
		return nil, err
	}
	return counts, nil
}
func (a *App) GetMarketplaceCreatorImages(environment string) (map[string]mp.CreatorImage, error) {
	session, e := a.marketplace(environment)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(session.ctx, 20*time.Second)
	defer cancel()
	images, e := session.service.CreatorImages(ctx)
	if e == nil {
		e = session.check()
	}
	return images, e
}
func (a *App) GetMarketplaceItem(source, id, environment string) (mp.Item, error) {
	session, e := a.marketplace(environment)
	if e != nil {
		return mp.Item{}, e
	}
	ctx, cancel := context.WithTimeout(session.ctx, 25*time.Second)
	defer cancel()
	result, e := session.service.Detail(ctx, source, id)
	if e == nil {
		e = session.check()
	}
	return result, e
}
func (a *App) ResolveMarketplaceReference(ref mp.Favorite) (mp.Item, error) {
	session, e := a.marketplace(ref.Environment)
	if e != nil {
		return mp.Item{}, e
	}
	ctx, cancel := context.WithTimeout(session.ctx, 25*time.Second)
	defer cancel()
	item, e := session.service.FavoriteItem(ctx, ref)
	if e == nil {
		e = session.check()
	}
	return item, e
}
func (a *App) readMarketplaceFavorites() ([]mp.Favorite, error) {
	path := utils.PathData("marketplace-favorites.json")
	raw, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return []mp.Favorite{}, nil
	}
	if e != nil {
		return nil, e
	}
	if len(raw) > 256<<10 {
		return nil, errors.New("Favorites file is too large")
	}
	var favorites []mp.Favorite
	if e = json.Unmarshal(raw, &favorites); e != nil {
		return nil, e
	}
	if len(favorites) > 1000 {
		return nil, errors.New("Too many favorites")
	}
	if favorites == nil {
		favorites = []mp.Favorite{}
	}
	return favorites, nil
}
func (a *App) SetMarketplaceFavorite(source, id string, enabled bool, environment string) error {
	session, e := a.marketplace(environment)
	if e != nil {
		return e
	}
	relation := "offers"
	if enabled {
		item, err := session.service.Known(source, id)
		if err != nil {
			return err
		}
		if !mp.IsServerOffer(item) {
			return errors.New("Only 3PServerContent offers can be favorited")
		}
		for _, p := range item.Provenance {
			if p.Source == source {
				relation = p.Relation
				break
			}
		}
	}
	a.marketMu.Lock()
	defer a.marketMu.Unlock()
	favorites, e := a.readMarketplaceFavorites()
	if e != nil {
		return e
	}
	out := []mp.Favorite{}
	for _, f := range favorites {
		if f.Source != source || f.Item != id || firstMarketEnvironment(f.Environment) != session.environment {
			out = append(out, f)
		}
	}
	if enabled {
		if len(out) >= 1000 {
			return errors.New("Favorites limit reached")
		}
		out = append(out, mp.Favorite{Environment: session.environment, Source: source, Item: id, Relation: relation})
	}
	raw, _ := json.Marshal(out)
	if len(raw) > 256<<10 {
		return errors.New("Favorites exceed the identifier storage limit")
	}
	path := utils.PathData("marketplace-favorites.json")
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	temp, e := os.CreateTemp(filepath.Dir(path), "marketplace-favorites-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(temp.Name())
	if e = temp.Chmod(0600); e == nil {
		_, e = temp.Write(raw)
	}
	ce := temp.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(temp.Name(), path)
}
func (a *App) GetMarketplaceFavoriteItems(environment string) (mp.Page, error) {
	session, e := a.marketplace(environment)
	if e != nil {
		return mp.Page{}, e
	}
	favorites, e := a.GetMarketplaceFavorites(environment)
	if e != nil {
		return mp.Page{}, e
	}
	page := mp.Page{Items: []mp.Item{}, Warnings: []string{}, Relation: "favorites", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	ctx, cancel := context.WithTimeout(session.ctx, 30*time.Second)
	defer cancel()
	for i, f := range favorites {
		if i >= 100 {
			page.Warnings = append(page.Warnings, "Showing the first 100 favorites")
			break
		}
		item, e := session.service.FavoriteItem(ctx, f)
		if e != nil {
			page.Warnings = append(page.Warnings, f.Item+": "+e.Error())
			continue
		}
		if mp.IsServerOffer(item) {
			page.Items = append(page.Items, item)
		}
	}
	if e = session.check(); e != nil {
		return mp.Page{}, e
	}
	return page, nil
}

func (a *App) GetMarketplaceAssetMatches(assetID string) ([]mp.Favorite, error) {
	a.marketMu.Lock()
	sessions := []*marketplaceSession{}
	for _, session := range a.marketSessions {
		sessions = append(sessions, session)
	}
	a.marketMu.Unlock()
	if len(sessions) == 0 {
		return []mp.Favorite{}, nil
	}
	detail, e := a.assets().Detail(assetID)
	if e != nil {
		return nil, e
	}
	out := []mp.Favorite{}
	for _, session := range sessions {
		if session.check() != nil {
			continue
		}
		matches := session.service.AssetMatches(detail.Item.UUID, detail.Metadata)
		for i := range matches {
			matches[i].Environment = session.environment
		}
		out = append(out, matches...)
	}
	return out, nil
}

func (a *App) closeMarketplace() {
	a.marketMu.Lock()
	defer a.marketMu.Unlock()
	for _, old := range a.marketSessions {
		old.cancel()
		old.mu.Lock()
		for _, client := range old.clients {
			client.Close()
		}
		old.mu.Unlock()
	}
	a.marketSessions = nil
}

func marketExportName(format string) string {
	if format == "csv" {
		return "marketplace.csv"
	}
	return "marketplace.json"
}

func firstMarketEnvironment(value string) string {
	if value == "" {
		return "production"
	}
	return value
}
func (a *App) GetMarketplaceFavorites(environment string) ([]mp.Favorite, error) {
	all, e := a.readMarketplaceFavorites()
	if e != nil {
		return nil, e
	}
	out := []mp.Favorite{}
	for _, f := range all {
		f.Environment = firstMarketEnvironment(f.Environment)
		if f.Environment == firstMarketEnvironment(environment) {
			out = append(out, f)
		}
	}
	return out, nil
}
