package marketplace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
)

// Discovery pages can carry a FriendlyId, which is not the catalog item ID.
func (s *Service) resolvePublishedPage(ctx context.Context, source Source) (Item, error) {
	item, err := s.resolve(ctx, source, source.StorePageID)
	if err == nil {
		return item, nil
	}
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || (catalogErr.Status != 400 && catalogErr.Status != 404) {
		return Item{}, err
	}
	raw, err := s.API.Call(ctx, source.TitleID, "GetItem", map[string]any{"AlternateId": map[string]string{"Type": "FriendlyId", "Value": source.StorePageID}})
	if err != nil {
		return Item{}, err
	}
	var result struct{ Item json.RawMessage }
	if json.Unmarshal(raw, &result) != nil || len(result.Item) == 0 {
		return Item{}, errors.New("Missing published collection")
	}
	item, err = Normalize(result.Item, source.TitleID)
	if err != nil {
		return Item{}, err
	}
	m, _ := decode(result.Item)
	valid := false
	for _, value := range list(field(m, "AlternateIds")) {
		a := object(value)
		if str(a, "Type") == "FriendlyId" && str(a, "Value") == source.StorePageID {
			valid = true
		}
	}
	if !valid {
		return Item{}, errors.New("Catalog returned a different published page")
	}
	Compatible(&item, s.Version)
	return item, nil
}
func isCollection(item Item) bool {
	return item.Type == "OfferCollectionQueries_V3.0" || strings.EqualFold(item.Kind, "store")
}
func collectionMatches(item Item, source Source) bool {
	for _, id := range item.IDs {
		if source.StorePageID != "" && id == source.StorePageID {
			return true
		}
	}
	if source.CreatorID == "" {
		return false
	}
	m, _ := decode(item.Technical)
	for _, id := range stringsOf(field(object(field(m, "DisplayProperties")), "creatorIds")) {
		if id == source.CreatorID {
			return true
		}
	}
	return item.CreatorID == source.CreatorID
}

// A store layout ID is not necessarily a catalog collection. Fall back to
// published collection metadata, retaining exact Discovery creator association.
func (s *Service) searchCollections(ctx context.Context, q Query, sources []Source, state cursor, page Page) (Page, error) {
	seen := map[string]bool{}
	add := func(source Source, item Item) {
		if !matches(item, q) {
			return
		}
		key := source.ID + ":" + item.ID
		if seen[key] {
			return
		}
		seen[key] = true
		item.Provenance = []Provenance{{Source: source.ID, Relation: "collection", ID: source.StorePageID, Description: "Published collection linked by exact discovery page or creator entity"}}
		s.remember(source, []Item{item})
		page.Items = append(page.Items, item)
	}
	if q.Cursor == "" {
		type published struct {
			source Source
			item   Item
		}
		resolved := make(chan published, len(sources))
		slots := make(chan struct{}, 4)
		var wg sync.WaitGroup
		for _, source := range sources {
			if source.StorePageID == "" {
				continue
			}
			wg.Add(1)
			go func(source Source) {
				defer wg.Done()
				select {
				case slots <- struct{}{}:
					defer func() { <-slots }()
				case <-ctx.Done():
					return
				}
				item, err := s.resolvePublishedPage(ctx, source)
				if err == nil {
					resolved <- published{source, item}
				}
			}(source)
		}
		wg.Wait()
		close(resolved)
		for result := range resolved {
			add(result.source, result.item)
		}
		if ctx.Err() != nil {
			return Page{}, ctx.Err()
		}
	}
	body := map[string]any{"Count": 50, "Language": "en-US", "Select": "title,description,images,displayProperties,contents,keywords", "Filter": "contentType eq 'OfferCollectionQueries_V3.0'", "OrderBy": "lastModifiedDate desc", "Search": q.Search}
	token := state.Next["collections"]
	var result struct {
		Items             []json.RawMessage
		ContinuationToken string
	}
	seenTokens := map[string]bool{}
	// Advance past irrelevant metadata pages without making users repeatedly
	// request empty pages. Work remains bounded and resumes with the catalog cursor.
	for scanned := 0; scanned < 8; scanned++ {
		if token != "" {
			body["ContinuationToken"] = token
			seenTokens[token] = true
		}
		raw, err := s.API.Call(ctx, first(s.TitleID, "20CA2"), "SearchItems", body)
		if err != nil {
			if len(page.Items) > 0 {
				result.ContinuationToken = token
				break
			}
			return Page{}, err
		}
		result.Items = nil
		result.ContinuationToken = ""
		if json.Unmarshal(raw, &result) != nil || len(result.Items) > 50 || len(result.ContinuationToken) > 8192 {
			return Page{}, errors.New("Invalid collection response")
		}
		if result.ContinuationToken != "" && seenTokens[result.ContinuationToken] {
			return Page{}, errors.New("Collection search repeated its continuation token")
		}
		for _, row := range result.Items {
			item, e := Normalize(row, first(s.TitleID, "20CA2"))
			if e != nil || item.Type != "OfferCollectionQueries_V3.0" {
				continue
			}
			Compatible(&item, s.Version)
			for _, source := range sources {
				if collectionMatches(item, source) {
					add(source, item)
				}
			}
		}
		if len(page.Items) > 0 || result.ContinuationToken == "" {
			break
		}
		token = result.ContinuationToken
	}

	s.mu.Lock()
	current := s.revision
	s.mu.Unlock()
	if current != state.Revision {
		return Page{}, errors.New("Discovery changed while searching. Search again")
	}
	if result.ContinuationToken != "" {
		state.Next = map[string]string{"collections": result.ContinuationToken}
		encoded, _ := json.Marshal(state)
		page.Cursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	sort.SliceStable(page.Items, func(i, j int) bool { return strings.ToLower(page.Items[i].Name) < strings.ToLower(page.Items[j].Name) })
	return page, nil
}
func (s *Service) CurrentSources() []Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Source, 0, len(s.sources))
	for _, source := range s.sources {
		out = append(out, source)
	}
	return out
}
