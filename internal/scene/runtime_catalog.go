package scene

import (
	"github.com/Daniel-Ric/CraftBox/assets"
	"os"
)

func LoadRuntimeCatalog(paths ...string) *Catalog {
	catalog := NewCatalog()
	if source, err := assets.Vanilla(); err == nil {
		if bundled, err := LoadCatalogFS(source); err == nil {
			catalog = bundled
		}
	}
	for _, directory := range paths {
		if directory == "" {
			continue
		}
		override, err := LoadCatalog(directory)
		if err != nil {
			continue
		}
		if len(catalog.sources) == 0 {
			return override
		}
		if err = catalog.AddPack(os.DirFS(directory)); err == nil {
			catalog.Source = override.Source
			return catalog
		}
	}
	return catalog
}
