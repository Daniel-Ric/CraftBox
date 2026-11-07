package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func run() error {
	source := "assets/vanilla-3d"
	if _, err := os.Stat(filepath.Join(source, "provenance.json")); err != nil {
		return fmt.Errorf("prepare vanilla assets first with tools/prepare-3d-assets.py: %w", err)
	}
	if _, err := os.Stat(filepath.Join(source, "font", "Mojangles.ttf")); err != nil {
		return fmt.Errorf("prepare the Minecraft sign font with tools/prepare-3d-assets.py --font <local Mojangles.ttf>: %w", err)
	}
	const target = "assets/bundle/vanilla-3d.zip"
	file, err := os.Create(target + ".tmp")
	if err != nil {
		return err
	}
	defer os.Remove(target + ".tmp")
	zw := zip.NewWriter(file)
	err = fs.WalkDir(os.DirFS(source), ".", func(name string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("non-regular asset: %s", name)
		}
		input, err := os.Open(filepath.Join(source, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			return err
		}
		_, err = io.Copy(output, input)
		return err
	})
	closeZip := zw.Close()
	closeFile := file.Close()
	if err != nil {
		return err
	}
	if closeZip != nil {
		return closeZip
	}
	if closeFile != nil {
		return closeFile
	}
	return os.Rename(target+".tmp", target)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
