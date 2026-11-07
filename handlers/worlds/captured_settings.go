package worlds

import (
	"sort"
	"strings"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (w *worldsHandler) rememberGameRules(rules []protocol.GameRule) {
	w.captureSettingsMu.Lock()
	defer w.captureSettingsMu.Unlock()
	if w.serverState.captureRules == nil {
		w.serverState.captureRules = map[string]protocol.GameRule{}
	}
	for _, rule := range rules {
		w.serverState.captureRules[strings.ToLower(rule.Name)] = rule
	}
}
func (w *worldsHandler) capturedGameData() minecraft.GameData {
	data := w.session.Server.GameData()
	w.captureSettingsMu.Lock()
	defer w.captureSettingsMu.Unlock()
	if w.serverState.captureMode != nil {
		data.WorldGameMode = *w.serverState.captureMode
	}
	if w.serverState.captureDifficulty != nil {
		data.Difficulty = *w.serverState.captureDifficulty
	}
	if w.serverState.captureRules != nil {
		names := make([]string, 0, len(w.serverState.captureRules))
		for name := range w.serverState.captureRules {
			names = append(names, name)
		}
		sort.Strings(names)
		data.GameRules = make([]protocol.GameRule, 0, len(names))
		for _, name := range names {
			data.GameRules = append(data.GameRules, w.serverState.captureRules[name])
		}
	}
	return data
}
func (w *worldsHandler) rememberWeatherEvent(event *packet.LevelEvent) {
	level := float32(event.EventData) / 65535
	if level <= 0 {
		level = 1
	}
	level = min(1, level)
	w.captureSettingsMu.Lock()
	switch event.EventType {
	case packet.LevelEventStartRaining:
		w.serverState.rainLevel = level
	case packet.LevelEventStopRaining:
		w.serverState.rainLevel = 0
	case packet.LevelEventStartThunderstorm:
		w.serverState.lightningLevel = level
	case packet.LevelEventStopThunderstorm:
		w.serverState.lightningLevel = 0
	default:
		w.captureSettingsMu.Unlock()
		return
	}
	rain, lightning := w.serverState.rainLevel, w.serverState.lightningLevel
	w.captureSettingsMu.Unlock()
	w.currentWorld(func(state *worldstate.World) { state.SetWeather(rain, lightning) })
}
