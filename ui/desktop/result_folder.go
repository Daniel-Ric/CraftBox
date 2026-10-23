package desktop

import (
	"os"
	"path/filepath"
)

func resultFolder(value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		absolute = filepath.Dir(absolute)
	}
	return absolute, nil
}
