package localworlds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/tailscale/hujson"
)

type reference struct {
	ID      string
	Version string
	Raw     json.RawMessage
}
type dependency struct {
	UUID    string          `json:"uuid"`
	Version json.RawMessage `json:"version"`
	Module  string          `json:"module_name"`
}
type manifest struct {
	Header struct {
		UUID    string          `json:"uuid"`
		Version json.RawMessage `json:"version"`
		Name    string          `json:"name"`
	} `json:"header"`
	Modules []struct {
		Type string `json:"type"`
	} `json:"modules"`
	Dependencies []dependency `json:"dependencies"`
}
type candidate struct {
	folder, kind, id, version, name string
	priority                        int
	manifest                        manifest
	protected                       bool
}

func version(raw json.RawMessage) string {
	var nums []int
	if json.Unmarshal(raw, &nums) == nil && len(nums) == 3 {
		parts := make([]string, 3)
		for i, n := range nums {
			if n < 0 {
				return ""
			}
			parts[i] = strconv.Itoa(n)
		}
		return strings.Join(parts, ".")
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		parts := strings.Split(text, ".")
		if len(parts) != 3 {
			return ""
		}
		for _, p := range parts {
			if n, e := strconv.Atoi(p); e != nil || n < 0 {
				return ""
			}
		}
		return text
	}
	return ""
}
func packID(value string) string {
	id, e := uuid.Parse(strings.TrimSpace(value))
	if e != nil || id == uuid.Nil {
		return ""
	}
	return id.String()
}
func readReferences(path string) ([]reference, error) {
	raw, err := readSmall(path, 4<<20)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	raw, err = hujson.Standardize(bytes.TrimPrefix(raw, []byte{239, 187, 191}))
	if err != nil {
		return nil, err
	}
	var values []json.RawMessage
	if err = json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	out := []reference{}
	for _, value := range values {
		var row struct {
			ID      string          `json:"pack_id"`
			Version json.RawMessage `json:"version"`
		}
		if e := json.Unmarshal(value, &row); e != nil {
			return nil, e
		}
		id, v := packID(row.ID), version(row.Version)
		if id == "" || v == "" {
			return nil, fmt.Errorf("a pack reference has an invalid ID or version")
		}
		out = append(out, reference{id, v, value})
	}
	return out, nil
}
func protection(folder string) bool {
	raw, err := readSmall(filepath.Join(folder, "contents.json"), 16<<20)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return true
	}
	var index struct {
		Content []struct {
			Key string `json:"key"`
		} `json:"content"`
	}
	if json.Unmarshal(bytes.TrimPrefix(raw, []byte{239, 187, 191}), &index) != nil {
		return true
	}
	for _, file := range index.Content {
		if file.Key != "" {
			return true
		}
	}
	return false
}
func indexPacks(ctx context.Context, w World) (map[string][]*candidate, error) {
	index := map[string][]*candidate{}
	seen := map[string]bool{}
	roots := []struct {
		folder, kind string
		priority     int
	}{}
	for _, kind := range []string{"resource", "behavior"} {
		roots = append(roots, struct {
			folder, kind string
			priority     int
		}{filepath.Join(w.Folder, kind+"_packs"), kind, 0})
		for _, dir := range []string{kind + "_packs", "development_" + kind + "_packs", filepath.Join("premium_cache", kind+"_packs")} {
			roots = append(roots, struct {
				folder, kind string
				priority     int
			}{filepath.Join(w.Storage, dir), kind, 2})
		}
	}
	for _, folder := range w.ExtraPackRoots {
		roots = append(roots, struct {
			folder, kind string
			priority     int
		}{folder, "", 1})
	}
	for _, dir := range []string{"world_templates", "development_world_templates", filepath.Join("premium_cache", "world_templates")} {
		roots = append(roots, struct {
			folder, kind string
			priority     int
		}{filepath.Join(w.Storage, dir), "", 4})
	}
	sort.SliceStable(roots, func(i, j int) bool { return roots[i].priority < roots[j].priority })
	for _, root := range roots {
		err := filepath.WalkDir(root.folder, func(path string, d fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				if path == root.folder {
					return fs.SkipDir
				}
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root.folder, path)
			if strings.Count(rel, string(filepath.Separator)) > 5 {
				return fs.SkipDir
			}
			raw, e := readSmall(filepath.Join(path, "manifest.json"), 4<<20)
			if os.IsNotExist(e) {
				return nil
			}
			if e != nil {
				return fs.SkipDir
			}
			var m manifest
			raw, e = hujson.Standardize(bytes.TrimPrefix(raw, []byte{239, 187, 191}))
			if e != nil || json.Unmarshal(raw, &m) != nil {
				return fs.SkipDir
			}
			id, v := packID(m.Header.UUID), version(m.Header.Version)
			if id == "" || v == "" {
				return fs.SkipDir
			}
			actual, e := canonical(path)
			if e != nil {
				return fs.SkipDir
			}
			if seen[strings.ToLower(actual)] {
				return fs.SkipDir
			}
			seen[strings.ToLower(actual)] = true
			kind := ""
			for _, module := range m.Modules {
				if module.Type == "data" || module.Type == "script" {
					kind = "behavior"
					break
				}
				if module.Type == "resources" {
					kind = "resource"
				}
			}
			if kind == "" {
				return nil
			}
			name := m.Header.Name
			if name == "" {
				name = id
			}
			c := &candidate{folder: actual, kind: kind, id: id, version: v, name: name, priority: root.priority, manifest: m, protected: protection(actual)}
			index[id+"|"+v] = append(index[id+"|"+v], c)
			return fs.SkipDir
		})
		if err != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	for _, values := range index {
		sort.Slice(values, func(i, j int) bool {
			if values[i].priority == values[j].priority {
				return values[i].folder < values[j].folder
			}
			return values[i].priority < values[j].priority
		})
	}
	return index, nil
}
func resolvePacks(ctx context.Context, w World, o Options, p *Plan) (map[string][]reference, error) {
	refs := map[string][]reference{}
	p.Packs = []Pack{}
	p.Issues = []Issue{}
	issue := func(code, message, id string) { p.Issues = append(p.Issues, Issue{"error", code, message, id}) }
	for _, kind := range []string{"resource", "behavior"} {
		values, err := readReferences(filepath.Join(w.Folder, "world_"+kind+"_packs.json"))
		if err != nil {
			issue("invalid_references", "The world's "+kind+" pack list could not be read. Refresh or repair it in Minecraft.", "")
		} else {
			refs[kind] = values
		}
	}
	if o.IncludeGlobal {
		values, err := readReferences(filepath.Join(w.Storage, "global_resource_packs.json"))
		if err != nil {
			issue("invalid_global_resources", "The active global texture pack list could not be read.", "")
		} else {
			refs["resource"] = append(refs["resource"], values...)
		}
	}
	index, err := indexPacks(ctx, w)
	if err != nil {
		return nil, err
	}
	state := map[string]int{}
	versions := map[string]string{}
	var visit func(reference, string, string)
	visit = func(ref reference, kind, by string) {
		key := ref.ID + "|" + ref.Version
		if state[key] == 1 {
			issue("dependency_cycle", "A pack dependency cycle was found.", ref.ID)
			return
		}
		if state[key] == 2 {
			return
		}
		state[key] = 1
		defer func() { state[key] = 2 }()
		if old, ok := versions[ref.ID]; ok && old != ref.Version {
			issue("version_conflict", "Different versions of the same pack are required.", ref.ID)
		}
		versions[ref.ID] = ref.Version
		choices := []*candidate{}
		for _, c := range index[key] {
			if kind == "" || c.kind == kind {
				choices = append(choices, c)
			}
		}
		if len(choices) == 0 {
			p.Packs = append(p.Packs, Pack{ID: ref.ID, Name: ref.ID, Version: ref.Version, Kind: kind, Status: "missing", RequiredBy: by})
			issue("missing_pack", "Required pack "+ref.ID+" version "+ref.Version+" is not installed.", ref.ID)
			return
		}
		c := choices[0]
		status := "included"
		if len(choices) > 1 && choices[1].priority == c.priority {
			status = "ambiguous"
			issue("ambiguous_pack", "Multiple installed folders have the same pack ID and version. Choose the correct copy in Minecraft before sharing.", ref.ID)
		}
		if c.protected {
			status = "protected"
			issue("protected_pack", "This pack contains protected content. A backup preserves it, but does not unlock it for another account.", ref.ID)
		}
		p.Packs = append(p.Packs, Pack{ID: ref.ID, Name: c.name, Version: c.version, Kind: c.kind, Status: status, RequiredBy: by, Embedded: inside(w.Folder, c.folder), candidate: c})
		for _, dependency := range c.manifest.Dependencies {
			if dependency.Module != "" {
				continue
			}
			id, v := packID(dependency.UUID), version(dependency.Version)
			if id == "" || v == "" {
				issue("invalid_dependency", "A required pack dependency has an invalid ID or version.", ref.ID)
				continue
			}
			raw, _ := json.Marshal(map[string]any{"pack_id": id, "version": dependency.Version})
			visit(reference{id, v, raw}, "", c.name)
		}
	}
	for _, kind := range []string{"resource", "behavior"} {
		for _, ref := range refs[kind] {
			visit(ref, kind, "World")
		}
	}
	// Embed dependencies and activate them below the world's existing stack entries.
	for _, pack := range p.Packs {
		if pack.candidate != nil {
			raw, _ := json.Marshal(map[string]any{"pack_id": pack.ID, "version": pack.candidate.manifest.Header.Version})
			refs[pack.Kind] = append(refs[pack.Kind], reference{pack.ID, pack.Version, raw})
		}
	}
	for kind, values := range refs {
		seen := map[string]bool{}
		unique := []reference{}
		for _, value := range values {
			key := value.ID + "|" + value.Version
			if !seen[key] {
				seen[key] = true
				unique = append(unique, value)
			}
		}
		refs[kind] = unique
	}
	return refs, nil
}
