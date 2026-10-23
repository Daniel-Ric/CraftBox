package behaviourpack

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/utils"
	"io/fs"
	"strings"
)

func (bp *Pack) IncludeClientEntities(pack fs.FS) error {
	return fs.WalkDir(pack, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasPrefix(name, "entity/") || !strings.HasSuffix(strings.ToLower(name), ".json") {
			return nil
		}
		raw, err := fs.ReadFile(pack, name)
		if err != nil {
			return err
		}
		var file struct {
			Client struct {
				Description struct {
					Identifier string `json:"identifier"`
				} `json:"description"`
			} `json:"minecraft:client_entity"`
		}
		if err = utils.ParseJson(raw, &file); err != nil {
			return fmt.Errorf("client entity %s: %w", name, err)
		}
		bp.SetEntityProperties(file.Client.Description.Identifier, nil)
		return nil
	})
}
