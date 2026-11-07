package desktop

import (
	"os"
	"path/filepath"
)

func writeExportFile(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".craftbox-output-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
