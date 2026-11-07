package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type CreatorImage struct {
	URL          string `json:"url"`
	CollectionID string `json:"collectionId"`
}

// CreatorImages reads published creator collection thumbnails, keyed only by
// exact creator entity IDs already present in Discovery. Collections are image
// metadata, never offers, and are not added to the known-offer registry.
func (s *Service) CreatorImages(ctx context.Context) (map[string]CreatorImage, error) {
	s.mu.Lock()
	revision := s.revision
	wanted := map[string]bool{}
	for _, source := range s.sources {
		if source.CreatorID != "" {
			wanted[source.CreatorID] = true
		}
	}
	s.mu.Unlock()
	images := map[string]CreatorImage{}
	if len(wanted) == 0 {
		return images, nil
	}
	token := ""
	seen := map[string]bool{}
	for page := 0; page < 40; page++ {
		body := map[string]any{"Count": 50, "Language": "en-US", "Select": "images,displayProperties", "Filter": "contentType eq 'OfferCollectionQueries_V3.0' and tags/any(t:t eq 'collection.creator')", "OrderBy": "lastModifiedDate desc"}
		if token != "" {
			body["ContinuationToken"] = token
		}
		raw, e := s.API.Call(ctx, first(s.TitleID, "20CA2"), "SearchItems", body)
		if e != nil {
			return nil, e
		}
		var result struct {
			Items             []json.RawMessage
			ContinuationToken string
		}
		if json.Unmarshal(raw, &result) != nil || len(result.Items) > 50 || len(result.ContinuationToken) > 8192 {
			return nil, errors.New("Invalid creator image response")
		}
		for _, row := range result.Items {
			item, e := Normalize(row, first(s.TitleID, "20CA2"))
			if e != nil || item.Type != "OfferCollectionQueries_V3.0" {
				continue
			}
			tagged := false
			for _, tag := range item.Tags {
				if tag == "collection.creator" {
					tagged = true
				}
			}
			if !tagged {
				continue
			}
			url := ""
			for _, im := range item.Images {
				if strings.EqualFold(im.Type, "thumbnail") || strings.EqualFold(im.Tag, "thumbnail") {
					url = im.URL
					break
				}
			}
			if url == "" {
				continue
			}
			m, _ := decode(item.Technical)
			for _, id := range stringsOf(field(object(field(m, "DisplayProperties")), "creatorIds")) {
				if wanted[id] && images[id].URL == "" {
					images[id] = CreatorImage{URL: url, CollectionID: item.ID}
				}
			}
		}
		if result.ContinuationToken == "" || len(images) == len(wanted) {
			break
		}
		if seen[result.ContinuationToken] {
			return nil, errors.New("Creator image search repeated its continuation token")
		}
		seen[result.ContinuationToken] = true
		token = result.ContinuationToken
		if page == 39 {
			return nil, errors.New("Creator image search exceeded its page limit")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if revision != s.revision {
		return nil, errors.New("Discovery changed while loading creator images")
	}
	return images, nil
}
