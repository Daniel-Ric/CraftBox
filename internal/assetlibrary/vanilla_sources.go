package assetlibrary

import (
	"io"
	"io/fs"
	"strings"
	"sync"

	"github.com/Daniel-Ric/CraftBox/assets"
)

func vanillaTexture(ref string) string {
	source, err := assets.Vanilla()
	if err != nil {
		return ""
	}
	for _, candidate := range []string{ref, ref + ".png", ref + ".tga", ref + ".jpg", ref + ".jpeg"} {
		if info, err := fs.Stat(source, candidate); err == nil && info.Mode().IsRegular() && fileKind(candidate) == "texture" {
			return candidate
		}
	}
	return ""
}

func (x objectIndex) readTexture(root, name string) (Preview, error) {
	if x.names[name] {
		return readAssetFile(root, name)
	}
	source, err := assets.Vanilla()
	if err != nil {
		return Preview{}, err
	}
	file, err := (previewFS{source}).Open(name)
	if err != nil {
		return Preview{}, err
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		return Preview{}, err
	}
	return assetFilePreview(name, "texture", raw)
}

var vanillaRenderControllers = sync.OnceValue(func() map[string]map[string]any {
	controllers := map[string]map[string]any{}
	source, err := assets.Vanilla()
	if err != nil {
		return controllers
	}
	fs.WalkDir(source, "render_controllers", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".json") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 4<<20 {
			return nil
		}
		raw, err := fs.ReadFile(source, name)
		if err != nil {
			return nil
		}
		var document map[string]any
		if normalizedJSON(raw, &document) != nil {
			return nil
		}
		for name, definition := range objectMap(document["render_controllers"]) {
			if controller, ok := definition.(map[string]any); ok {
				controllers[strings.ToLower(name)] = controller
			}
		}
		return nil
	})
	return controllers
})

func previewRenderControllers() map[string]map[string]any {
	controllers := map[string]map[string]any{}
	for name, controller := range vanillaRenderControllers() {
		controllers[name] = controller
	}
	return controllers
}
