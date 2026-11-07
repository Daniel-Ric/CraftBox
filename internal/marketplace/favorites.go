package marketplace

import (
	"context"
	"encoding/json"
	"errors"
)

func (s *Service) FavoriteItem(ctx context.Context, f Favorite) (Item, error) {
	source, e := s.Source(f.Source)
	if e != nil {
		return Item{}, e
	}
	if f.Item == source.Overview.ID {
		item := source.Overview
		s.remember(source, []Item{item})
		return s.Detail(ctx, f.Source, f.Item)
	}
	if f.Relation == "creator" {
		item, e := s.resolve(ctx, source, f.Item)
		if e != nil {
			return Item{}, e
		}
		if source.CreatorID == "" || item.CreatorID != source.CreatorID || !IsServerOffer(item) {
			return Item{}, errors.New("Favorite is no longer associated with this creator")
		}
		item.Provenance = []Provenance{{Source: source.ID, Relation: "creator", ID: source.CreatorID, Description: "Exact discovery creator entity and 3PServerContent tag"}}
		s.remember(source, []Item{item})
		return item, nil
	}
	if source.StorePageID == "" {
		f.Relation = "creator"
		return s.FavoriteItem(ctx, f)
	}
	root, e := s.resolvePublishedPage(ctx, source)
	if e != nil {
		return Item{}, e
	}
	root.Provenance = []Provenance{{Source: source.ID, Relation: "collection", ID: source.StorePageID, Description: "Published discovery store/collection"}}
	s.remember(source, []Item{root})
	if root.ID == f.Item {
		root.Provenance = []Provenance{{Source: source.ID, Relation: "collection", ID: source.StorePageID, Description: "Published discovery store page"}}
		s.remember(source, []Item{root})
		return root, nil
	}
	authorized := false
	var scopedPrice *Item
	if root.Kind == "store" {
		raw, e := s.API.Call(ctx, source.TitleID, "SearchItems", map[string]any{"Count": 1, "Language": "en-US", "Filter": "tags/any(t:t eq '3PServerContent') and id eq " + quote(f.Item), "Store": map[string]string{"Id": root.ID}})
		if e != nil {
			return Item{}, e
		}
		var page struct{ Items []json.RawMessage }
		if json.Unmarshal(raw, &page) != nil {
			return Item{}, errors.New("Invalid store response")
		}
		for _, r := range page.Items {
			candidate, parseErr := Normalize(r, source.TitleID)
			if parseErr == nil && candidate.ID == f.Item && IsServerOffer(candidate) {
				authorized = true
				scopedPrice = &candidate
			}
		}
	} else {
		for _, ref := range root.References {
			if ref.ID == f.Item {
				authorized = true
			}
		}
	}
	if !authorized {
		return Item{}, errors.New("Favorite is no longer in the published store/collection")
	}
	item, e := s.resolve(ctx, source, f.Item)
	if e != nil {
		return Item{}, e
	}
	if !IsServerOffer(item) {
		return Item{}, errors.New("Favorite no longer has the required 3PServerContent tag")
	}
	if scopedPrice != nil {
		item.Prices = scopedPrice.Prices
		item.Free = scopedPrice.Free
		item.Sale = scopedPrice.Sale
	}
	item.Provenance = []Provenance{{Source: source.ID, Relation: "offers", ID: root.ID, Description: "Verified membership in published discovery store/collection"}}
	s.remember(source, []Item{item})
	return item, nil
}
