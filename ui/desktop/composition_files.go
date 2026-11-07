package desktop

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/utils"
)

func compositionArchive(ctx context.Context, folder, path string) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".craftbox-compose-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	writer := zip.NewWriter(file)
	err = filepath.WalkDir(folder, func(source string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("Unsupported archive file")
		}
		relative, err := filepath.Rel(folder, source)
		if err != nil {
			return err
		}
		out, err := writer.Create(filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		in, err := os.Open(source)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(out, renderContextReader{ctx: ctx, reader: in})
		return err
	})
	err = errors.Join(err, writer.Close(), file.Close())
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

type compositionPackManifest struct {
	Header struct {
		UUID    string `json:"uuid"`
		Version [3]int `json:"version"`
	} `json:"header"`
}

func compositionPackHash(ctx context.Context, root string) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("Linked pack files cannot be merged")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		io.WriteString(hash, filepath.ToSlash(rel)+"\x00")
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(hash, renderContextReader{ctx: ctx, reader: in})
		return err
	})
	return hex.EncodeToString(hash.Sum(nil)), err
}
func copyIdentityPacks(ctx context.Context, source, target, kind string) error {
	paths, err := renderPackPaths(source, kind, "world_"+kind+".json")
	if err != nil {
		return err
	}
	for _, src := range paths {
		if err = ctx.Err(); err != nil {
			return err
		}
		entry, e := os.Lstat(src)
		if e != nil {
			return e
		}
		if !entry.IsDir() {
			if !entry.Mode().IsRegular() {
				return fmt.Errorf("Linked packs cannot be merged")
			}
			root, temporary, e := extractRenderArchive(ctx, src)
			if e != nil {
				return e
			}
			defer os.RemoveAll(temporary)
			src = root
		}
		raw, err := os.ReadFile(filepath.Join(src, "manifest.json"))
		if err != nil {
			return err
		}
		var manifest compositionPackManifest
		if err = utils.ParseJson(raw, &manifest); err != nil {
			return err
		}
		id := manifest.Header.UUID
		if id == "" {
			return fmt.Errorf("Pack has no identifier")
		}
		v := manifest.Header.Version
		name := utils.MakeValidFilename(id) + fmt.Sprintf("_%d.%d.%d", v[0], v[1], v[2])
		dest := filepath.Join(target, kind, name)
		existing, _ := os.ReadDir(filepath.Join(target, kind))
		for _, other := range existing {
			data, e := os.ReadFile(filepath.Join(target, kind, other.Name(), "manifest.json"))
			if e != nil {
				return e
			}
			var m compositionPackManifest
			if e = utils.ParseJson(data, &m); e != nil {
				return e
			}
			if strings.EqualFold(m.Header.UUID, id) {
				if m.Header.Version != v {
					return fmt.Errorf("Pack %s uses different versions in these sources", id)
				}
				a, e := compositionPackHash(ctx, src)
				if e != nil {
					return e
				}
				b, e := compositionPackHash(ctx, filepath.Join(target, kind, other.Name()))
				if e != nil {
					return e
				}
				if a != b {
					return fmt.Errorf("Pack %s has conflicting content in these sources", id)
				}
				dest = ""
				break
			}
		}
		if dest == "" {
			continue
		}
		err = filepath.WalkDir(src, func(path string, item fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			to := filepath.Join(dest, rel)
			if item.IsDir() {
				return os.MkdirAll(to, 0700)
			}
			if !item.Type().IsRegular() {
				return fmt.Errorf("Linked packs cannot be merged")
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, renderContextReader{ctx: ctx, reader: in})
			return errors.Join(err, out.Close())
		})
		if err != nil {
			return err
		}
	}
	return nil
}
