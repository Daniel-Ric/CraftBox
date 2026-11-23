//go:build gui && !android

package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/localworlds"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
)

type MinecraftFolders struct {
	Channel       string `json:"channel,omitempty"`
	ProfileKind   string `json:"profileKind"`
	ProfileName   string `json:"profileName,omitempty"`
	ProfileRef    string `json:"profileRef,omitempty"`
	LookupKey     string `json:"lookupKey,omitempty"`
	Worlds        string `json:"worlds,omitempty"`
	ResourcePacks string `json:"resourcePacks,omitempty"`
	BehaviorPacks string `json:"behaviorPacks,omitempty"`
}

func (a *App) OpenCraftBoxFolder(folder string) error {
	var path string
	switch folder {
	case "worlds", "captures":
		path = utils.PathData(folder)
	case "asset-library":
		path = a.assets().Root
	case "packs":
		path = filepath.Join(a.assets().Root, "originals")
	case "packets":
		path = utils.PathData("packet-inspector")
	case "structures":
		path = structureRoot()
	default:
		return fmt.Errorf("unknown CraftBox folder: %s", folder)
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	return a.OpenResultFolder(path)
}

func (a *App) GetMinecraftFolders() []MinecraftFolders {
	var installations []assetlibrary.Installation
	if runtime.GOOS == "windows" {
		installations = assetlibrary.DiscoverMinecraft(os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), nil)
	}
	a.mu.Lock()
	selected := append([]localworlds.Root(nil), a.localWorldRoots...)
	a.mu.Unlock()
	return minecraftFolderLocations(installations, selected)
}

func (a *App) ResolveMinecraftGamertags() (map[string]string, error) {
	account := auth.Auth.Account()
	if account == nil {
		return nil, fmt.Errorf("sign in to look up Xbox gamertags")
	}
	keysByXUID := map[string][]string{}
	if runtime.GOOS == "windows" {
		for _, installation := range assetlibrary.DiscoverMinecraft(os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), nil) {
			_, xuid := installation.SavedProfile()
			if xuid != "" {
				keysByXUID[xuid] = append(keysByXUID[xuid], installation.ID)
			}
		}
	}
	names := map[string]string{}
	if len(keysByXUID) == 0 {
		return names, nil
	}
	ctx, cancel := context.WithTimeout(a.localWorldContext(), 12*time.Second)
	defer cancel()
	client, err := account.XBLClient(ctx)
	if err != nil {
		return nil, err
	}
	save := func(xuid, gamertag string) {
		if gamertag != "" {
			for _, key := range keysByXUID[xuid] {
				names[key] = gamertag
			}
		}
	}
	current := accountProfileFromClaims(client.UserInfo())
	if current.Gamertag != "" {
		save(current.XUID, current.Gamertag)
		delete(keysByXUID, current.XUID)
	}
	ids := make([]string, 0, len(keysByXUID))
	for id := range keysByXUID {
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		users, err := client.Social().UsersByXUIDs(ctx, ids)
		if err != nil && len(names) == 0 {
			return nil, err
		}
		for _, user := range users {
			name := user.UniqueModernGamerTag
			if name == "" && user.ModernGamerTag != "" {
				name = user.ModernGamerTag
				if user.ModernGamerTagSuffix != "" {
					name += "#" + user.ModernGamerTagSuffix
				}
			}
			if name == "" {
				name = user.GamerTag
			}
			save(user.XUID, name)
		}
	}
	if auth.Auth.Account() != account {
		return nil, fmt.Errorf("account changed")
	}
	return names, nil
}

func minecraftFolderLocations(installations []assetlibrary.Installation, selected []localworlds.Root) []MinecraftFolders {
	locations := []MinecraftFolders{}
	seen := map[string]bool{}
	existing := func(path string) string {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return ""
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return ""
		}
		key := resolved
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			return ""
		}
		seen[key] = true
		return resolved
	}
	for _, installation := range installations {
		if !installation.StorageAvailable || installation.Storage == "" {
			continue
		}
		folders := MinecraftFolders{
			Channel:       installation.Channel,
			Worlds:        existing(filepath.Join(installation.Storage, "minecraftWorlds")),
			ResourcePacks: existing(filepath.Join(installation.Storage, "resource_packs")),
			BehaviorPacks: existing(filepath.Join(installation.Storage, "behavior_packs")),
		}
		if folders.Worlds != "" || folders.ResourcePacks != "" || folders.BehaviorPacks != "" {
			switch {
			case installation.Layout == "UWP":
				folders.ProfileKind = "legacy"
			case strings.EqualFold(installation.ProfileFolder, "Shared"):
				folders.ProfileKind = "shared"
			case installation.ProfileFolder == "":
				folders.ProfileKind = "default"
			default:
				folders.ProfileKind = "personal"
				name, xuid := installation.SavedProfile()
				folders.ProfileName = name
				folders.ProfileRef = installation.ProfileReference()
				if xuid != "" {
					folders.LookupKey = installation.ID
				}
			}
			locations = append(locations, folders)
		}
	}
	for _, root := range selected {
		worlds := root.Folder
		if worlds == "" && root.Storage != "" {
			worlds = filepath.Join(root.Storage, "minecraftWorlds")
		}
		if worlds != "" {
			if path := existing(worlds); path != "" {
				locations = append(locations, MinecraftFolders{
					ProfileKind: "manual",
					ProfileName: filepath.Base(path),
					Worlds:      path,
				})
			}
		}
	}
	return locations
}
