package worlds

import (
	"maps"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

func isHiveGameVersion(version string) bool { return strings.Contains(version, "Hive Games proxy") }

func isNPCPlayer(xuids map[uuid.UUID]string, id uuid.UUID) bool {
	x, ok := xuids[id]
	return ok && (x == "" || x == "0")
}

func (w *worldsHandler) keepPlayer() func(uuid.UUID) bool {
	overrides := maps.Clone(w.serverState.playerSaveOverrides)
	xuids := maps.Clone(w.serverState.playerXUIDs)
	players, npcs := w.settings.Players, w.serverState.isHive && w.settings.SaveEntities
	return func(id uuid.UUID) bool {
		if save, ok := overrides[id]; ok {
			return save
		}
		return players || npcs && isNPCPlayer(xuids, id)
	}
}

func (w *worldsHandler) onMapSeen(id int64) {
	w.requestMap(id, false)
}

func (w *worldsHandler) requestMap(id int64, retry bool) bool {
	if id == ViewMapID || w.session == nil || w.session.Server == nil {
		return false
	}
	now := time.Now()
	w.serverState.mapsMu.Lock()
	if w.serverState.requestedMaps == nil {
		w.serverState.requestedMaps = map[int64]time.Time{}
	}
	last := w.serverState.requestedMaps[id]
	if w.serverState.knownMaps[id].PixelsComplete() || (!last.IsZero() && (!retry || now.Sub(last) < 3*time.Second)) {
		w.serverState.mapsMu.Unlock()
		return false
	}
	w.serverState.requestedMaps[id] = now
	w.serverState.mapsMu.Unlock()
	if err := w.session.Server.WritePacket(&packet.MapInfoRequest{MapID: id}); err != nil {
		w.serverState.mapsMu.Lock()
		delete(w.serverState.requestedMaps, id)
		w.serverState.mapsMu.Unlock()
		logrus.Debugf("MapInfoRequest %d: %s", id, err)
		return false
	}
	return true
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
