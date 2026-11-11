package bedrocklab

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) preserveSchemas(schemas *Schemas) ([]File, error) {
	files := make([]File, 0, len(schemas.documents))
	for i, doc := range schemas.documents {
		b, err := json.Marshal(doc.Schema)
		if err != nil {
			return nil, err
		}
		hash, err := s.put(b)
		if err != nil {
			return nil, err
		}

		rel, err := filepath.Rel(schemas.root, doc.File)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if !safePath(rel) {
			return nil, fmt.Errorf("unsafe schema document %d", i)
		}
		files = append(files, File{Path: rel, Hash: hash, Bytes: int64(len(b)), Kind: "json"})
	}
	return files, nil
}

func (s *Store) Redecode(ctx context.Context, id, schemaPath string, progress func(Progress)) (Snapshot, error) {
	d, err := s.Get(id)
	if err != nil {
		return d, err
	}
	if schemaPath == "" {
		if len(d.SchemaObjects) == 0 {
			return d, errors.New("select matching BDS schemas; this snapshot has no embedded schemas")
		}
		temp, err := os.MkdirTemp("", "craftbox-schemas-*")
		if err != nil {
			return d, err
		}
		defer os.RemoveAll(temp)
		for _, f := range d.SchemaObjects {
			if err := ctx.Err(); err != nil {
				return d, err
			}
			if !safePath(f.Path) {
				return d, errors.New("unsafe embedded schema path")
			}
			b, err := s.read(f, maxFile+1)
			if err != nil {
				return d, err
			}
			if digest(b) != f.Hash {
				return d, errors.New("embedded schema integrity mismatch")
			}
			p := filepath.Join(temp, filepath.FromSlash(f.Path))
			if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				return d, err
			}
			if err = os.WriteFile(p, b, 0600); err != nil {
				return d, err
			}
		}
		schemaPath = temp
	} else {
		if b, err := os.ReadFile(filepath.Join(schemaPath, "craftbox-schema.json")); err == nil {
			var bundle SchemaBundle
			if err = json.Unmarshal(b, &bundle); err != nil {
				return d, err
			}
			game, build := strings.Split(d.Version, "."), strings.Split(bundle.Version, ".")
			if len(game) >= 3 && len(build) >= 3 && strings.Join(game[:3], ".") != strings.Join(build[:3], ".") {
				return d, fmt.Errorf("BDS schema version %s does not match game data %s", bundle.Version, d.Version)
			}
			d.SchemaVersion, d.SchemaHash = bundle.Version, bundle.Hash
		} else {
			d.SchemaVersion, d.SchemaHash = "", ""
		}
		d.SchemaSource = schemaPath
	}
	schemas, err := LoadSchemas(schemaPath)
	if err != nil {
		return d, err
	}
	d.SchemaObjects, err = s.preserveSchemas(schemas)
	if err != nil {
		return d, err
	}
	d.ParentID = d.ID
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return d, err
	}
	d.ID = hex.EncodeToString(random)
	d.Name += " (decoded)"
	d.Created = time.Now().UTC().Format(time.RFC3339Nano)
	d.Decoded, d.Unresolved = 0, 0
	d.Bytes = 0
	for i := range d.Files {
		if err = ctx.Err(); err != nil {
			return d, err
		}
		f := &d.Files[i]
		if f.Kind == "mcb" || f.RawHash != "" {
			if progress != nil {
				progress(Progress{Running: true, Phase: "Restoring MCB", Path: f.Path, Done: i, Total: len(d.Files)})
			}
			original := *f
			if f.RawHash != "" {
				original.Hash = f.RawHash
			}
			raw, err := s.read(original, maxFile+1)
			if err != nil {
				return d, err
			}
			if int64(len(raw)) > maxFile || digest(raw) != original.Hash {
				return d, fmt.Errorf("compiled original integrity mismatch: %s", f.Path)
			}
			decoded, decodeErr := schemas.Decode(raw)
			if decodeErr != nil {
				f.Kind, f.Hash, f.RawHash, f.Semantic, f.Bytes = "mcb", original.Hash, "", "", int64(len(raw))
				f.Identifiers = []string{}
				f.Issue = "MCB preserved: " + decodeErr.Error()
				d.Unresolved++
			} else {
				hash, err := s.put(decoded)
				if err != nil {
					return d, err
				}
				canonical, err := canonicalJSON(decoded)
				if err != nil {
					return d, err
				}
				f.Kind, f.Hash, f.RawHash, f.Semantic, f.Bytes = "json", hash, original.Hash, digest(canonical), int64(len(decoded))
				f.Identifiers = uniqueIdentifiers(canonical)
				f.Issue = ""
				d.Decoded++
			}
		}
		d.Bytes += f.Bytes
	}
	return s.saveSnapshot(ctx, d)
}
