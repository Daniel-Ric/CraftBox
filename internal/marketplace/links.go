package marketplace

import "strings"

func (s *Service) AssetMatches(uuid string, metadata map[string]any) []Favorite {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Favorite{}
	for source, items := range s.known {
		for _, item := range items {
			if !IsServerOffer(item) {
				continue
			}
			ids := map[string]bool{}
			for _, id := range item.IDs {
				if len(id) >= 16 {
					ids[strings.ToLower(id)] = true
				}
			}
			if ids[strings.ToLower(uuid)] || metadataMatches(metadata, ids, 0) {
				relation := "offers"
				for _, p := range item.Provenance {
					if p.Source == source {
						relation = p.Relation
						break
					}
				}
				out = append(out, Favorite{Source: source, Item: item.ID, Relation: relation})
			}
		}
	}
	return out
}
func metadataMatches(v any, ids map[string]bool, depth int) bool {
	if depth > 24 {
		return false
	}
	switch n := v.(type) {
	case string:
		return ids[strings.ToLower(n)]
	case map[string]any:
		for _, v := range n {
			if metadataMatches(v, ids, depth+1) {
				return true
			}
		}
	case []any:
		for _, v := range n {
			if metadataMatches(v, ids, depth+1) {
				return true
			}
		}
	}
	return false
}
