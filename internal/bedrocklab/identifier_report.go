package bedrocklab

import (
	"context"
	"encoding/json"
	"sort"
)

type IdentifierReport struct {
	Added             []string `json:"added"`
	Removed           []string `json:"removed"`
	ReferencesAdded   []string `json:"referencesAdded"`
	ReferencesRemoved []string `json:"referencesRemoved"`
	Unresolved        int      `json:"unresolved"`
}

func (s *Store) IdentifierReport(ctx context.Context, before, after string) (IdentifierReport, error) {
	out := IdentifierReport{Added: []string{}, Removed: []string{}, ReferencesAdded: []string{}, ReferencesRemoved: []string{}}
	cache := map[string][]string{}
	collect := func(id string) (map[string]bool, map[string]bool, error) {
		declarations, references := map[string]bool{}, map[string]bool{}
		if id == "" {
			return declarations, references, nil
		}
		analysis, err := s.analyzeSnapshot(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		d := analysis.snapshot
		out.Unresolved += d.Unresolved
		for _, f := range d.Files {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
			for _, v := range f.Identifiers {
				references[v] = true
			}
			if f.Kind != "json" {
				continue
			}
			ids, ok := cache[f.Hash]
			if !ok {
				b, err := analysis.read(s, f, maxFile+1)
				if err != nil {
					return nil, nil, err
				}
				canonical, err := canonicalJSON(b)
				if err != nil {
					cache[f.Hash] = nil
					continue
				}
				var v any
				if err = json.Unmarshal(canonical, &v); err != nil {
					return nil, nil, err
				}
				var visit func(any)
				visit = func(v any) {
					switch x := v.(type) {
					case map[string]any:
						for k, v := range x {
							if k == "identifier" {
								if name, ok := v.(string); ok {
									ids = append(ids, name)
								}
							}
							visit(v)
						}
					case []any:
						for _, v := range x {
							visit(v)
						}
					}
				}
				visit(v)
				cache[f.Hash] = ids
			}
			for _, v := range ids {
				declarations[v] = true
			}
		}
		return declarations, references, nil
	}
	a, ar, err := collect(before)
	if err != nil {
		return out, err
	}
	b, br, err := collect(after)
	if err != nil {
		return out, err
	}
	difference := func(a, b map[string]bool) []string {
		v := []string{}
		for k := range b {
			if !a[k] {
				v = append(v, k)
			}
		}
		sort.Strings(v)
		return v
	}
	out.Added, out.Removed = difference(a, b), difference(b, a)
	out.ReferencesAdded, out.ReferencesRemoved = difference(ar, br), difference(br, ar)
	return out, nil
}
