package osabs

import (
	"os"
	"path/filepath"
)

func implInit() error {
	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dataPath := filepath.Join(config, "CraftBox")
	cachePath := filepath.Join(cache, "CraftBox")
	for _, dir := range []string{dataPath, cachePath} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	dataDir, cacheDir = dataPath, cachePath
	return nil
}
