package assetlibrary

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dblezek/tga"
	"github.com/tailscale/hujson"
)

const MaxImportBytes int64 = 2 << 30
const MaxFiles = 50000
const MaxPreviewBytes = 16 << 20

type Item struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	UUID             string   `json:"uuid"`
	MinimumVersion   string   `json:"minimumVersion"`
	Source           string   `json:"source"`
	Created          string   `json:"created"`
	Bytes            int64    `json:"bytes"`
	Files            int      `json:"files"`
	Favourite        bool     `json:"favourite"`
	Tags             []string `json:"tags"`
	Image            string   `json:"image"`
	TexturePath      string   `json:"texturePath"`
	Original         string   `json:"original,omitempty"`
	Encrypted        bool     `json:"encrypted"`
	ArmModel         string   `json:"armModel"`
	ArmModelOverride bool     `json:"armModelOverride,omitempty"`
	CaptureType      string   `json:"captureType,omitempty"`
	PlayerUUID       string   `json:"playerUUID,omitempty"`
	HasGeometry      bool     `json:"hasGeometry"`
	HasCape          bool     `json:"hasCape"`
	HasAnimations    bool     `json:"hasAnimations"`
	Persona          bool     `json:"persona"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
}
type File struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Hash  string `json:"hash"`
	Kind  string `json:"kind"`
}
type Issue struct {
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	Reference string `json:"reference,omitempty"`
}
type Detail struct {
	Partial  bool           `json:"partial,omitempty"`
	Item     Item           `json:"item"`
	Files    []File         `json:"files"`
	Issues   []Issue        `json:"issues"`
	Metadata map[string]any `json:"metadata"`
}
type Preview struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Data      string `json:"data"`
	Truncated bool   `json:"truncated"`
}
type Change struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}
type Conflict struct {
	Path   string   `json:"path"`
	Winner string   `json:"winner"`
	Packs  []string `json:"packs"`
}
type Stack struct {
	Conflicts []Conflict        `json:"conflicts"`
	Issues    []Issue           `json:"issues"`
	Resolved  map[string]string `json:"resolved"`
}
type Library struct {
	definitionMu  sync.Mutex
	blockRevision atomic.Uint64
	Root          string
	mu            sync.Mutex
	catalogMu     sync.Mutex
	catalogs      map[string]catalogEntry
	previews      *previewCache
	listCache     []Item
	listIndex     time.Time
	listEquipment time.Time
	listChecked   time.Time
}

func New(root string) *Library {
	return &Library{Root: root, previews: &previewCache{values: map[string]ObjectPreview{}}}
}
func (l *Library) rawItems() ([]Item, error) {
	b, e := os.ReadFile(filepath.Join(l.Root, "index.json"))
	if os.IsNotExist(e) {
		return []Item{}, nil
	}
	if e != nil {
		return nil, e
	}
	var items []Item
	if e = json.Unmarshal(b, &items); e != nil {
		return nil, fmt.Errorf("Cannot read asset library: %w", e)
	}
	if items == nil {
		items = []Item{}
	}
	return items, nil
}
func (l *Library) load() ([]Item, error) {
	items, err := l.rawItems()
	if err != nil {
		return nil, err
	}
	l.hydrate(items)
	return items, nil
}
func (l *Library) hydrate(items []Item) {
	records, _ := l.equipmentRecords()
	for i := range items {
		if items[i].Kind == "skins" {
			root, _ := l.RootFor(items[i].ID)
			if b, err := readLimit(filepath.Join(root, "capture.json"), MaxPreviewBytes); err == nil {
				var meta map[string]any
				if json.Unmarshal(b, &meta) == nil {
					items[i].PlayerUUID = objectString(meta["playerUUID"])
					if !items[i].ArmModelOverride && skinPatchSlim(meta) {
						items[i].ArmModel = "slim"
					}
					items[i].CaptureType = capturedSkinType(meta, equipmentForMetadata(records, meta))
				}
			}
		}
		if items[i].Kind == "packs" {
			root, _ := l.RootFor(items[i].ID)
			if root != "" {
				items[i].Encrypted = encryptedPack(root)
			}
		}
	}
}
func encryptedPack(root string) bool {
	f, e := os.Open(filepath.Join(root, "contents.json"))
	if e != nil {
		return false
	}
	defer f.Close()
	var header [8]byte
	if _, e = io.ReadFull(f, header[:]); e != nil {
		return false
	}
	return bytes.Equal(header[4:8], []byte{0xfc, 0xb9, 0xcf, 0x9b})
}
func (l *Library) save(items []Item) error {
	l.listCache = nil
	if e := os.MkdirAll(l.Root, 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(items, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(l.Root, "index-*.tmp")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(name, filepath.Join(l.Root, "index.json"))
}
func (l *Library) List() ([]Item, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	stamp := func(name string) time.Time {
		if info, err := os.Stat(filepath.Join(l.Root, name)); err == nil {
			return info.ModTime()
		}
		return time.Time{}
	}
	index, equipment := stamp("index.json"), stamp("equipment.json")
	if l.listCache == nil || !index.Equal(l.listIndex) || !equipment.Equal(l.listEquipment) || time.Since(l.listChecked) > 2*time.Second {
		items, err := l.load()
		if err != nil {
			return nil, err
		}
		l.listCache = items
		l.listIndex = index
		l.listEquipment = equipment
		l.listChecked = time.Now()
	}
	items := append([]Item{}, l.listCache...)
	for i := range items {
		items[i].Tags = append([]string{}, items[i].Tags...)
	}
	return items, nil
}
func validID(id string) bool { b, e := hex.DecodeString(id); return e == nil && len(b) == 32 }
func (l *Library) RootFor(id string) (string, error) {
	if !validID(id) {
		return "", errors.New("Invalid asset")
	}
	return filepath.Join(l.Root, "items", id), nil
}
func (l *Library) get(id string) (Item, error) {
	items, e := l.rawItems()
	if e != nil {
		return Item{}, e
	}
	for _, v := range items {
		if v.ID == id {
			one := []Item{v}
			l.hydrate(one)
			return one[0], nil
		}
	}
	return Item{}, errors.New("This asset is no longer in the library")
}
func safeName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if !fs.ValidPath(name) || strings.Contains(name, ":") || strings.HasSuffix(name, ".") || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("Invalid asset path: %s", name)
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasSuffix(part, " ") || strings.HasSuffix(part, ".") {
			return "", fmt.Errorf("Ambiguous asset path: %s", name)
		}
	}
	return name, nil
}
func readLimit(name string, limit int64) ([]byte, error) {
	f, e := os.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e == nil && int64(len(b)) > limit {
		e = fmt.Errorf("File exceeds %d MiB", limit>>20)
	}
	return b, e
}
func normalizedJSON(b []byte, v any) error {
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	b, e := hujson.Standardize(b)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func textVersion(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var p []string
		for _, n := range t {
			p = append(p, fmt.Sprint(n))
		}
		return strings.Join(p, ".")
	}
	return ""
}
func fileKind(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".tga":
		return "texture"
	case ".ogg", ".wav", ".mp3":
		return "sound"
	case ".json":
		if strings.Contains(name, "animation") {
			return "animation"
		}
		if strings.Contains(name, "geo") || strings.Contains(name, "models/") {
			return "model"
		}
		return "json"
	case ".lang", ".txt":
		return "text"
	}
	return "file"
}
func imageURL(b []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
}
func checkedPNG(b []byte) (image.Config, error) {
	c, e := png.DecodeConfig(bytes.NewReader(b))
	if e != nil {
		return c, e
	}
	if c.Width <= 0 || c.Height <= 0 || int64(c.Width)*int64(c.Height) > 16<<20 {
		return c, errors.New("Image exceeds 16 megapixels")
	}
	return c, nil
}

func copySource(ctx context.Context, source, dest string) error {
	info, e := os.Lstat(source)
	if e != nil {
		return e
	}
	var total int64
	count := 0
	seen := map[string]bool{}
	write := func(name string, size int64, mode fs.FileMode, open func() (io.ReadCloser, error)) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		name, e = safeName(name)
		if e != nil {
			return e
		}
		if mode&os.ModeSymlink != 0 || !mode.IsRegular() {
			return errors.New("Only regular asset files can be imported")
		}
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("Duplicate file: %s", name)
		}
		seen[key] = true
		count++
		if count > MaxFiles || size < 0 || size > MaxImportBytes-total {
			return errors.New("Asset exceeds 2 GiB or 50,000 files")
		}
		total += size
		p := filepath.Join(dest, filepath.FromSlash(name))
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		in, e := open()
		if e != nil {
			return e
		}
		defer in.Close()
		out, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		n, e := io.Copy(out, io.LimitReader(in, size+1))
		ce := out.Close()
		if e != nil {
			return e
		}
		if n != size {
			return errors.New("Asset size changed during import")
		}
		return ce
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Choose a regular file or folder")
	}
	if info.IsDir() {
		return filepath.WalkDir(source, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if e = ctx.Err(); e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			i, e := d.Info()
			if e != nil {
				return e
			}
			rel, e := filepath.Rel(source, p)
			if e != nil {
				return e
			}
			return write(filepath.ToSlash(rel), i.Size(), i.Mode(), func() (io.ReadCloser, error) { return os.Open(p) })
		})
	}
	if strings.EqualFold(filepath.Ext(source), ".png") {
		return write("skin.png", info.Size(), info.Mode(), func() (io.ReadCloser, error) { return os.Open(source) })
	}
	z, e := zip.OpenReader(source)
	if e != nil {
		return fmt.Errorf("Choose a PNG, mcpack, ZIP or folder: %w", e)
	}
	defer z.Close()
	if len(z.File) > MaxFiles {
		return errors.New("Too many files in archive")
	}
	prefix := ""
	rootManifest := false
	manifests := []string{}
	for _, f := range z.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if name == "manifest.json" {
			rootManifest = true
		}
		if strings.HasSuffix(name, "/manifest.json") {
			manifests = append(manifests, strings.TrimSuffix(name, "manifest.json"))
		}
	}
	if rootManifest {
		prefix = ""
	} else if len(manifests) == 1 {
		prefix = manifests[0]
	} else if len(manifests) > 1 {
		return errors.New("Archive contains multiple packs; import each pack folder separately")
	}
	for _, f := range z.File {
		raw := strings.TrimSuffix(strings.ReplaceAll(f.Name, "\\", "/"), "/")
		if _, e = safeName(raw); e != nil {
			return e
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if prefix != "" && !strings.HasPrefix(raw, prefix) {
			return errors.New("Archive contains files outside the pack folder")
		}
		if f.UncompressedSize64 > uint64(MaxImportBytes) {
			return errors.New("File exceeds import limit")
		}
		if e = write(strings.TrimPrefix(raw, prefix), int64(f.UncompressedSize64), f.Mode(), f.Open); e != nil {
			return e
		}
	}
	return nil
}
func scan(root string) ([]File, int64, string, error) {
	files := []File{}
	var total int64
	h := sha256.New()
	e := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("Asset contains a symbolic link")
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		name := filepath.ToSlash(rel)
		if _, e = safeName(name); e != nil {
			return e
		}
		f, e := os.Open(p)
		if e != nil {
			return e
		}
		sum := sha256.New()
		n, e := io.Copy(sum, io.LimitReader(f, MaxImportBytes-total+1))
		f.Close()
		if e != nil {
			return e
		}
		total += n
		if total > MaxImportBytes || len(files) >= MaxFiles {
			return errors.New("Asset exceeds import limits")
		}
		digest := hex.EncodeToString(sum.Sum(nil))
		files = append(files, File{name, n, digest, fileKind(name)})
		fmt.Fprintf(h, "%s\x00%s\x00", name, digest)
		return nil
	})
	return files, total, hex.EncodeToString(h.Sum(nil)), e
}
func describe(root, kind, name, source string) (Item, error) {
	files, total, id, e := scan(root)
	if e != nil {
		return Item{}, e
	}
	item := Item{ID: id, Kind: kind, Name: name, Source: source, Created: time.Now().UTC().Format(time.RFC3339), Bytes: total, Files: len(files), Tags: []string{}, ArmModel: "wide"}
	item.Encrypted = encryptedPack(root)
	if b, e := readLimit(filepath.Join(root, "manifest.json"), 1<<20); e == nil {
		var m map[string]any
		if normalizedJSON(b, &m) == nil {
			if h, ok := m["header"].(map[string]any); ok {
				if n, ok := h["name"].(string); ok {
					item.Name = n
				}
				item.UUID, _ = h["uuid"].(string)
				item.Version = textVersion(h["version"])
				item.MinimumVersion = textVersion(h["min_engine_version"])
			}
			if modules, ok := m["modules"].([]any); ok {
				for _, v := range modules {
					if mod, ok := v.(map[string]any); ok && mod["type"] == "skin_pack" {
						item.Kind = "skins"
					}
				}
			}
		}
	}
	if item.Kind == "skins" {
		skinFile := "skin.png"
		armDeclared := false
		if b, e := readLimit(filepath.Join(root, "skins.json"), 4<<20); e == nil {
			var s struct {
				Skins []struct {
					Texture  string `json:"texture"`
					Geometry string `json:"geometry"`
				} `json:"skins"`
			}
			if normalizedJSON(b, &s) == nil && len(s.Skins) > 0 {
				skinFile = s.Skins[0].Texture
				armDeclared = s.Skins[0].Geometry != ""
				if strings.Contains(strings.ToLower(s.Skins[0].Geometry), "slim") || strings.Contains(strings.ToLower(s.Skins[0].Geometry), "alex") {
					item.ArmModel = "slim"
				}
			}
		}
		if _, e = safeName(skinFile); e != nil {
			return Item{}, e
		}
		b, e := readLimit(filepath.Join(root, filepath.FromSlash(skinFile)), MaxPreviewBytes)
		if e != nil {
			return Item{}, fmt.Errorf("Skin texture is missing: %w", e)
		}
		c, e := checkedPNG(b)
		if e != nil {
			return Item{}, e
		}
		item.Image = imageURL(b)
		item.TexturePath = skinFile
		item.Width = c.Width
		item.Height = c.Height
		for _, f := range files {
			if f.Path == "geometry.json" {
				armDeclared = true
			}
		}
		if !armDeclared && skinTextureSlim(b) {
			item.ArmModel = "slim"
		}
		if b, e := readLimit(filepath.Join(root, "capture.json"), MaxPreviewBytes); e == nil {
			var meta map[string]any
			if json.Unmarshal(b, &meta) == nil {
				item.Persona, _ = meta["persona"].(bool)
				item.CaptureType = capturedSkinType(meta)
				if a := declaredSkinArms(meta); a != "" {
					item.ArmModel = a
				}
				if skinPatchSlim(meta) {
					item.ArmModel = "slim"
				}
				item.HasAnimations = meta["hasAnimations"] == true
			}
		}
		for _, f := range files {
			item.HasGeometry = item.HasGeometry || f.Path == "geometry.json"
			item.HasCape = item.HasCape || f.Path == "cape.png"
		}
	} else {
		if _, e := os.Stat(filepath.Join(root, "manifest.json")); e != nil {
			return Item{}, errors.New("Resource pack has no manifest.json")
		}
		if b, e := readLimit(filepath.Join(root, "pack_icon.png"), MaxPreviewBytes); e == nil {
			if _, e = checkedPNG(b); e == nil {
				item.Image = imageURL(b)
			}
		}
	}
	if item.UUID != "" {
		lang, _ := readLimit(filepath.Join(root, "texts", "en_US.lang"), 1<<20)
		for _, line := range strings.Split(string(lang), "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if ok && k == item.Name {
				item.Name = v
			}
		}
	}
	return item, nil
}
func (l *Library) Import(ctx context.Context, kind string, paths []string, source string) ([]Item, error) {
	if kind != "skins" && kind != "packs" {
		return nil, errors.New("Choose skins or resource packs")
	}
	if len(paths) > 200 {
		return nil, errors.New("Import up to 200 assets at a time")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	items, e := l.load()
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(filepath.Join(l.Root, "items"), 0700); e != nil {
		return nil, e
	}
	out := []Item{}
	for _, p := range paths {
		stage, e := os.MkdirTemp(l.Root, "import-")
		if e != nil {
			return nil, e
		}
		e = copySource(ctx, p, stage)
		if e != nil {
			os.RemoveAll(stage)
			return out, e
		}
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		prepared, e := splitSkinPack(ctx, stage)
		if e != nil {
			os.RemoveAll(stage)
			return out, e
		}
		if len(prepared) == 0 {
			prepared = []preparedSkin{{stage, name}}
		} else {
			original, e := l.preserveOriginal(ctx, stage)
			if e != nil {
				os.RemoveAll(stage)
				for _, p := range prepared {
					os.RemoveAll(p.directory)
				}
				return out, e
			}
			for _, p := range prepared {
				if e = os.WriteFile(filepath.Join(p.directory, "original-pack-ref.txt"), []byte(original), 0600); e != nil {
					os.RemoveAll(stage)
					for _, p := range prepared {
						os.RemoveAll(p.directory)
					}
					return out, e
				}
			}
			os.RemoveAll(stage)
		}
		for _, entry := range prepared {
			defer os.RemoveAll(entry.directory)
		}
		for _, entry := range prepared {
			stage = entry.directory
			entryKind := kind
			if len(prepared) > 1 {
				entryKind = "skins"
			}
			if _, statErr := os.Stat(filepath.Join(stage, "skin.png")); statErr == nil {
				entryKind = "skins"
			}
			v, e := describe(stage, entryKind, entry.name, source)
			if e != nil {
				os.RemoveAll(stage)
				return out, e
			}
			if ref, e := readLimit(filepath.Join(stage, "original-pack-ref.txt"), 128); e == nil && validID(string(ref)) {
				v.Original = string(ref)
			}
			duplicate := false
			for _, old := range items {
				if old.ID == v.ID {
					v = old
					duplicate = true
					break
				}
			}
			if duplicate {
				os.RemoveAll(stage)
			} else {
				obsolete := []string{}
				if v.Kind == "packs" && !v.Encrypted && v.UUID != "" {
					kept := []Item{}
					for _, old := range items {
						if old.Encrypted && old.UUID == v.UUID && old.Version == v.Version {
							obsolete = append(obsolete, old.ID)
							v.Favourite = old.Favourite
							v.Tags = old.Tags
							if old.Name != "" {
								v.Name = old.Name
							}
						} else {
							kept = append(kept, old)
						}
					}
					items = kept
				}
				dest, _ := l.RootFor(v.ID)
				if _, e = os.Stat(dest); os.IsNotExist(e) {
					e = os.Rename(stage, dest)
				} else {
					e = errors.New("Asset directory already exists without an index entry")
				}
				if e != nil {
					os.RemoveAll(stage)
					return out, e
				}
				items = append(items, v)
				if e = l.save(items); e != nil {
					os.RemoveAll(dest)
					return out, e
				}
				for _, id := range obsolete {
					p, _ := l.RootFor(id)
					if e = os.RemoveAll(p); e != nil {
						return out, e
					}
				}
			}
			out = append(out, v)
		}
	}
	return out, nil
}
func (l *Library) Update(id, name string, favourite bool, tags []string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 160 || len(tags) > 20 {
		return errors.New("Enter a name up to 160 characters and up to 20 tags")
	}
	for _, v := range tags {
		if len(v) > 40 {
			return errors.New("Tags must be 40 characters or fewer")
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	items, e := l.load()
	if e != nil {
		return e
	}
	for i := range items {
		if items[i].ID == id {
			items[i].Name = name
			items[i].Favourite = favourite
			items[i].Tags = tags
			return l.save(items)
		}
	}
	return errors.New("Asset not found")
}
func (l *Library) Delete(ids []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	items, e := l.load()
	if e != nil {
		return e
	}
	selected := map[string]bool{}
	for _, id := range ids {
		if !validID(id) {
			return errors.New("Invalid asset")
		}
		selected[id] = true
	}
	kept := []Item{}
	removed := []Item{}
	for _, v := range items {
		if selected[v.ID] {
			removed = append(removed, v)
		} else {
			kept = append(kept, v)
		}
	}
	for _, v := range removed {
		root, _ := l.RootFor(v.ID)
		if e = os.RemoveAll(root); e != nil {
			return e
		}
	}
	if e = l.save(kept); e != nil {
		return e
	}
	used := map[string]bool{}
	for _, v := range kept {
		used[v.Original] = true
	}
	for _, v := range removed {
		if v.Original != "" && !used[v.Original] {
			if p, e := l.originalPath(v.Original); e == nil {
				if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
					return e
				}
			}
			used[v.Original] = true
		}
	}
	return nil
}
func (l *Library) Read(id, name string) (Preview, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	item, e := l.get(id)
	if e != nil {
		return Preview{}, e
	}
	if item.Encrypted && name != "pack_icon.png" && name != "manifest.json" {
		return Preview{}, errors.New("This pack contains encrypted files. Capture the server again to add its readable session copy")
	}
	root, _ := l.RootFor(id)
	if strings.HasPrefix(name, capturedBlockPrefix) {
		return l.capturedDefinitionPreview(id, name)
	}
	return readAssetFile(root, name)
}
func readAssetFile(root, name string) (Preview, error) {
	name, e := safeName(name)
	if e != nil {
		return Preview{}, e
	}
	file := filepath.Join(root, filepath.FromSlash(name))
	i, e := os.Lstat(file)
	if e != nil {
		return Preview{}, e
	}
	if !i.Mode().IsRegular() {
		return Preview{}, errors.New("Choose a regular file")
	}
	kind := fileKind(name)
	limit := int64(MaxPreviewBytes)
	if kind == "json" || kind == "text" || kind == "model" || kind == "animation" {
		limit = 1 << 20
	}
	if name == "animations.json" {
		limit = MaxPreviewBytes
	}
	b, e := readLimit(file, limit)
	if e != nil {
		return Preview{}, e
	}
	result := Preview{Path: name, Kind: kind}
	switch kind {
	case "texture":
		if strings.EqualFold(filepath.Ext(name), ".tga") {
			if len(b) < 18 || uint64(binary.LittleEndian.Uint16(b[12:14]))*uint64(binary.LittleEndian.Uint16(b[14:16])) > 16<<20 {
				return Preview{}, errors.New("TGA image exceeds the preview limit")
			}
			img, e := tga.Decode(bytes.NewReader(b))
			if e != nil {
				return Preview{}, e
			}
			var encoded bytes.Buffer
			if e = png.Encode(&encoded, img); e != nil {
				return Preview{}, e
			}
			result.Data = imageURL(encoded.Bytes())
			return result, nil
		}
		c, _, e := image.DecodeConfig(bytes.NewReader(b))
		if e != nil {
			return Preview{}, fmt.Errorf("%s is not a readable image: %w", name, e)
		}
		if int64(c.Width)*int64(c.Height) > 16<<20 {
			return Preview{}, fmt.Errorf("%s exceeds the 16 megapixel preview limit", name)
		}
		mime := "image/png"
		if !strings.EqualFold(filepath.Ext(name), ".png") {
			mime = "image/jpeg"
		}
		result.Data = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
	case "sound":
		mime := "audio/ogg"
		if strings.EqualFold(filepath.Ext(name), ".wav") {
			mime = "audio/wav"
		}
		if strings.EqualFold(filepath.Ext(name), ".mp3") {
			mime = "audio/mpeg"
		}
		result.Data = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
	case "json", "model", "animation", "text":
		if kind != "text" {
			normalized, err := hujson.Standardize(b)
			if err != nil {
				return Preview{}, fmt.Errorf("%s is not readable JSON; it may be encrypted or damaged", name)
			}
			var formatted bytes.Buffer
			if err := json.Indent(&formatted, normalized, "", "  "); err != nil {
				return Preview{}, err
			}
			b = formatted.Bytes()
		}
		result.Data = string(b)
	default:
		return Preview{}, errors.New("This file has no preview. Export it to open it in another app")
	}
	return result, nil
}
func (l *Library) Detail(id string) (Detail, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	item, e := l.get(id)
	if e != nil {
		return Detail{}, e
	}
	root, _ := l.RootFor(id)
	files, _, _, e := scan(root)
	if e != nil {
		return Detail{}, e
	}
	d := Detail{Item: item, Files: files, Issues: []Issue{}, Metadata: map[string]any{}}
	validate(root, &d)
	if item.Kind == "packs" {
		if x, err := l.catalog(id); err == nil {
			appendCapturedDefinitionFiles(x, &d)
		}
	}
	if item.Kind == "skins" {
		d.Metadata["equipment"] = l.capturedEquipment(d.Metadata)
	}
	return d, nil
}
func (l *Library) Compare(left, right string) ([]Change, error) {
	a, e := l.Detail(left)
	if e != nil {
		return nil, e
	}
	b, e := l.Detail(right)
	if e != nil {
		return nil, e
	}
	m := map[string]string{}
	for _, f := range a.Files {
		m[f.Path] = f.Hash
	}
	out := []Change{}
	for _, f := range b.Files {
		old, ok := m[f.Path]
		if !ok {
			out = append(out, Change{f.Path, "added"})
		} else if old != f.Hash {
			out = append(out, Change{f.Path, "modified"})
		}
		delete(m, f.Path)
	}
	for p := range m {
		out = append(out, Change{p, "removed"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
func (l *Library) Stack(ids []string) (Stack, error) {
	out := Stack{Conflicts: []Conflict{}, Issues: []Issue{}, Resolved: map[string]string{}}
	byPath := map[string][]string{}
	available := map[string]string{}
	var details []Detail
	for _, id := range ids {
		d, e := l.Detail(id)
		if e != nil {
			return out, e
		}
		if d.Item.Kind != "packs" {
			return out, errors.New("Only resource packs can be stacked")
		}
		details = append(details, d)
		available[d.Item.UUID] = d.Item.Version
		for _, f := range d.Files {
			if f.Path == "manifest.json" || f.Path == "pack_icon.png" || strings.HasPrefix(f.Path, capturedBlockPrefix) {
				continue
			}
			byPath[f.Path] = append(byPath[f.Path], id)
			if out.Resolved[f.Path] == "" {
				out.Resolved[f.Path] = id
			}
		}
	}
	for p, all := range byPath {
		if len(all) > 1 {
			out.Conflicts = append(out.Conflicts, Conflict{p, all[0], all})
		}
	}
	sort.Slice(out.Conflicts, func(i, j int) bool { return out.Conflicts[i].Path < out.Conflicts[j].Path })
	for _, d := range details {
		if deps, ok := d.Metadata["dependencies"].([]any); ok {
			for _, v := range deps {
				dep, ok := v.(map[string]any)
				if !ok {
					continue
				}
				id, _ := dep["uuid"].(string)
				if id != "" {
					if version, ok := available[id]; !ok || version != textVersion(dep["version"]) {
						out.Issues = append(out.Issues, Issue{"warning", "dependency", "manifest.json", "Required pack or version is missing from this stack", id})
					}
				}
			}
		}
	}
	return out, nil
}
