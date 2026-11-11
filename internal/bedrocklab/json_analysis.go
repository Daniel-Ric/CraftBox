package bedrocklab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type FieldChange struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Before string `json:"before"`
	After  string `json:"after"`
}
type JSONAnalysis struct {
	Fields      []FieldChange `json:"fields"`
	VersionOnly bool          `json:"versionOnly"`
	Truncated   bool          `json:"truncated"`
}

func AnalyzeJSON(before, after []byte) (JSONAnalysis, error) {
	result := JSONAnalysis{Fields: []FieldChange{}}
	parse := func(b []byte) (any, error) {
		if len(b) == 0 {
			return nil, nil
		}
		canonical, err := canonicalJSON(b)
		if err != nil {
			return nil, err
		}
		var v any
		d := json.NewDecoder(bytes.NewReader(canonical))
		d.UseNumber()
		err = d.Decode(&v)
		return v, err
	}
	a, err := parse(before)
	if err != nil {
		return result, err
	}
	b, err := parse(after)
	if err != nil {
		return result, err
	}
	compact := func(v any) string {
		b, _ := json.Marshal(v)
		r := []rune(string(b))
		if len(r) > 240 {
			return string(r[:240]) + "…"
		}
		return string(r)
	}
	pointer := func(key string) string { return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1") }
	var visit func(any, bool, any, bool, string)
	visit = func(a any, hasA bool, b any, hasB bool, p string) {
		if hasA == hasB && reflect.DeepEqual(a, b) {
			return
		}
		if len(result.Fields) >= 256 {
			result.Truncated = true
			return
		}
		if hasA && hasB {
			am, aok := a.(map[string]any)
			bm, bok := b.(map[string]any)
			if aok && bok {
				keys := map[string]bool{}
				for k := range am {
					keys[k] = true
				}
				for k := range bm {
					keys[k] = true
				}
				names := make([]string, 0, len(keys))
				for k := range keys {
					names = append(names, k)
				}
				sort.Strings(names)
				for _, k := range names {
					av, ah := am[k]
					bv, bh := bm[k]
					visit(av, ah, bv, bh, p+"/"+pointer(k))
				}
				return
			}
			aa, aok := a.([]any)
			ba, bok := b.([]any)
			if aok && bok {
				for i := 0; i < max(len(aa), len(ba)); i++ {
					var av, bv any
					ah, bh := i < len(aa), i < len(ba)
					if ah {
						av = aa[i]
					}
					if bh {
						bv = ba[i]
					}
					visit(av, ah, bv, bh, fmt.Sprintf("%s/%d", p, i))
				}
				return
			}
		}
		c := FieldChange{Path: p, Status: "modified"}
		if hasA {
			c.Before = compact(a)
		} else {
			c.Status = "added"
		}
		if hasB {
			c.After = compact(b)
		} else {
			c.Status = "removed"
		}
		result.Fields = append(result.Fields, c)
	}
	visit(a, len(before) > 0, b, len(after) > 0, "")
	result.VersionOnly = len(result.Fields) == 1 && result.Fields[0].Path == "/format_version" && !result.Truncated
	return result, nil
}
