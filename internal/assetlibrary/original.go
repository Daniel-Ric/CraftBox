package assetlibrary

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (l *Library) preserveOriginal(ctx context.Context, root string) (string, error) {
	files, _, id, e := scan(root)
	if e != nil {
		return "", e
	}
	parent := filepath.Join(l.Root, "originals")
	if e = os.MkdirAll(parent, 0700); e != nil {
		return "", e
	}
	dest := filepath.Join(parent, id+".mcpack")
	if _, e = os.Stat(dest); e == nil {
		return id, nil
	}
	f, e := os.CreateTemp(parent, "pack-*")
	if e != nil {
		return "", e
	}
	defer os.Remove(f.Name())
	z := zip.NewWriter(f)
	for _, file := range files {
		if e = ctx.Err(); e != nil {
			break
		}
		var out io.Writer
		out, e = z.CreateHeader(&zip.FileHeader{Name: file.Path, Method: zip.Store})
		if e != nil {
			break
		}
		in, openErr := os.Open(filepath.Join(root, filepath.FromSlash(file.Path)))
		if openErr != nil {
			e = openErr
			break
		}
		_, e = io.Copy(out, in)
		in.Close()
		if e != nil {
			break
		}
	}
	ze := z.Close()
	ce := f.Close()
	if e == nil {
		e = ze
	}
	if e == nil {
		e = ce
	}
	if e != nil {
		return "", e
	}
	if e = os.Rename(f.Name(), dest); e != nil {
		return "", e
	}
	return id, nil
}
func (l *Library) originalPath(id string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("Invalid original pack reference")
	}
	return filepath.Join(l.Root, "originals", id+".mcpack"), nil
}
