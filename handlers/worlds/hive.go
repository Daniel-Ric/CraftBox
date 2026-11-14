package worlds

import (
	"maps"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

var colorCodes = regexp.MustCompile("§.")

func isHiveServer(address, serverName string) bool {
	address = strings.ToLower(address)
	if strings.Contains(address, "hivebedrock.network") || strings.Contains(address, "playhive.com") {
		return true
	}
	name := strings.ToLower(colorCodes.ReplaceAllString(serverName, ""))
	return strings.Contains(name, "the hive") || strings.Contains(name, "hivebedrock")
}

func hasHivePacks(packs []*resource.Pack) bool {
	for _, p := range packs {
		if p == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(colorCodes.ReplaceAllString(p.Name(), "")))
		if strings.HasPrefix(name, "hive") || strings.HasPrefix(name, "the hive") {
			return true
		}
	}
	return false
}

func isNPCPlayer(xuids map[uuid.UUID]string, id uuid.UUID) bool {
	x, ok := xuids[id]
	return ok && (x == "" || x == "0")
}

func (w *worldsHandler) keepPlayer() func(uuid.UUID) bool {
	if w.settings.Players {
		return func(uuid.UUID) bool { return true }
	}
	if !w.serverState.isHive || !w.settings.SaveEntities {
		return nil
	}
	xuids := maps.Clone(w.serverState.playerXUIDs)
	return func(id uuid.UUID) bool { return isNPCPlayer(xuids, id) }
}
