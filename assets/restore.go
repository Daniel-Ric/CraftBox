package assets

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
)

func restore(source fs.FS, directory, digest string) error {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	lock, err := root.OpenFile(".craftbox-assets.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = lockAssets(lock); err != nil {
		return err
	}
	const marker = ".craftbox-bundle"
	previous, _ := root.ReadFile(marker)
	changed := string(previous) != digest
	err = fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if !fs.ValidPath(name) {
			return fmt.Errorf("invalid bundled asset: %q", name)
		}
		if entry.IsDir() {
			return root.MkdirAll(name, 0700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular bundled asset: %s", name)
		}
		if existing, statErr := root.Lstat(name); statErr == nil {
			if !existing.Mode().IsRegular() {
				return fmt.Errorf("non-regular installed asset: %s", name)
			}
			if !changed && existing.Size() == info.Size() {
				return nil
			}
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return statErr
		}
		if err := root.MkdirAll(path.Dir(name), 0700); err != nil {
			return err
		}
		input, err := source.Open(name)
		if err != nil {
			return err
		}
		defer input.Close()
		temporary := path.Join(path.Dir(name), ".craftbox-"+rand.Text()+".tmp")
		output, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		defer root.Remove(temporary)
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return root.Rename(temporary, name)
	})
	if err != nil {
		return err
	}
	temporary := ".craftbox-" + rand.Text() + ".tmp"
	defer root.Remove(temporary)
	if err = root.WriteFile(temporary, []byte(digest), 0600); err != nil {
		return err
	}
	return root.Rename(temporary, marker)
}
