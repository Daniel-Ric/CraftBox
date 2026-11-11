package bedrocklab

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tailscale/hujson"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const maxFile = int64(256 << 20)

type CaptureRequest struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Channel string `json:"channel"`
	Schemas string `json:"schemas"`
}

type File struct {
	Path        string   `json:"path"`
	Hash        string   `json:"hash"`
	RawHash     string   `json:"rawHash,omitempty"`
	Semantic    string   `json:"semantic"`
	Bytes       int64    `json:"bytes"`
	Kind        string   `json:"kind"`
	Origin      string   `json:"origin"`
	Issue       string   `json:"issue,omitempty"`
	Identifiers []string `json:"identifiers"`
}

type Snapshot struct {
	ParentID      string   `json:"parentId,omitempty"`
	SchemaObjects []File   `json:"schemaObjects,omitempty"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Channel       string   `json:"channel"`
	Source        string   `json:"source"`
	Created       string   `json:"created"`
	SchemaSource  string   `json:"schemaSource"`
	SchemaVersion string   `json:"schemaVersion,omitempty"`
	SchemaHash    string   `json:"schemaHash,omitempty"`
	Files         []File   `json:"files"`
	Count         int      `json:"count"`
	Bytes         int64    `json:"bytes"`
	Archives      int      `json:"archives"`
	Decoded       int      `json:"decoded"`
	Unresolved    int      `json:"unresolved"`
	Issues        []string `json:"issues"`
}

type Progress struct {
	Running  bool   `json:"running"`
	Phase    string `json:"phase"`
	Path     string `json:"path"`
	Done     int    `json:"done"`
	Total    int    `json:"total"`
	Snapshot string `json:"snapshot"`
	Error    string `json:"error"`
}

type Change struct {
	Category   string `json:"category,omitempty"`
	Summary    string `json:"summary,omitempty"`
	FieldCount int    `json:"fieldCount,omitempty"`
	Path       string `json:"path"`
	Status     string `json:"status"`
	Before     *File  `json:"before,omitempty"`
	After      *File  `json:"after,omitempty"`
}

type Preview struct {
	Snapshot  string        `json:"snapshot"`
	Waveform  []float64     `json:"waveform,omitempty"`
	Decoding  *DecodingInfo `json:"decoding,omitempty"`
	Binary    *BinaryInfo   `json:"binary,omitempty"`
	Audio     string        `json:"audio,omitempty"`
	Original  string        `json:"original,omitempty"`
	Canonical string        `json:"canonical,omitempty"`
	File      File          `json:"file"`
	Text      string        `json:"text"`
	Image     string        `json:"image"`
	Truncated bool          `json:"truncated"`
}

type Store struct{ Root string }

func New(root string) *Store { return &Store{root} }
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validID(s string) bool  { b, e := hex.DecodeString(s); return e == nil && len(b) == 16 }
func (s *Store) object(h string) (string, error) {
	b, e := hex.DecodeString(h)
	if e != nil || len(b) != 32 {
		return "", errors.New("invalid object hash")
	}
	return filepath.Join(s.Root, "objects", h[:2], h[2:]), nil
}
func (s *Store) put(b []byte) (string, error) {
	h := digest(b)
	p, _ := s.object(h)
	if _, e := os.Stat(p); e == nil {
		return h, nil
	}
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return "", e
	}
	t, e := os.CreateTemp(filepath.Dir(p), ".object-*")
	if e != nil {
		return "", e
	}
	name := t.Name()
	defer os.Remove(name)
	if _, e = t.Write(b); e != nil {
		t.Close()
		return "", e
	}
	if e = t.Close(); e != nil {
		return "", e
	}
	if e = os.Rename(name, p); e != nil {
		if _, stat := os.Stat(p); stat != nil {
			return "", e
		}
	}
	return h, nil
}
func (s *Store) Get(id string) (Snapshot, error) {
	var d Snapshot
	if !validID(id) {
		return d, errors.New("invalid snapshot ID")
	}
	b, e := os.ReadFile(filepath.Join(s.Root, "snapshots", id+".json"))
	if e != nil {
		return d, e
	}
	if e = json.Unmarshal(b, &d); e != nil {
		return d, e
	}
	if d.ID != id {
		return d, errors.New("snapshot identity mismatch")
	}
	return d, nil
}
func (s *Store) List() ([]Snapshot, error) {
	result := []Snapshot{}
	entries, e := os.ReadDir(filepath.Join(s.Root, "snapshots"))
	if os.IsNotExist(e) {
		return result, nil
	}
	if e != nil {
		return nil, e
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		d, e := s.Get(strings.TrimSuffix(entry.Name(), ".json"))
		if e != nil {
			return nil, fmt.Errorf("snapshot %s: %w", entry.Name(), e)
		}
		d.Files = nil
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Created > result[j].Created })
	return result, nil
}
func (s *Store) Delete(id string) error {
	if !validID(id) {
		return errors.New("invalid snapshot ID")
	}
	return os.Remove(filepath.Join(s.Root, "snapshots", id+".json"))
}

var identifiers = regexp.MustCompile(`\b[a-z][a-z0-9_.-]*:[a-zA-Z0-9_./-]+`)

func uniqueIdentifiers(b []byte) []string {
	set := map[string]bool{}
	for _, match := range identifiers.FindAll(b, 10000) {
		set[string(match)] = true
	}
	values := make([]string, 0, len(set))
	for k := range set {
		values = append(values, k)
	}
	sort.Strings(values)
	return values
}
func canonicalJSON(b []byte) ([]byte, error) {
	b, err := hujson.Standardize(bytes.TrimPrefix(b, []byte{239, 187, 191}))
	if err != nil {
		return nil, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var v any
	if e := d.Decode(&v); e != nil {
		return nil, e
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return nil, errors.New("trailing JSON data")
	}
	return json.Marshal(v)
}
func classify(name string, b []byte) string {
	if len(b) >= 4 && bytes.Equal(b[:4], []byte{127, 77, 67, 66}) {
		return "mcb"
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".json", ".material":
		return "json"
	case ".png", ".jpg", ".jpeg", ".gif":
		return "image"
	case ".brarchive":
		return "archive"
	case ".lang", ".txt", ".md", ".xml", ".hlsl", ".glsl", ".js", ".html", ".css", ".properties", ".csv", ".mcfunction":
		return "text"
	}
	if len(b) < 1<<20 && utf8.Valid(b) && !bytes.ContainsRune(b, 0) {
		return "text"
	}
	return "binary"
}

func (s *Store) Capture(ctx context.Context, request CaptureRequest, progress func(Progress)) (Snapshot, error) {
	d := Snapshot{Name: strings.TrimSpace(request.Name), Version: request.Version, Channel: request.Channel, Created: time.Now().UTC().Format(time.RFC3339Nano), Files: []File{}, Issues: []string{}, SchemaSource: request.Schemas}
	random := make([]byte, 16)
	if _, e := rand.Read(random); e != nil {
		return d, e
	}
	d.ID = hex.EncodeToString(random)
	source, e := filepath.Abs(request.Source)
	if e != nil {
		return d, e
	}
	d.Source = source
	if d.Name == "" {
		d.Name = filepath.Base(source)
	}
	storeRoot, e := filepath.Abs(s.Root)
	if e != nil {
		return d, e
	}
	rel, e := filepath.Rel(source, storeRoot)
	if e == nil && (rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")) {
		return d, errors.New("source cannot contain the Bedrock Lab library")
	}
	root, e := os.OpenRoot(source)
	if e != nil {
		return d, e
	}
	defer root.Close()
	var schemas *Schemas
	if request.Schemas != "" {
		schemas, e = LoadSchemas(request.Schemas)
		if e != nil {
			return d, e
		}
		d.SchemaObjects, e = s.preserveSchemas(schemas)
		if e != nil {
			return d, e
		}
		if b, err := os.ReadFile(filepath.Join(request.Schemas, "craftbox-schema.json")); err == nil {
			var bundle SchemaBundle
			if err = json.Unmarshal(b, &bundle); err != nil {
				return d, err
			}
			d.SchemaVersion, d.SchemaHash = bundle.Version, bundle.Hash
			if request.Version != "" {
				game, build := strings.Split(request.Version, "."), strings.Split(bundle.Version, ".")
				if len(game) >= 3 && len(build) >= 3 && strings.Join(game[:3], ".") != strings.Join(build[:3], ".") {
					return d, fmt.Errorf("BDS schema version %s does not match game data %s; select a matching schema bundle", bundle.Version, request.Version)
				}
			}
		}
	}
	paths := []string{}
	emit := func(phase, p string, done, total int) {
		if progress != nil {
			progress(Progress{Running: true, Phase: phase, Path: p, Done: done, Total: total})
		}
	}
	emit("Scanning", source, 0, 0)
	e = fs.WalkDir(root.FS(), ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if len(d.Issues) < 500 {
				d.Issues = append(d.Issues, "Skipped symbolic link: "+p)
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !safePath(p) {
			return fmt.Errorf("unsafe source path %q", p)
		}
		paths = append(paths, p)
		if len(paths) > 200000 {
			return errors.New("source exceeds 200,000 files")
		}
		return nil
	})
	if e != nil {
		return d, e
	}
	files := map[string]File{}
	lower := map[string]string{}
	totalBytes := int64(0)
	add := func(name, origin string, b []byte) error {
		if !safePath(name) {
			return fmt.Errorf("unsafe extracted path %q", name)
		}
		key := strings.ToLower(name)
		if old := lower[key]; old != "" && old != name {
			return fmt.Errorf("case-colliding paths %q and %q", old, name)
		}
		lower[key] = name
		if existing, ok := files[name]; ok && (origin != "" && existing.Origin == "" && existing.Bytes > 0 || len(b) == 0) {
			return nil
		}
		if len(files) >= 500000 {
			return errors.New("snapshot exceeds 500,000 entries")
		}
		totalBytes += int64(len(b))
		if totalBytes > 32<<30 {
			return errors.New("snapshot exceeds 32 GiB")
		}
		f := File{Path: name, Origin: origin, Bytes: int64(len(b)), Kind: classify(name, b), Identifiers: []string{}}
		raw, e := s.put(b)
		if e != nil {
			return e
		}
		f.Hash = raw
		if f.Kind == "mcb" {
			if schemas == nil {
				f.Issue = "MCB preserved: select matching BDS schemas to decode"
			} else if decoded, err := schemas.Decode(b); err != nil {
				f.Issue = "MCB preserved: " + err.Error()
			} else {
				f.RawHash = raw
				b = decoded
				f.Kind = "json"
				f.Bytes = int64(len(b))
				f.Hash, e = s.put(b)
				if e != nil {
					return e
				}
				d.Decoded++
			}
			if f.Issue != "" {
				d.Unresolved++
			}
		}
		if f.Kind == "json" {
			if canonical, err := canonicalJSON(b); err == nil {
				f.Semantic = digest(canonical)
				f.Identifiers = uniqueIdentifiers(canonical)
			} else {
				f.Issue = "Invalid JSON: " + err.Error()
			}
		}
		if f.Kind == "text" {
			f.Identifiers = uniqueIdentifiers(b)
		}
		if f.Kind == "image" {
			if _, _, err := image.DecodeConfig(bytes.NewReader(b)); err != nil {
				f.Issue = "Invalid image header: " + err.Error()
			}
		}
		files[name] = f
		return nil
	}
	for i, p := range paths {
		if e := ctx.Err(); e != nil {
			return d, e
		}
		emit("Reading", p, i, len(paths))
		f, e := root.Open(p)
		if e != nil {
			return d, e
		}
		info, e := f.Stat()
		if e != nil {
			f.Close()
			return d, e
		}
		if !info.Mode().IsRegular() {
			f.Close()
			return d, fmt.Errorf("unsupported source file %q", p)
		}
		if strings.EqualFold(path.Ext(p), ".brarchive") {
			if info.Size() > 2<<30 {
				f.Close()
				return d, fmt.Errorf("archive exceeds 2 GiB: %s", p)
			}
			entries, err := ReadArchive(f, info.Size())
			if err != nil {
				f.Close()
				return d, fmt.Errorf("%s: %w", p, err)
			}
			d.Archives++
			for _, entry := range entries {
				if e := ctx.Err(); e != nil {
					f.Close()
					return d, e
				}
				if entry.Size > maxFile {
					f.Close()
					return d, fmt.Errorf("archive entry exceeds 256 MiB: %s", entry.Path)
				}
				b := make([]byte, entry.Size)
				if _, e = f.ReadAt(b, entry.Offset); e != nil && !(e == io.EOF && len(b) == 0) {
					f.Close()
					return d, e
				}
				if e = add(path.Join(archiveDestination(p), entry.Path), p, b); e != nil {
					f.Close()
					return d, e
				}
			}
			if info.Size() <= maxFile {
				if _, e = f.Seek(0, 0); e != nil {
					f.Close()
					return d, e
				}
				b, e := io.ReadAll(io.LimitReader(f, maxFile+1))
				if e != nil {
					f.Close()
					return d, e
				}
				if e = add(p, "", b); e != nil {
					f.Close()
					return d, e
				}
			} else {
				d.Issues = append(d.Issues, "Expanded archive; original exceeds storage limit: "+p)
			}
		} else {
			if info.Size() > maxFile {
				f.Close()
				return d, fmt.Errorf("file exceeds 256 MiB: %s", p)
			}
			b, e := io.ReadAll(io.LimitReader(f, maxFile+1))
			if e != nil {
				f.Close()
				return d, e
			}
			if int64(len(b)) > maxFile {
				f.Close()
				return d, fmt.Errorf("file grew beyond limit: %s", p)
			}
			if e = add(p, "", b); e != nil {
				f.Close()
				return d, e
			}
		}
		f.Close()
	}
	if e := ctx.Err(); e != nil {
		return d, e
	}
	for _, f := range files {
		d.Files = append(d.Files, f)
		d.Bytes += f.Bytes
	}
	sort.Slice(d.Files, func(i, j int) bool { return d.Files[i].Path < d.Files[j].Path })
	d.Count = len(d.Files)
	emit("Saving", d.Name, len(paths), len(paths))
	return s.saveSnapshot(ctx, d)
}

func (s *Store) saveSnapshot(ctx context.Context, d Snapshot) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return d, err
	}
	var e error
	if e = os.MkdirAll(filepath.Join(s.Root, "snapshots"), 0700); e != nil {
		return d, e
	}
	b, e := json.Marshal(d)
	if e != nil {
		return d, e
	}
	tmp, e := os.CreateTemp(filepath.Join(s.Root, "snapshots"), ".snapshot-*")
	if e != nil {
		return d, e
	}
	defer os.Remove(tmp.Name())
	if _, e = tmp.Write(b); e != nil {
		tmp.Close()
		return d, e
	}
	if e = tmp.Close(); e != nil {
		return d, e
	}
	if e = ctx.Err(); e != nil {
		return d, e
	}
	e = os.Rename(tmp.Name(), filepath.Join(s.Root, "snapshots", d.ID+".json"))
	return d, e
}

func (s *Store) Compare(before, after string) ([]Change, error) {
	report, err := s.Comparison(context.Background(), before, after)
	return report.Changes, err
}
func compareSnapshots(a, b Snapshot) []Change {
	left := map[string]File{}
	right := map[string]File{}
	for _, f := range a.Files {
		left[f.Path] = f
	}
	for _, f := range b.Files {
		right[f.Path] = f
	}
	changes := []Change{}
	for p, f := range left {
		g, ok := right[p]
		if !ok {
			changes = append(changes, Change{Path: p, Status: "removed", Before: &f})
			continue
		}
		if f.Hash == g.Hash {
			continue
		}
		status := "modified"
		if f.Semantic != "" && f.Semantic == g.Semantic {
			status = "formatting"
		}
		changes = append(changes, Change{Path: p, Status: status, Before: &f, After: &g})
	}
	for p, f := range right {
		if _, ok := left[p]; !ok {
			changes = append(changes, Change{Path: p, Status: "added", After: &f})
		}
	}

	removed, added := map[string][]int{}, map[string][]int{}
	for i, c := range changes {
		if c.Status == "removed" {
			removed[c.Before.Hash] = append(removed[c.Before.Hash], i)
		}
		if c.Status == "added" {
			added[c.After.Hash] = append(added[c.After.Hash], i)
		}
	}
	paired := map[int]bool{}
	for hash, old := range removed {
		if next := added[hash]; len(old) == 1 && len(next) == 1 {
			i, j := old[0], next[0]
			changes[j].Status, changes[j].Before = "renamed", changes[i].Before
			paired[i] = true
		}
	}
	filtered := changes[:0]
	for i, c := range changes {
		if !paired[i] {
			filtered = append(filtered, c)
		}
	}
	changes = filtered
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}
func (s *Store) file(id, name string) (File, error) {
	d, e := s.Get(id)
	if e != nil {
		return File{}, e
	}
	for _, f := range d.Files {
		if f.Path == name {
			return f, nil
		}
	}
	return File{}, errors.New("file is not part of this snapshot")
}
func (s *Store) read(f File, limit int64) ([]byte, error) {
	p, e := s.object(f.Hash)
	if e != nil {
		return nil, e
	}
	r, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, limit))
}
func (s *Store) Preview(id, name string) (Preview, error) {
	analysis, e := s.analyzeSnapshot(context.Background(), id)
	if e != nil {
		return Preview{}, e
	}
	f, ok := analysis.files[name]
	if !ok {
		return Preview{}, errors.New("file is not part of this snapshot")
	}
	p := Preview{File: f, Snapshot: id, Decoding: &analysis.info}
	if e != nil {
		return p, e
	}
	b, e := analysis.read(s, f, 1<<20)
	if e != nil {
		return p, e
	}
	p.Truncated = f.Bytes > int64(len(b))
	p.Binary = InspectBinary(name, b, f.Bytes)
	if !p.Truncated && p.Binary != nil && p.Binary.Format == "FMOD FSB5" {
		if wave, peaks, err := previewFSBAudio(b, p.Binary); err == nil {
			p.Audio = "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(wave)
			p.Waveform = peaks
		}
	}
	if !p.Truncated && (strings.EqualFold(path.Ext(name), ".ogg") || strings.EqualFold(path.Ext(name), ".wav") || strings.EqualFold(path.Ext(name), ".mp3")) {
		mime := "audio/ogg"
		if strings.EqualFold(path.Ext(name), ".wav") {
			mime = "audio/wav"
		}
		if strings.EqualFold(path.Ext(name), ".mp3") {
			mime = "audio/mpeg"
		}
		p.Audio = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
	}
	if f.Kind == "image" {
		if !p.Truncated {
			_, format, err := image.DecodeConfig(bytes.NewReader(b))
			if err == nil {
				p.Image = "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(b)
			}
		}
		return p, nil
	}
	if f.Kind == "json" && !p.Truncated {
		p.Original = string(b)
		if canonical, err := canonicalJSON(b); err == nil {
			var normalized bytes.Buffer
			if json.Indent(&normalized, canonical, "", "  ") == nil {
				p.Canonical = normalized.String()
			}
		}
		var out bytes.Buffer
		if json.Indent(&out, b, "", "  ") == nil {
			p.Text = out.String()
			return p, nil
		}
	}
	if (f.Kind == "json" || f.Kind == "text") && utf8.Valid(b) {
		p.Text = string(b)
	} else {
		if len(b) > 65536 {
			b = b[:65536]
			p.Truncated = true
		}
		p.Text = hex.Dump(b)
	}
	return p, nil
}

type Query struct {
	Snapshot string `json:"snapshot"`
	Search   string `json:"search"`
	Kind     string `json:"kind"`
	Mode     string `json:"mode"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}
type Page struct {
	Files     []File `json:"files"`
	Total     int    `json:"total"`
	Truncated bool   `json:"truncated"`
}

func (s *Store) Query(ctx context.Context, q Query) (Page, error) {
	analysis, e := s.analyzeSnapshot(ctx, q.Snapshot)
	result := Page{Files: []File{}}
	if e != nil {
		return result, e
	}
	d := analysis.snapshot
	if q.Limit < 1 || q.Limit > 500 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	needle := strings.ToLower(q.Search)
	scanned := int64(0)
	for _, f := range d.Files {
		if e := ctx.Err(); e != nil {
			return result, e
		}
		if q.Kind != "" && q.Kind != f.Kind {
			continue
		}
		match := needle == "" || strings.Contains(strings.ToLower(f.Path), needle)
		if q.Mode == "issues" {
			match = f.Issue != "" && (needle == "" || strings.Contains(strings.ToLower(f.Path+" "+f.Issue), needle))
		}
		if q.Mode == "identifiers" {
			match = false
			for _, id := range f.Identifiers {
				if needle == "" || strings.Contains(strings.ToLower(id), needle) {
					match = true
					break
				}
			}
		}
		if q.Mode == "content" && !match && (f.Kind == "json" || f.Kind == "text") {
			if scanned > 256<<20 {
				result.Truncated = true
				break
			}
			b, err := analysis.read(s, f, 4<<20)
			if err != nil {
				return result, err
			}
			scanned += int64(len(b))
			if int64(len(b)) < f.Bytes {
				result.Truncated = true
			}
			match = bytes.Contains(bytes.ToLower(b), []byte(needle))
		}
		if !match {
			continue
		}
		if result.Total >= q.Offset && len(result.Files) < q.Limit {
			result.Files = append(result.Files, f)
		}
		result.Total++
	}
	return result, nil
}

type Validation struct {
	Checked int      `json:"checked"`
	Issues  []string `json:"issues"`
}

func (s *Store) Validate(ctx context.Context, id string) (Validation, error) {
	d, e := s.Get(id)
	v := Validation{Issues: []string{}}
	if e != nil {
		return v, e
	}
	for _, f := range d.Files {
		if e := ctx.Err(); e != nil {
			return v, e
		}
		b, e := s.read(f, maxFile+1)
		if e != nil {
			return v, e
		}
		v.Checked++
		if digest(b) != f.Hash || int64(len(b)) != f.Bytes {
			v.Issues = append(v.Issues, "Object integrity mismatch: "+f.Path)
		}
		if f.Issue != "" {
			v.Issues = append(v.Issues, f.Path+": "+f.Issue)
		}
		if f.Kind == "json" {
			if _, err := canonicalJSON(b); err != nil {
				v.Issues = append(v.Issues, "JSON integrity: "+f.Path+": "+err.Error())
			}
		}
		if f.Kind == "image" {
			config, _, err := image.DecodeConfig(bytes.NewReader(b))
			if err == nil && int64(config.Width)*int64(config.Height) > 64<<20 {
				v.Issues = append(v.Issues, "Image payload exceeds validation pixel limit: "+f.Path)
			} else if err != nil {
				v.Issues = append(v.Issues, "Image payload: "+f.Path+": "+err.Error())
			} else if _, _, err = image.Decode(bytes.NewReader(b)); err != nil {
				v.Issues = append(v.Issues, "Image payload: "+f.Path+": "+err.Error())
			}
		}
		if f.Kind == "archive" {
			if _, err := ReadArchive(bytes.NewReader(b), int64(len(b))); err != nil {
				v.Issues = append(v.Issues, "Archive directory: "+f.Path+": "+err.Error())
			}
		}
		if f.RawHash != "" {
			original := f
			original.Hash = f.RawHash
			raw, err := s.read(original, maxFile+1)
			if err != nil {
				return v, err
			}
			if digest(raw) != f.RawHash {
				v.Issues = append(v.Issues, "Compiled original integrity mismatch: "+f.Path)
			}
		}
	}
	for _, f := range d.SchemaObjects {
		if err := ctx.Err(); err != nil {
			return v, err
		}
		b, err := s.read(f, maxFile+1)
		if err != nil {
			return v, err
		}
		if digest(b) != f.Hash || int64(len(b)) != f.Bytes {
			v.Issues = append(v.Issues, "Schema integrity mismatch: "+f.Path)
		}
	}
	if err := s.validateArchiveMembers(ctx, d, &v); err != nil {
		return v, err
	}
	return v, nil
}

func (s *Store) Export(ctx context.Context, id string, w io.Writer) error {
	d, e := s.Get(id)
	if e != nil {
		return e
	}
	z := zip.NewWriter(w)
	defer z.Close()
	b, _ := json.MarshalIndent(d, "", "  ")
	manifest, e := z.Create("craftbox-snapshot.json")
	if e != nil {
		return e
	}
	if _, e = manifest.Write(b); e != nil {
		return e
	}

	var attributes strings.Builder
	attributes.WriteString("# Generated by CraftBox Archive; compiled originals remain byte-exact.\n* -text\ncompiled/** binary\nschemas/** text diff merge eol=lf\n")
	for _, f := range d.Files {
		quoted, _ := json.Marshal("files/" + f.Path)
		attributes.Write(quoted)
		if f.Kind == "json" || f.Kind == "text" {
			attributes.WriteString(" text diff merge eol=lf\n")
		} else {
			attributes.WriteString(" binary\n")
		}
	}
	attr, e := z.Create(".gitattributes")
	if e != nil {
		return e
	}
	if _, e = io.WriteString(attr, attributes.String()); e != nil {
		return e
	}
	for _, f := range d.Files {
		if e := ctx.Err(); e != nil {
			return e
		}
		if !safePath(f.Path) {
			return errors.New("unsafe snapshot path")
		}
		p, e := s.object(f.Hash)
		if e != nil {
			return e
		}
		r, e := os.Open(p)
		if e != nil {
			return e
		}
		dst, e := z.Create("files/" + f.Path)
		if e == nil {
			_, e = io.Copy(dst, r)
		}
		r.Close()
		if e != nil {
			return e
		}
		if f.RawHash != "" {
			raw, e := s.object(f.RawHash)
			if e != nil {
				return e
			}
			r, e := os.Open(raw)
			if e != nil {
				return e
			}
			dst, e := z.Create("compiled/" + f.Path + ".mcb")
			if e == nil {
				_, e = io.Copy(dst, r)
			}
			r.Close()
			if e != nil {
				return e
			}
		}
	}
	for _, f := range d.SchemaObjects {
		if e := ctx.Err(); e != nil {
			return e
		}
		if !safePath(f.Path) {
			return errors.New("unsafe schema path")
		}
		b, err := s.read(f, maxFile+1)
		if err != nil {
			return err
		}
		if digest(b) != f.Hash {
			return errors.New("schema integrity mismatch: " + f.Path)
		}
		dst, err := z.Create("schemas/" + f.Path)
		if err != nil {
			return err
		}
		if _, err = dst.Write(b); err != nil {
			return err
		}
	}
	return z.Close()
}

func (s *Store) ExportPack(ctx context.Context, id, name string, w io.Writer) error {
	analysis, e := s.analyzeSnapshot(ctx, id)
	if e != nil {
		return e
	}
	d := analysis.snapshot
	found := false
	files := map[string]File{}
	for _, f := range d.Files {
		files[f.Path] = f
		if f.Path == name {
			found = true
		}
	}
	if !found {
		return errors.New("selected file is not in the snapshot")
	}
	root := path.Dir(name)
	for {
		manifest := path.Join(root, "manifest.json")
		if _, ok := files[manifest]; ok {
			break
		}
		if root == "." {
			return errors.New("this file has no parent pack manifest")
		}
		root = path.Dir(root)
	}
	prefix := root + "/"
	if root == "." {
		prefix = ""
	}
	z := zip.NewWriter(w)
	defer z.Close()
	for _, f := range d.Files {
		if !strings.HasPrefix(f.Path, prefix) {
			continue
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		rel := strings.TrimPrefix(f.Path, prefix)
		if !safePath(rel) {
			return errors.New("unsafe pack path")
		}
		if b, ok := analysis.decoded[f.Path]; ok {
			dst, err := z.Create(rel)
			if err != nil {
				return err
			}
			if _, err = dst.Write(b); err != nil {
				return err
			}
			continue
		}
		p, e := s.object(f.Hash)
		if e != nil {
			return e
		}
		src, e := os.Open(p)
		if e != nil {
			return e
		}
		dst, e := z.Create(rel)
		if e == nil {
			_, e = io.Copy(dst, src)
		}
		src.Close()
		if e != nil {
			return e
		}
	}
	return z.Close()
}
