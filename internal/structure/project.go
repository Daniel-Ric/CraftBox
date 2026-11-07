package structure

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type projectManifest struct {
	Format  string         `json:"format"`
	Version int            `json:"version"`
	Name    string         `json:"name"`
	Size    Vec3           `json:"size"`
	View    map[string]any `json:"view"`
	Packs   []Dependency   `json:"packs"`
}

func sparseNBT(d *Document) map[string]any {
	states := []map[string]any{}
	for _, s := range d.Palette {
		states = append(states, map[string]any{"name": s.Name, "states": s.States, "version": s.Version})
	}
	sections := []map[string]any{}
	keys := make([]string, 0, len(d.Sections))
	for key := range d.Sections {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		s := d.Sections[key]
		sections = append(sections, map[string]any{"key": key, "layers": [][]int32{s.Layers[0], s.Layers[1]}, "known": s.Known})
	}
	regions := []map[string]any{}
	for _, r := range d.Regions {
		regions = append(regions, map[string]any{"id": r.ID, "name": r.Name, "min": r.Bounds.Min[:], "max": r.Bounds.Max[:]})
	}
	known := uint8(0)
	if d.DefaultKnown {
		known = 1
	}
	return map[string]any{"version": int32(1), "size": d.Size[:], "origin": d.Origin[:], "anchor": d.Anchor[:], "default": []int32{d.Default.Layers[0], d.Default.Layers[1]}, "default_known": known, "palette": states, "sections": sections, "block_data": d.BlockData, "entities": d.Entities, "regions": regions, "root_extra": d.RootExtra, "structure_extra": d.StructureExtra, "palette_extra": d.PaletteExtra}
}
func restoreSparse(root map[string]any, name string) (*Document, error) {
	if v, _ := integer(root["version"]); v != 1 {
		return nil, fmt.Errorf("Unsupported Structure Studio data version")
	}
	size, err := vector(root["size"])
	if err != nil {
		return nil, err
	}
	d, err := New(name, size)
	if err != nil {
		return nil, err
	}
	d.Origin, err = vector(root["origin"])
	if err != nil {
		return nil, err
	}
	d.Anchor, err = vector(root["anchor"])
	if err != nil {
		return nil, err
	}
	a, err := indices(root["default"])
	if err != nil || len(a) != 2 {
		return nil, fmt.Errorf("Invalid default cell")
	}
	d.Default.Layers = [2]int32{a[0], a[1]}
	k, _ := root["default_known"].(uint8)
	d.DefaultKnown = k != 0
	d.Default.Known = d.DefaultKnown
	d.Palette = nil
	for _, v := range list(root["palette"]) {
		m := compound(v)
		s := State{States: compound(m["states"])}
		s.Name, _ = m["name"].(string)
		s.Version, _ = integer(m["version"])
		s, err = NormalizeState(s)
		if err != nil {
			return nil, err
		}
		d.Palette = append(d.Palette, s)
	}
	for _, v := range list(root["sections"]) {
		m := compound(v)
		key, _ := m["key"].(string)
		layers := list(m["layers"])
		if len(layers) != 2 {
			return nil, fmt.Errorf("Invalid saved section")
		}
		a, err := indices(layers[0])
		if err != nil {
			return nil, err
		}
		b, err := indices(layers[1])
		if err != nil {
			return nil, err
		}
		known, _ := m["known"].([]uint8)
		if _, exists := d.Sections[key]; exists {
			return nil, fmt.Errorf("Duplicate structure section")
		}
		d.Sections[key] = &Section{Layers: [2][]int32{a, b}, Known: known}
	}
	for key, v := range compound(root["block_data"]) {
		d.BlockData[key] = compound(v)
	}
	for _, v := range list(root["entities"]) {
		if e := compound(v); !isPlayer(e) {
			d.Entities = append(d.Entities, e)
		}
	}
	for _, v := range list(root["regions"]) {
		m := compound(v)
		r := Region{}
		r.ID, _ = m["id"].(string)
		r.Name, _ = m["name"].(string)
		r.Bounds.Min, err = vector(m["min"])
		if err != nil {
			return nil, err
		}
		r.Bounds.Max, err = vector(m["max"])
		if err != nil {
			return nil, err
		}
		d.Regions = append(d.Regions, r)
	}
	d.RootExtra = compound(root["root_extra"])
	d.StructureExtra = compound(root["structure_extra"])
	d.PaletteExtra = compound(root["palette_extra"])
	return d, d.Validate()
}
func safeArchiveName(name string) bool {
	return fs.ValidPath(name) && !strings.Contains(name, "\\") && !strings.Contains(name, ":") && !strings.HasPrefix(name, "/")
}
func WriteProject(ctx context.Context, w io.Writer, d *Document, view map[string]any, includePacks bool) (result error) {
	if err := d.Validate(); err != nil {
		return err
	}
	z := zip.NewWriter(w)
	defer func() {
		if err := z.Close(); result == nil {
			result = err
		}
	}()
	manifest := projectManifest{Format: "craftbox-structure", Version: 1, Name: d.Name, Size: d.Size, View: view, Packs: append([]Dependency(nil), d.Packs...)}
	for i, p := range manifest.Packs {
		manifest.Packs[i].Source = ""
		manifest.Packs[i].Missing = !includePacks || p.Source == "" || p.Missing
	}
	f, err := z.Create("project.json")
	if err != nil {
		return err
	}
	if err = json.NewEncoder(f).Encode(manifest); err != nil {
		return err
	}
	f, err = z.Create("document.nbt")
	if err != nil {
		return err
	}
	if err = nbt.NewEncoderWithEncoding(&contextWriter{ctx, f}, nbt.LittleEndian).Encode(sparseNBT(d)); err != nil {
		return err
	}
	var total int64
	count := 0
	if includePacks {
		for i, p := range d.Packs {
			if p.Source == "" || p.Missing {
				continue
			}
			prefix := fmt.Sprintf("packs/%d", i)
			info, err := os.Stat(p.Source)
			if err != nil {
				return fmt.Errorf("Required pack %s is unavailable: %w", p.Name, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("Pack %s must be imported before saving a portable project", p.Name)
			}
			err = filepath.WalkDir(p.Source, func(file string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				if entry.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("Pack contains a symbolic link")
				}
				if entry.IsDir() {
					return nil
				}
				relative, err := filepath.Rel(p.Source, file)
				if err != nil {
					return err
				}
				name := path.Join(prefix, filepath.ToSlash(relative))
				if !safeArchiveName(name) {
					return fmt.Errorf("Unsafe resource path")
				}
				info, err := entry.Info()
				if err != nil {
					return err
				}
				total += info.Size()
				count++
				if total > 2<<30 || count > 100000 {
					return fmt.Errorf("Portable resources exceed 2 GiB or 100,000 files")
				}
				out, err := z.Create(name)
				if err != nil {
					return err
				}
				in, err := os.Open(file)
				if err != nil {
					return err
				}
				_, err = io.Copy(out, &contextReader{ctx: ctx, r: in})
				closeErr := in.Close()
				if err != nil {
					return err
				}
				return closeErr
			})
			if err != nil {
				return err
			}
		}
	}
	return ctx.Err()
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}
type contextWriter struct {
	ctx context.Context
	w   io.Writer
}

func (w *contextWriter) Write(b []byte) (int, error) {
	if e := w.ctx.Err(); e != nil {
		return 0, e
	}
	return w.w.Write(b)
}

type ProjectBrief struct {
	Name string `json:"name"`
	Size Vec3   `json:"size"`
}

func ReadProjectBrief(path string) (ProjectBrief, error) {
	z, e := zip.OpenReader(path)
	if e != nil {
		return ProjectBrief{}, e
	}
	defer z.Close()
	if len(z.File) > 100002 {
		return ProjectBrief{}, fmt.Errorf("Too many project entries")
	}
	for _, f := range z.File {
		if f.Name != "project.json" {
			continue
		}
		if f.UncompressedSize64 > 1<<20 {
			return ProjectBrief{}, fmt.Errorf("Project manifest too large")
		}
		in, e := f.Open()
		if e != nil {
			return ProjectBrief{}, e
		}
		defer in.Close()
		var m projectManifest
		if e = json.NewDecoder(io.LimitReader(in, 1<<20)).Decode(&m); e != nil {
			return ProjectBrief{}, e
		}
		if m.Format != "craftbox-structure" || m.Version != 1 {
			return ProjectBrief{}, fmt.Errorf("Unsupported structure project")
		}
		if e = ValidateName(m.Name); e != nil {
			return ProjectBrief{}, e
		}
		return ProjectBrief{m.Name, m.Size}, nil
	}
	return ProjectBrief{}, fmt.Errorf("Missing structure manifest")
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
func ReadProject(ctx context.Context, r io.ReaderAt, size int64, ownedDirectory string) (*Document, map[string]any, error) {
	z, err := zip.NewReader(r, size)
	if err != nil {
		return nil, nil, err
	}
	if len(z.File) > 100002 {
		return nil, nil, fmt.Errorf("Project has too many files")
	}
	files := map[string]*zip.File{}
	var total uint64
	for _, f := range z.File {
		if !safeArchiveName(strings.TrimSuffix(f.Name, "/")) || f.Mode()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("Unsafe project archive entry")
		}
		if _, exists := files[f.Name]; exists {
			return nil, nil, fmt.Errorf("Duplicate project archive entry")
		}
		if f.UncompressedSize64 > uint64(2<<30)-total {
			return nil, nil, fmt.Errorf("Project expands beyond 2 GiB")
		}
		total += f.UncompressedSize64
		files[f.Name] = f
	}
	read := func(name string, limit int64) ([]byte, error) {
		f := files[name]
		if f == nil {
			return nil, fmt.Errorf("Project is missing %s", name)
		}
		if f.UncompressedSize64 > uint64(limit) {
			return nil, fmt.Errorf("Project entry is too large")
		}
		reader, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(io.LimitReader(&contextReader{ctx: ctx, r: reader}, limit+1))
	}
	raw, err := read("project.json", 2<<20)
	if err != nil {
		return nil, nil, err
	}
	var m projectManifest
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, nil, err
	}
	if m.Format != "craftbox-structure" || m.Version != 1 {
		return nil, nil, fmt.Errorf("This is not a supported Structure Studio project")
	}
	if len(m.Packs) > 128 {
		return nil, nil, fmt.Errorf("Too many project pack dependencies")
	}
	raw, err = read("document.nbt", MaxNBTBytes)
	if err != nil {
		return nil, nil, err
	}
	root, err := readNBT(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, err
	}
	doc, err := restoreSparse(root, m.Name)
	if err != nil {
		return nil, nil, err
	}
	for i, p := range m.Packs {
		prefix := fmt.Sprintf("packs/%d/", i)
		p.Source = filepath.Join(ownedDirectory, "packs", fmt.Sprint(i))
		found := false
		for _, f := range z.File {
			if !strings.HasPrefix(f.Name, prefix) || f.FileInfo().IsDir() {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
			relative := strings.TrimPrefix(f.Name, prefix)
			target := filepath.Join(p.Source, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return nil, nil, err
			}
			in, err := f.Open()
			if err != nil {
				return nil, nil, err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				in.Close()
				return nil, nil, err
			}
			_, copyErr := io.Copy(out, &contextReader{ctx, in})
			inErr := in.Close()
			outErr := out.Close()
			if copyErr != nil {
				return nil, nil, copyErr
			}
			if inErr != nil {
				return nil, nil, inErr
			}
			if outErr != nil {
				return nil, nil, outErr
			}
			found = true
		}
		p.Missing = !found
		doc.Packs = append(doc.Packs, p)
	}
	return doc, m.View, nil
}
func WriteAtomic(path string, write func(io.Writer) error) (result error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".structure-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = write(f); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temp, path)
}
