package worlds

import (
	"maps"
	"regexp"
	"strings"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
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

func (w *worldsHandler) onMapSeen(id int64) {
	if id == ViewMapID || w.session == nil || w.session.Server == nil {
		return
	}
	w.serverState.mapsMu.Lock()
	if w.serverState.requestedMaps == nil {
		w.serverState.requestedMaps = map[int64]struct{}{}
	}
	known := w.serverState.knownMaps[id]
	have := known.PixelsComplete()
	_, asked := w.serverState.requestedMaps[id]
	if !have && !asked {
		w.serverState.requestedMaps[id] = struct{}{}
	}
	w.serverState.mapsMu.Unlock()
	if have || asked || w.session == nil || w.session.Server == nil {
		return
	}
	if err := w.session.Server.WritePacket(&packet.MapInfoRequest{MapID: id}); err != nil {
		w.serverState.mapsMu.Lock()
		delete(w.serverState.requestedMaps, id)
		w.serverState.mapsMu.Unlock()
		logrus.Debugf("MapInfoRequest %d: %s", id, err)
	}
}

func (w *worldsHandler) storeKnownMap(pk *packet.ClientBoundMapItemData) {
	w.serverState.mapsMu.Lock()
	defer w.serverState.mapsMu.Unlock()
	if w.serverState.knownMaps == nil {
		w.serverState.knownMaps = map[int64]*worldstate.Map{}
	}
	w.serverState.knownMaps[pk.MapID] = worldstate.ApplyMapPacket(w.serverState.knownMaps[pk.MapID], pk)
}

func (w *worldsHandler) knownMapsSnapshot() map[int64]*worldstate.Map {
	w.serverState.mapsMu.Lock()
	defer w.serverState.mapsMu.Unlock()
	return maps.Clone(w.serverState.knownMaps)
}
