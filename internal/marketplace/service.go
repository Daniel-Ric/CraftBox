package marketplace

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	TitleID         string
	API             API
	Discover        func(context.Context) ([]json.RawMessage, error)
	CreatorRegistry func(context.Context) (map[string]string, error)
	Observe         func(string, map[string]any)
	Version         string
	mu              sync.Mutex
	sources         map[string]Source
	known           map[string]map[string]Item
	revision        string
	refreshSequence uint64
}

func New(api API, discovery func(context.Context) ([]json.RawMessage, error), version string) *Service {
	return &Service{API: api, Discover: discovery, Version: version, sources: map[string]Source{}, known: map[string]map[string]Item{}}
}
func (s *Service) Sources(ctx context.Context) (Discovery, error) {
	s.mu.Lock()
	s.refreshSequence++
	sequence := s.refreshSequence
	s.mu.Unlock()
	raw, e := s.Discover(ctx)
	if e != nil {
		return Discovery{}, e
	}
	if len(raw) > 256 {
		return Discovery{}, errors.New("Discovery returned more than 256 sources")
	}
	registry := map[string]string{}
	if s.CreatorRegistry != nil {
		var err error
		registry, err = s.CreatorRegistry(ctx)
		if err != nil && s.Observe != nil {
			s.Observe("creator_registry_unavailable", map[string]any{"error": err.Error()})
		}
	}
	out := Discovery{Sources: []Source{}, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	sources := map[string]Source{}
	known := map[string]map[string]Item{}
	for _, r := range raw {
		m, e := decode(r)
		if e != nil {
			continue
		}
		if str(m, "ContentType") != "3PP_V2.0" {
			continue
		}
		dp := object(field(m, "DisplayProperties"))
		host := str(dp, "url")
		port, ok := integer(field(dp, "port"))
		experience := str(dp, "experienceId")
		kind := "experience"
		address := ""
		if host != "" && ok && port > 0 && port <= 65535 {
			kind = "featured"
			address = net.JoinHostPort(host, fmt.Sprint(port))
		} else if experience == "" {
			continue
		}
		title := first(s.TitleID, "20CA2")
		if !titlePattern.MatchString(title) {
			continue
		}
		overview, e := Normalize(r, "20CA2")
		if e != nil {
			continue
		}
		// Discovery is published separately from the creator's offers. Its
		// CreatorEntity may identify the publisher of the listing, while the
		// originalCreatorId explicitly identifies the Marketplace creator.
		if original := str(dp, "originalCreatorId"); original != "" {
			if !strings.Contains(original, "!") {
				original = "master_player_account!" + original
			}
			overview.CreatorID = original
		}
		if registered := registry[overview.Creator]; registered != "" {
			overview.CreatorID = registered
			technical, _ := decode(overview.Technical)
			technical["CreatorMapping"] = map[string]string{"creatorId": registered, "source": "Minecraft store session configuration", "displayName": overview.Creator}
			overview.Technical, _ = json.MarshalIndent(technical, "", "  ")
		}
		if s.Observe != nil {
			s.Observe("source_mapping", map[string]any{"source": overview.Name, "creatorId": overview.CreatorID, "registryMatch": registry[overview.Creator] != ""})
		}
		id := kind + ":" + overview.ID
		source := Source{ID: id, Kind: kind, Name: overview.Name, Address: address, ExperienceID: experience, TitleID: strings.ToUpper(title), StorePageID: first(str(dp, "storePageId"), str(dp, "storeId")), CreatorID: overview.CreatorID, Creator: overview.Creator, Overview: overview}
		source.Overview.Provenance = []Provenance{{Source: id, Relation: "discovery", ID: overview.ID, Description: "Published 3PP_V2.0 discovery entry"}}
		sources[id] = source
		known[id] = map[string]Item{overview.ID: source.Overview}
		out.Sources = append(out.Sources, source)
	}
	sort.Slice(out.Sources, func(i, j int) bool {
		return strings.ToLower(out.Sources[i].Name) < strings.ToLower(out.Sources[j].Name)
	})
	hash := sha256.New()
	for _, source := range out.Sources {
		b, _ := json.Marshal(source)
		hash.Write(b)
	}
	out.Revision = hex.EncodeToString(hash.Sum(nil))[:20]
	s.mu.Lock()
	if sequence != s.refreshSequence {
		s.mu.Unlock()
		return Discovery{}, errors.New("A newer discovery refresh replaced this request")
	}
	if s.revision == out.Revision {
		for id, entries := range s.known {
			if _, ok := sources[id]; ok {
				known[id] = entries
				known[id][sources[id].Overview.ID] = sources[id].Overview
			}
		}
	}
	s.sources = sources
	s.known = known
	s.revision = out.Revision
	s.mu.Unlock()
	return out, nil
}
func (s *Service) Source(id string) (Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.sources[id]
	if !ok {
		return Source{}, errors.New("Source is not in the current discovery list. Refresh Marketplace")
	}
	return source, nil
}
func (s *Service) remember(source Source, items []Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.sources[source.ID]; !ok || !sameBinding(current, source) {
		return
	}
	known := s.known[source.ID]
	if known == nil {
		return
	}
	for _, item := range items {
		if len(known) >= 6000 {
			break
		}
		known[item.ID] = Item{ID: item.ID, TitleID: item.TitleID, Name: item.Name, Type: item.Type, Kind: item.Kind, CreatorID: item.CreatorID, IDs: item.IDs, Tags: item.Tags, Provenance: item.Provenance}
	}
}
func sameBinding(a, b Source) bool {
	return a.ID == b.ID && a.TitleID == b.TitleID && a.StorePageID == b.StorePageID && a.CreatorID == b.CreatorID && a.ExperienceID == b.ExperienceID && a.Address == b.Address
}
func (s *Service) Known(sourceID, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.known[sourceID][id]
	if !ok {
		return Item{}, errors.New("Item has not been returned for this discovery source")
	}
	return item, nil
}
func (s *Service) resolve(ctx context.Context, source Source, id string) (Item, error) {
	raw, e := s.API.Call(ctx, source.TitleID, "GetItem", map[string]any{"Id": id})
	if e != nil {
		return Item{}, e
	}
	var result struct{ Item json.RawMessage }
	if json.Unmarshal(raw, &result) != nil || len(result.Item) == 0 {
		return Item{}, errors.New("Missing catalog item")
	}
	item, e := Normalize(result.Item, source.TitleID)
	if e == nil && item.ID != id {
		return Item{}, errors.New("Catalog returned a different item ID")
	}
	if e == nil {
		Compatible(&item, s.Version)
	}
	return item, e
}
func (s *Service) Detail(ctx context.Context, sourceID, id string) (Item, error) {
	source, e := s.Source(sourceID)
	if e != nil {
		return Item{}, e
	}
	previous, e := s.Known(sourceID, id)
	if e != nil {
		return Item{}, e
	}
	if id == source.Overview.ID && previous.TitleID == "20CA2" {
		source.TitleID = "20CA2"
	}
	item, e := s.resolve(ctx, source, id)
	if e != nil {
		return Item{}, e
	}
	if previous.Type != "3PP_V2.0" && previous.Kind != "store" && !(isCollection(item) && collectionMatches(item, source)) && !IsServerOffer(item) {
		return Item{}, errors.New("This item no longer has the required 3PServerContent tag")
	}
	if IsServerOffer(item) {
		for _, association := range previous.Provenance {
			if association.Relation == "creator" && source.CreatorID != "" && item.CreatorID != source.CreatorID {
				return Item{}, errors.New("Offer is no longer associated with this discovery creator")
			}
			root, knownErr := s.Known(sourceID, association.ID)
			if association.Relation != "offers" || knownErr != nil || root.Kind != "store" {
				continue
			}
			raw, err := s.API.Call(ctx, source.TitleID, "SearchItems", map[string]any{"Count": 1, "Language": "en-US", "Select": "title,description,images,contents", "Store": map[string]string{"Id": root.ID}, "Filter": "tags/any(t:t eq '3PServerContent') and id eq " + quote(item.ID)})
			if err != nil {
				return Item{}, err
			}
			var result struct{ Items []json.RawMessage }
			if json.Unmarshal(raw, &result) != nil {
				return Item{}, errors.New("Invalid store price response")
			}
			found := false
			for _, row := range result.Items {
				scoped, err := Normalize(row, source.TitleID)
				if err == nil && scoped.ID == item.ID && IsServerOffer(scoped) {
					item.Prices = scoped.Prices
					item.Free = scoped.Free
					item.Sale = scoped.Sale
					found = true
					technical, _ := decode(item.Technical)
					technical["StoreContext"] = map[string]any{"Id": root.ID, "PriceOptions": sanitize(object(fieldMust(row, "PriceOptions")), "", 0)}
					item.Technical, _ = json.MarshalIndent(technical, "", "  ")
				}
			}
			if !found {
				return Item{}, errors.New("Offer is no longer in this published store")
			}
		}
	}
	item.Provenance = previous.Provenance
	if current, err := s.Source(sourceID); err != nil || !sameBinding(current, source) && previous.Type != "3PP_V2.0" {
		return Item{}, errors.New("Discovery association changed. Refresh the offer")
	}
	s.remember(source, []Item{item})
	return item, nil
}
func fieldMust(raw json.RawMessage, key string) any { m, _ := decode(raw); return field(m, key) }

type cursor struct {
	Revision    string            `json:"revision"`
	Fingerprint string            `json:"fingerprint"`
	Next        map[string]string `json:"next"`
}

func queryKey(q Query) string {
	q.Cursor = ""
	q.Count = 0
	b, _ := json.Marshal(q)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])[:24]
}
func (s *Service) Search(ctx context.Context, q Query) (Page, error) {
	if len(q.Search) > 200 || len(q.Cursor) > 128<<10 || len(q.Type) > 120 || len(q.Tag) > 120 || len(q.Currency) > 200 {
		return Page{}, errors.New("Search value is too long")
	}
	if q.Relation == "" {
		q.Relation = "offers"
	}
	if q.Relation != "offers" && q.Relation != "creator" && q.Relation != "collections" {
		return Page{}, errors.New("Unknown marketplace relation")
	}
	if q.Count <= 0 {
		q.Count = 24
	}
	if q.Count > 50 {
		q.Count = 50
	}
	if q.MaxPrice != nil && *q.MaxPrice < 0 {
		return Page{}, errors.New("Maximum price must be positive")
	}
	if q.MaxPrice != nil && q.Currency == "" {
		return Page{}, errors.New("Select a currency before setting a maximum price")
	}
	if (q.Sort == "price-asc" || q.Sort == "price-desc") && q.Currency == "" {
		return Page{}, errors.New("Select a currency before sorting prices")
	}
	s.mu.Lock()
	revision := s.revision
	selected := []Source{}
	for id, source := range s.sources {
		if (q.Source == "all" || q.Source == id) && (q.Creator == "" || q.Creator == source.CreatorID) {
			selected = append(selected, source)
		}
	}
	s.mu.Unlock()
	if len(selected) == 0 {
		return Page{}, errors.New("Select a current discovery source")
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	key := queryKey(q)
	state := cursor{Revision: revision, Fingerprint: key, Next: map[string]string{}}
	if q.Cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(q.Cursor)
		if e != nil || json.Unmarshal(raw, &state) != nil || state.Revision != revision || state.Fingerprint != key {
			return Page{}, errors.New("Search changed or discovery refreshed. Start the search again")
		}
	}
	page := Page{Items: []Item{}, Warnings: []string{}, Relation: q.Relation, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	if q.Relation == "collections" {
		return s.searchCollections(ctx, q, selected, state, page)
	}
	next := cursor{Revision: revision, Fingerprint: key, Next: map[string]string{}}
	type outcome struct {
		source  Source
		items   []Item
		token   string
		warning string
		err     error
	}
	results := make(chan outcome, len(selected))
	slots := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, source := range selected {
		source := source
		if q.Cursor != "" {
			if _, ok := state.Next[source.ID]; !ok {
				continue
			}
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				results <- outcome{source: source, err: ctx.Err()}
				return
			}
			count := q.Count
			if len(selected) > 1 {
				count = max(1, min(12, 48/len(selected)))
			}
			items, token, warning, e := s.searchSource(ctx, source, q, state.Next[source.ID], count)
			results <- outcome{source, items, token, warning, e}
		}()
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for result := range results {
		if result.err != nil {
			page.Warnings = append(page.Warnings, result.source.Name+": "+result.err.Error())
			continue
		}
		succeeded++
		if result.warning != "" {
			page.Warnings = append(page.Warnings, result.source.Name+": "+result.warning)
		}
		if result.token != "" {
			next.Next[result.source.ID] = result.token
		}
		s.remember(result.source, result.items)
		for _, item := range result.items {
			if matches(item, q) {
				page.Items = append(page.Items, item)
			}
		}
	}
	if succeeded == 0 && len(page.Warnings) > 0 {
		return Page{}, errors.New(strings.Join(page.Warnings, "; "))
	}
	if ctx.Err() != nil {
		return Page{}, ctx.Err()
	}
	s.mu.Lock()
	currentRevision := s.revision
	s.mu.Unlock()
	if currentRevision != revision {
		return Page{}, errors.New("Discovery changed while searching. Search again")
	}
	dedup := map[string]int{}
	unique := []Item{}
	for _, item := range page.Items {
		id := item.TitleID + ":" + item.ID
		if len(item.Provenance) > 0 {
			id += ":" + item.Provenance[0].Source
		}
		if index, ok := dedup[id]; ok {
			unique[index].Provenance = append(unique[index].Provenance, item.Provenance...)
		} else {
			dedup[id] = len(unique)
			unique = append(unique, item)
		}
	}
	page.Items = unique
	sort.SliceStable(page.Items, func(i, j int) bool {
		a, b := page.Items[i], page.Items[j]
		switch q.Sort {
		case "name":
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "rating":
			if a.Rating != nil && b.Rating != nil {
				return *a.Rating > *b.Rating
			}
			return a.Rating != nil
		case "popular":
			return a.Ratings > b.Ratings
		case "recent":
			return a.Start > b.Start
		case "price-asc", "price-desc":
			av, aok := comparablePrice(a, q.Currency)
			bv, bok := comparablePrice(b, q.Currency)
			if aok != bok {
				return aok
			}
			if !aok {
				return a.Name < b.Name
			}
			if q.Sort == "price-desc" {
				return av > bv
			}
			return av < bv
		}
		return a.Name < b.Name
	})
	if len(next.Next) > 0 {
		raw, _ := json.Marshal(next)
		page.Cursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, nil
}
func comparablePrice(item Item, currency string) (int64, bool) {
	var value int64
	found := false
	for _, p := range item.Prices {
		if len(p.Amounts) != 1 {
			continue
		}
		a := p.Amounts[0]
		if a.Currency != currency {
			continue
		}
		if !found || a.Value < value {
			value = a.Value
			found = true
		}
	}
	return value, found
}
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func (s *Service) searchSource(ctx context.Context, source Source, q Query, token string, count int) ([]Item, string, string, error) {
	provenance := Provenance{Source: source.ID, Relation: q.Relation}
	creatorFallback := func(reason string) ([]Item, string, string, error) {
		q.Relation = "creator"
		items, next, warning, err := s.searchSource(ctx, source, q, token, count)
		if warning != "" {
			reason += "; " + warning
		}
		return items, next, reason, err
	}
	if q.Relation == "offers" && source.StorePageID == "" {
		q.Relation = "creator"
		provenance.Relation = "creator"
	}
	var root Item
	var e error
	if q.Relation != "creator" {
		if source.StorePageID == "" {
			return []Item{}, "", "This discovery entry does not publish a store or collection. More by creator is separate", nil
		}
		root, e = s.resolvePublishedPage(ctx, source)
		if e != nil {
			if q.Relation == "offers" && source.CreatorID != "" {
				return creatorFallback("Published store page is unavailable; showing tagged offers from the exact discovery creator")
			}
			return nil, "", "", fmt.Errorf("Published store page unavailable: %w", e)
		}
		root.Provenance = []Provenance{{Source: source.ID, Relation: "collection", ID: source.StorePageID, Description: "storePageId published in discovery"}}
		s.remember(source, []Item{root})
		if q.Relation == "collections" {
			return []Item{root}, "", "", nil
		}
		provenance.ID = root.ID
		provenance.Description = "Membership in the published discovery store/collection"
		if root.Kind != "store" {
			if len(root.References) == 0 && source.CreatorID != "" {
				return creatorFallback("Published page has no item membership; showing tagged offers from the exact discovery creator")
			}
			if token != "" {
				return nil, "", "", errors.New("Collection has no continuation")
			}
			items := []Item{}
			if len(root.References) == 0 {
				return items, "", "Published page has no item references", nil
			}
			ids := []string{}
			for _, ref := range root.References {
				if len(ids) >= 50 {
					break
				}
				ids = append(ids, ref.ID)
			}
			raw, e := s.API.Call(ctx, source.TitleID, "GetItems", map[string]any{"Ids": ids})
			if e != nil {
				return nil, "", "", e
			}
			var result struct{ Items []json.RawMessage }
			if json.Unmarshal(raw, &result) != nil {
				return nil, "", "", errors.New("Invalid collection response")
			}
			allowed := map[string]bool{}
			for _, id := range ids {
				allowed[id] = true
			}
			for _, r := range result.Items {
				item, e := Normalize(r, source.TitleID)
				if e != nil || !allowed[item.ID] || !IsServerOffer(item) {
					continue
				}
				Compatible(&item, s.Version)
				item.Provenance = []Provenance{provenance}
				items = append(items, item)
			}
			warning := ""
			if len(root.References) > 50 {
				warning = "Showing the first 50 published collection references"
			}
			return items, "", warning, nil
		}
	}
	body := map[string]any{"Count": count, "Language": "en-US", "Select": "title,description,images,contents,keywords", "Search": q.Search}
	filters := []string{"tags/any(t:t eq '3PServerContent')"}
	if q.Relation == "creator" {
		if source.CreatorID == "" {
			return []Item{}, "", "No creator entity ID is published by this discovery entry", nil
		}
		filters = append(filters, "creatorId eq "+quote(source.CreatorID))
		provenance.ID = source.CreatorID
		provenance.Description = "Exact discovery creator entity and mandatory 3PServerContent tag"
	} else {
		body["Store"] = map[string]string{"Id": root.ID}
	}
	if q.Type != "" {
		filters = append(filters, "ContentType eq "+quote(q.Type))
	}
	if q.Tag != "" {
		filters = append(filters, "tags/any(t:t eq "+quote(q.Tag)+")")
	}
	if len(filters) > 0 {
		body["Filter"] = strings.Join(filters, " and ")
	}
	if token != "" {
		body["ContinuationToken"] = token
	}
	switch q.Sort {
	case "recent":
		body["OrderBy"] = "startDate desc"
	case "rating":
		body["OrderBy"] = "rating/average desc"
	case "popular":
		body["OrderBy"] = "rating/totalCount desc"
	}
	raw, e := s.API.Call(ctx, source.TitleID, "SearchItems", body)
	if e != nil {
		return nil, "", "", e
	}
	var result struct {
		Items             []json.RawMessage
		ContinuationToken string
	}
	if json.Unmarshal(raw, &result) != nil || len(result.Items) > 50 || len(result.ContinuationToken) > 8192 {
		return nil, "", "", errors.New("Invalid search response")
	}
	if token != "" && result.ContinuationToken == token {
		return nil, "", "", errors.New("Catalog repeated its continuation token")
	}
	items := []Item{}
	for _, r := range result.Items {
		item, e := Normalize(r, source.TitleID)
		if e != nil || !IsServerOffer(item) {
			continue
		}
		Compatible(&item, s.Version)
		item.Provenance = []Provenance{provenance}
		items = append(items, item)
	}
	if s.Observe != nil {
		s.Observe("offer_search", map[string]any{"source": source.Name, "titleId": source.TitleID, "creatorId": source.CreatorID, "relation": q.Relation, "rawItems": len(result.Items), "taggedOffers": len(items)})
	}
	return items, result.ContinuationToken, "", nil
}
func IsServerOffer(item Item) bool {
	for _, tag := range item.Tags {
		if tag == "3PServerContent" {
			return true
		}
	}
	return false
}

func matches(item Item, q Query) bool {
	if q.Type != "" && item.Type != q.Type {
		return false
	}
	if q.Compatible && item.Compatibility != "compatible" {
		return false
	}
	if q.Price == "free" && !item.Free || q.Price == "paid" && (len(item.Prices) == 0 || item.Free) || q.Price == "sale" && !item.Sale {
		return false
	}
	if q.Tag != "" {
		ok := false
		for _, t := range item.Tags {
			if t == q.Tag {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if q.Currency != "" || q.MaxPrice != nil {
		found := false
		for _, p := range item.Prices {
			if len(p.Amounts) != 1 {
				continue
			}
			a := p.Amounts[0]
			if q.Currency != "" && a.Currency != q.Currency {
				continue
			}
			if q.MaxPrice != nil && a.Value > *q.MaxPrice {
				continue
			}
			found = true
		}
		if !found {
			return false
		}
	}
	if q.Search != "" {
		hay := strings.ToLower(item.Name + " " + item.Description + " " + item.Creator + " " + strings.Join(item.Tags, " ") + " " + strings.Join(item.IDs, " "))
		for _, word := range strings.Fields(strings.ToLower(q.Search)) {
			if !strings.Contains(hay, word) {
				return false
			}
		}
	}
	return true
}
