package assets

import (
	"archive/zip"
	"crypto/sha256"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"sync"
)

//go:embed bundle/*
var bundled embed.FS
var once sync.Once
var vanilla *zip.Reader
var vanillaErr error
var vanillaDigest string

func Vanilla() (fs.FS, error) {
	once.Do(func() {
		file, err := bundled.Open("bundle/vanilla-3d.zip")
		if err != nil {
			vanillaErr = err
			return
		}

		reader, ok := file.(io.ReaderAt)
		if !ok {
			vanillaErr = fmt.Errorf("embedded asset archive does not support random access")
			return
		}
		info, err := file.Stat()
		if err != nil {
			vanillaErr = err
			return
		}
		digest := sha256.New()
		if _, err = io.Copy(digest, file); err != nil {
			vanillaErr = err
			return
		}
		vanillaDigest = fmt.Sprintf("%x", digest.Sum(nil))
		vanilla, vanillaErr = zip.NewReader(reader, info.Size())
	})
	if vanillaErr != nil {
		return nil, vanillaErr
	}
	return vanilla, nil
}

func Restore(root string) error {
	source, err := Vanilla()
	if err != nil {
		return err
	}
	return restore(source, root, vanillaDigest)
}
