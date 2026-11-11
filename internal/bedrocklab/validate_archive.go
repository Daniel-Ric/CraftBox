package bedrocklab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path"
	"sort"
)

func (s *Store) validateArchiveMembers(ctx context.Context, d Snapshot, v *Validation) error {
	files := map[string]File{}
	groups := map[string][]File{}
	for _, f := range d.Files {
		files[f.Path] = f
		if f.Origin != "" {
			groups[f.Origin] = append(groups[f.Origin], f)
		}
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		archive, ok := files[name]
		if !ok {
			v.Issues = append(v.Issues, "Archive provenance unavailable: "+name)
			continue
		}
		p, err := s.object(archive.Hash)
		if err != nil {
			return err
		}
		r, err := os.Open(p)
		if err != nil {
			return err
		}
		info, err := r.Stat()
		if err != nil {
			r.Close()
			return err
		}
		entries, err := ReadArchive(r, info.Size())
		if err != nil {
			r.Close()
			continue
		}
		index := map[string]ArchiveEntry{}
		for _, entry := range entries {
			index[path.Join(archiveDestination(name), entry.Path)] = entry
		}
		for _, f := range groups[name] {
			if err = ctx.Err(); err != nil {
				r.Close()
				return err
			}
			entry, ok := index[f.Path]
			if !ok {
				v.Issues = append(v.Issues, "Archive member missing: "+f.Path)
				continue
			}
			hash := sha256.New()
			if _, err = io.Copy(hash, io.NewSectionReader(r, entry.Offset, entry.Size)); err != nil {
				r.Close()
				return err
			}
			expected := f.Hash
			if f.RawHash != "" {
				expected = f.RawHash
			}
			if hex.EncodeToString(hash.Sum(nil)) != expected {
				v.Issues = append(v.Issues, "Archive member mismatch: "+f.Path)
			}
		}
		r.Close()
	}
	return nil
}
