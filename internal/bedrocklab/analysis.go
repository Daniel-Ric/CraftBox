package bedrocklab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type DecodingInfo struct {
	Restored   int    `json:"restored"`
	Unresolved int    `json:"unresolved"`
	Schema     string `json:"schema"`
	Version    string `json:"version"`
	Note       string `json:"note"`
}
type analyzedSnapshot struct {
	snapshot Snapshot
	files    map[string]File
	decoded  map[string][]byte
	info     DecodingInfo
	stamp    string
	created  time.Time
}

var archiveAnalysis = struct {
	sync.Mutex
	snapshots  map[string]*analyzedSnapshot
	registries map[string]*Schemas
}{snapshots: map[string]*analyzedSnapshot{}, registries: map[string]*Schemas{}}

func (s *Store) ClearAnalysis() {
	archiveAnalysis.Lock()
	defer archiveAnalysis.Unlock()
	root, _ := filepath.Abs(s.Root)
	for key := range archiveAnalysis.snapshots {
		if strings.HasPrefix(key, root+"|") {
			delete(archiveAnalysis.snapshots, key)
		}
	}
	archiveAnalysis.registries = map[string]*Schemas{}
}

func schemaStamp(p string) string {
	for _, name := range []string{"craftbox-schema.json", "exist.json"} {
		if st, err := os.Stat(filepath.Join(p, name)); err == nil {
			return fmt.Sprintf("%s:%s:%d:%d", p, name, st.Size(), st.ModTime().UnixNano())
		}
	}
	if st, err := os.Stat(p); err == nil {
		return fmt.Sprintf("%s:%d", p, st.ModTime().UnixNano())
	}
	return p
}

func (s *Store) embeddedSchemas(d Snapshot) (*Schemas, error) {
	root := filepath.Join(s.Root, "embedded", d.ID)
	r := &Schemas{root: root, byID: map[string]*schemaDocument{}, byFile: map[string]*schemaDocument{}}
	for _, f := range d.SchemaObjects {
		if !safePath(f.Path) {
			return nil, fmt.Errorf("unsafe embedded schema path")
		}
		b, err := s.read(f, 16<<20)
		if err != nil {
			return nil, err
		}
		if digest(b) != f.Hash {
			return nil, fmt.Errorf("embedded schema integrity mismatch: %s", f.Path)
		}
		var m schema
		if err = json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		id := norm(text(m, "$id"))
		if text(m, "$id") == "" || r.byID[id] != nil {
			return nil, fmt.Errorf("invalid embedded schema ID")
		}
		doc := &schemaDocument{ID: id, File: filepath.Join(root, filepath.FromSlash(f.Path)), Title: title(text(m, "title")), Version: text(m, "x-format-version"), Schema: m}
		r.documents = append(r.documents, doc)
		r.byID[id] = doc
		r.byFile[strings.ToLower(doc.File)] = doc
	}
	return r, nil
}

func (s *Store) analyzeSnapshot(ctx context.Context, id string) (*analyzedSnapshot, error) {
	if !validID(id) {
		return nil, fmt.Errorf("invalid snapshot ID")
	}
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(filepath.Join(root, "snapshots", id+".json"))
	if err != nil {
		return nil, err
	}
	key := root + "|" + id
	stamp := fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())
	bundles, err := s.SchemaBundles()
	if err != nil {
		return nil, err
	}
	sort.Slice(bundles, func(i, j int) bool { return bundles[i].Path < bundles[j].Path })
	for _, bundle := range bundles {
		stamp += "|" + schemaStamp(bundle.Path)
	}
	archiveAnalysis.Lock()
	defer archiveAnalysis.Unlock()
	if cached := archiveAnalysis.snapshots[key]; cached != nil && cached.stamp == stamp && time.Since(cached.created) < 30*time.Second {
		return cached, nil
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	d, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	result := &analyzedSnapshot{snapshot: d, files: map[string]File{}, decoded: map[string][]byte{}, stamp: stamp, created: time.Now()}
	var registry *Schemas
	if len(d.SchemaObjects) > 0 {
		registry, err = s.embeddedSchemas(d)
		result.info.Schema = "Embedded schema documents"
		result.info.Version = d.SchemaVersion
	} else {
		candidates := []SchemaBundle{}
		for _, b := range bundles {
			game, build := strings.Split(d.Version, "."), strings.Split(b.Version, ".")
			if len(game) >= 3 && len(build) >= 3 && strings.Join(game[:3], ".") == strings.Join(build[:3], ".") && (d.Channel == "" || d.Channel == b.Channel) {
				candidates = append(candidates, b)
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return versionCompare(candidates[i].Version, candidates[j].Version) > 0 })
		source := d.SchemaSource
		if len(candidates) > 0 {
			source = candidates[0].Path
			result.info.Version = candidates[0].Version
		}
		if source != "" {
			registryKey := schemaStamp(source)
			registry = archiveAnalysis.registries[registryKey]
			if registry == nil {
				registry, err = LoadSchemas(source)
				if err == nil {
					if len(archiveAnalysis.registries) >= 3 {
						archiveAnalysis.registries = map[string]*Schemas{}
					}
					archiveAnalysis.registries[registryKey] = registry
				}
			}
			result.info.Schema = source
		}
	}
	if err != nil {
		result.info.Note = "Schema loading failed: " + err.Error()
		registry = nil
	}
	result.snapshot.Files = append([]File(nil), d.Files...)
	result.snapshot.Decoded = 0
	result.snapshot.Unresolved = 0
	result.snapshot.Bytes = 0
	for i := range result.snapshot.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		f := &result.snapshot.Files[i]
		if f.Kind == "mcb" && registry != nil {
			raw, err := s.read(*f, maxFile+1)
			if err != nil {
				return nil, err
			}
			if digest(raw) != f.Hash {
				return nil, fmt.Errorf("compiled original integrity mismatch: %s", f.Path)
			}
			decoded, decodeErr := registry.Decode(raw)
			if decodeErr != nil {
				f.Issue = "MCB reconstruction unavailable: " + decodeErr.Error()
			} else {
				canonical, err := canonicalJSON(decoded)
				if err != nil {
					return nil, err
				}
				f.RawHash = f.Hash
				f.Hash = digest(decoded)
				f.Bytes = int64(len(decoded))
				f.Kind = "json"
				f.Semantic = digest(canonical)
				f.Identifiers = uniqueIdentifiers(canonical)
				f.Issue = ""
				result.decoded[f.Path] = decoded
			}
		}
		if f.RawHash != "" {
			result.info.Restored++
			result.snapshot.Decoded++
		}
		if f.Kind == "mcb" {
			result.info.Unresolved++
			result.snapshot.Unresolved++
		}
		result.snapshot.Bytes += f.Bytes
		result.files[f.Path] = *f
	}
	if registry == nil && result.info.Unresolved > 0 && result.info.Note == "" {
		result.info.Note = "No matching schema bundle is installed for this snapshot. Configure MCB schemas to reconstruct its JSON."
	}
	if len(archiveAnalysis.snapshots) >= 4 {
		oldKey := ""
		var oldest time.Time
		for k, v := range archiveAnalysis.snapshots {
			if oldKey == "" || v.created.Before(oldest) {
				oldKey, oldest = k, v.created
			}
		}
		delete(archiveAnalysis.snapshots, oldKey)
	}
	archiveAnalysis.snapshots[key] = result
	return result, nil
}

func (a *analyzedSnapshot) read(s *Store, f File, limit int64) ([]byte, error) {
	if b, ok := a.decoded[f.Path]; ok {
		return b[:min(int64(len(b)), limit)], nil
	}
	return s.read(f, limit)
}

type Comparison struct {
	Before         Snapshot     `json:"before"`
	After          Snapshot     `json:"after"`
	BeforeDecoding DecodingInfo `json:"beforeDecoding"`
	AfterDecoding  DecodingInfo `json:"afterDecoding"`
	Changes        []Change     `json:"changes"`
}

func (s *Store) Comparison(ctx context.Context, before, after string) (Comparison, error) {
	a, err := s.analyzeSnapshot(ctx, before)
	if err != nil {
		return Comparison{}, err
	}
	b, err := s.analyzeSnapshot(ctx, after)
	if err != nil {
		return Comparison{}, err
	}
	report := Comparison{Before: a.snapshot, After: b.snapshot, BeforeDecoding: a.info, AfterDecoding: b.info, Changes: compareSnapshots(a.snapshot, b.snapshot)}
	report.Before.Files, report.After.Files = nil, nil
	report.Before.SchemaObjects, report.After.SchemaObjects = nil, nil
	for i := range report.Changes {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		c := &report.Changes[i]
		f := c.After
		if f == nil {
			f = c.Before
		}
		c.Category = f.Kind
		if f.Kind == "mcb" {
			c.Category = "unresolved"
		}
		if strings.EqualFold(filepath.Ext(c.Path), ".fsb") || strings.EqualFold(filepath.Ext(c.Path), ".ogg") || strings.EqualFold(filepath.Ext(c.Path), ".wav") {
			c.Category = "audio"
		}
		switch c.Status {
		case "added":
			c.Summary = "New file"
		case "removed":
			c.Summary = "File removed"
		case "renamed":
			c.Summary = "Moved; contents are identical"
		case "formatting":
			c.Category = "formatting"
			c.Summary = "Formatting or property order changed; JSON values are identical"
		default:
			c.Summary = "File content changed"
		}
		if c.Status == "modified" && c.Before.Kind == "json" && c.After.Kind == "json" && c.Before.Bytes <= 1<<20 && c.After.Bytes <= 1<<20 {
			left, err := a.read(s, *c.Before, 1<<20)
			if err != nil {
				return report, err
			}
			right, err := b.read(s, *c.After, 1<<20)
			if err != nil {
				return report, err
			}
			analysis, err := AnalyzeJSON(left, right)
			if err == nil {
				c.FieldCount = len(analysis.Fields)
				c.Category = "content"
				c.Summary = fmt.Sprintf("%d JSON fields changed", len(analysis.Fields))
				if analysis.Truncated {
					c.Summary = "At least 256 JSON fields changed"
				}
				if analysis.VersionOnly {
					c.Category = "version"
					c.Summary = "Only format_version changed; other values are identical"
				}
			}
		}
		if c.Category == "audio" && c.Status == "modified" {
			c.Summary = "Audio bank or encoded audio changed; inspect stream metadata"
		}
	}
	return report, nil
}
