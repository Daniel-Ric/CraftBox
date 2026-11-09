package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/minecraftservers"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
)

type MinecraftServerSources struct {
	Profiles []minecraftservers.Profile `json:"profiles"`
	Sources  []Source                   `json:"sources"`
}

func (a *App) GetMinecraftServerSources() MinecraftServerSources {
	installations := assetlibrary.DiscoverMinecraft(os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), nil)
	result := MinecraftServerSources{Profiles: minecraftservers.ReadProfiles(installations), Sources: []Source{}}
	targets := map[string]*connectinfo.ConnectInfo{}
	for _, profile := range result.Profiles {
		for _, server := range profile.Servers {
			hash := sha256.Sum256([]byte(profile.ID + "\x00" + strings.ToLower(server.Address)))
			id := "source:minecraft:" + hex.EncodeToString(hash[:16])
			targets[id] = &connectinfo.ConnectInfo{Value: server.Address}
			result.Sources = append(result.Sources, Source{ID: id, Name: server.Name, Address: server.Address, Description: server.Address + " · " + profile.Name, Available: true, Route: "NetherNet · Minecraft saved server", ProfileID: profile.ID, ProfileName: profile.Name})
		}
	}
	a.mu.Lock()
	if a.sources == nil {
		a.sources = map[string]*connectinfo.ConnectInfo{}
	}
	for id := range a.sources {
		if strings.HasPrefix(id, "source:minecraft:") {
			delete(a.sources, id)
		}
	}
	for id, target := range targets {
		a.sources[id] = target
	}
	a.mu.Unlock()
	return result
}
