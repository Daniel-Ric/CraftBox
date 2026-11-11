package gatherings

import (
	"context"
	"errors"
	"fmt"
	"github.com/df-mc/go-playfab/v2/catalog"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/service"
	native "github.com/sandertv/gophertunnel/minecraft/service/gatherings"
	"time"
)

type catalogToken struct{ token *service.Token }

func (s catalogToken) ServiceToken(context.Context) (*service.Token, error) { return s.token, nil }

func (g *Service) GetCatalog(ctx context.Context, token *service.Token) ([]FeaturedServer, error) {
	client := g.catalogClient(token)
	items, err := g.GetCatalogItems(ctx, token)
	if err != nil {
		return nil, err
	}
	out := make([]FeaturedServer, 0, len(items))
	for _, item := range items {
		server, err := client.ParseFeaturedServer(item)
		if err != nil {
			return nil, err
		}
		exp, err := client.ParseExperience(item)
		if err != nil && !server.Valid() {
			return nil, err
		}
		name := item.Title.Neutral()
		if server.Valid() {
			if name == "" {
				name = server.Host
			}
			source := FeaturedServer{Name: name, Address: server.Address(), Description: server.CreatorName}
			if err == nil && exp.Valid() {
				source.ExperienceId = exp.ID.String()
			}
			out = append(out, source)
			continue
		}
		if exp.Valid() {
			if name == "" {
				name = exp.NewsTitle
			}
			if name == "" {
				name = exp.ID.String()
			}
			out = append(out, FeaturedServer{Name: name, ExperienceId: exp.ID.String(), Description: exp.CreatorName})
			continue
		}
	}
	return out, nil
}

func (g *Service) GetCatalogItems(ctx context.Context, token *service.Token) ([]catalog.Item, error) {
	filter := catalog.SearchFilter{Filter: "ContentType eq '3PP_V2.0'", OrderBy: "startDate desc", Select: "images"}
	items := []catalog.Item{}
	seen := map[string]bool{}
	for page := 0; page < 8; page++ {
		var result *catalog.SearchResult
		var err error
		for attempt := range 3 {
			if err := context.Cause(ctx); err != nil {
				return nil, err
			}
			if attempt > 0 {
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return nil, context.Cause(ctx)
				}
			}
			result, err = g.catalogClient(token).SearchItems(ctx, filter)
			if err == nil && (len(result.Items) > 0 || filter.ContinuationToken != "" || result.ContinuationToken != "") {
				break
			}
		}
		if cause := context.Cause(ctx); cause != nil {
			return nil, cause
		}
		if err != nil {
			return nil, err
		}
		items = append(items, result.Items...)
		if result.ContinuationToken == "" {
			return items, nil
		}
		if seen[result.ContinuationToken] {
			return nil, errors.New("Discovery repeated its continuation token")
		}
		seen[result.ContinuationToken] = true
		filter.ContinuationToken = result.ContinuationToken
	}
	return nil, errors.New("Discovery exceeded its page limit")
}

func (g *Service) catalogClient(token *service.Token) *native.Client {
	if g.CatalogClient != nil {
		return g.CatalogClient
	}
	return (&native.Environment{ServiceURI: g.ServiceURI}).New(catalogToken{token})
}

func (g *Service) GetPlayerCounts(ctx context.Context, token *service.Token) (map[uuid.UUID]int, error) {
	return g.playerCounts.get(ctx, func() (map[uuid.UUID]int, error) { return g.fetchPlayerCounts(ctx, token) })
}

func (g *Service) fetchPlayerCounts(ctx context.Context, token *service.Token) (map[uuid.UUID]int, error) {
	client := g.catalogClient(token)
	if g.ServiceURI != nil {
		client = (&native.Environment{ServiceURI: g.ServiceURI}).New(catalogToken{token})
	}
	counts, err := client.PlayerCounts(ctx)
	if err != nil {
		if g.ServiceURI != nil {
			return nil, fmt.Errorf("player counts from %s: %w", g.ServiceURI.Host, err)
		}
		return nil, err
	}
	for id, n := range counts {
		if n < 0 {
			return nil, fmt.Errorf("invalid negative player count for experience %s", id)
		}
	}
	return counts, nil
}

func (g *Service) GetFeaturedServers(ctx context.Context, token *service.Token) ([]FeaturedServer, error) {
	list, err := g.GetCatalog(ctx, token)
	if err != nil {
		return nil, err
	}
	out := make([]FeaturedServer, 0)
	for _, source := range list {
		if source.Address != "" {
			out = append(out, source)
		}
	}
	return out, nil
}

func (g *Service) GetExperiences(ctx context.Context, token *service.Token) ([]FeaturedServer, error) {
	list, err := g.GetCatalog(ctx, token)
	if err != nil {
		return nil, err
	}
	out := make([]FeaturedServer, 0)
	for _, source := range list {
		if source.Address == "" && source.ExperienceId != "" {
			out = append(out, source)
		}
	}
	return out, nil
}
