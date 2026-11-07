package desktop

import (
	"errors"
	"os"
	"strings"
)

func (a *App) ValidateLocalPath(path, kind string) error {
	if kind != "file" && kind != "directory" {
		return errors.New("Unknown path type.")
	}
	if path == "" {
		return nil
	}
	if strings.ContainsRune(path, 0) {
		return errors.New("Remove invalid path characters.")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if kind == "directory" {
				return errors.New("Folder not found.")
			}
			return errors.New("File not found.")
		}
		if kind == "directory" {
			return errors.New("Cannot access this folder.")
		}
		return errors.New("Cannot access this file.")
	}
	if kind == "directory" && !info.IsDir() {
		return errors.New("Choose a folder.")
	}
	if kind == "file" && !info.Mode().IsRegular() {
		return errors.New("Choose a file.")
	}
	return nil
}
