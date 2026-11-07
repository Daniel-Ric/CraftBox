package worlds

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/google/uuid"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var captureMarkerIcons = map[string]string{"pin": "📍", "home": "🏠", "sword": "⚔", "star": "★", "flag": "⛳", "shop": "🛒", "camera": "📷", "portal": "◈", "warning": "⚠", "chest": "▣"}

func validateCaptureLabel(value string) error {
	if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > 100 || strings.Contains(value, "§") || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("Enter a name of 1–100 characters without formatting or control characters")
	}
	return nil
}
func (w *worldsHandler) commandPoint(kind string, args []string) bool {
	usage := "/cb " + kind + " <name>"
	icon := "pin"
	if kind == "marker" {
		usage = "/cb marker <icon> <name>"
		if len(args) < 2 {
			return w.commandError("Choose an icon and enter a marker name", usage)
		}
		icon = strings.ToLower(args[0])
		args = args[1:]
		if _, ok := captureMarkerIcons[icon]; !ok {
			return w.commandError("Unknown marker icon; use autocomplete", usage)
		}
	}
	name := strings.Join(args, " ")
	if e := validateCaptureLabel(name); e != nil {
		return w.commandError(e.Error(), usage)
	}
	if w.session == nil || !w.serverState.haveStartGame {
		return w.commandError("Wait until the player has joined the world", usage)
	}
	player := w.session.PlayerSnapshot()
	for _, v := range append(player.Position[:], player.Yaw, player.Pitch) {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || math.Abs(float64(v)) > 30_000_000 {
			return w.commandError("Player position is not valid", usage)
		}
	}
	now := time.Now()
	elapsed := float64(0)
	if !w.commandStarted.IsZero() {
		elapsed = now.Sub(w.commandStarted).Seconds()
	}
	point := messages.CapturePoint{ID: uuid.NewString(), Kind: kind, Name: name, Icon: captureMarkerIcons[icon], Position: [3]float32(player.Position), Yaw: player.Yaw, Pitch: player.Pitch, Created: now, Elapsed: elapsed}
	if kind == "camera" {
		point.Icon = captureMarkerIcons["camera"]
	}
	found := false
	var err error
	var points []messages.CapturePoint
	var id string
	w.currentWorld(func(ws *worldstate.World) {
		found = true
		id = ws.ID
		points, err = ws.AddCapturePoint(point)
	})
	if !found {
		return w.commandError("No active world capture", usage)
	}
	if err != nil {
		return w.commandError(err.Error(), usage)
	}
	messages.SendEvent(&messages.EventCapturePoints{WorldID: id, Points: points})
	w.session.SendMessage(fmt.Sprintf("§a%s saved: §f%s§r\n§7X §f%.3f §7Y §f%.3f §7Z §f%.3f§r", strings.ToUpper(kind[:1])+kind[1:], name, player.Position[0], player.Position[1], player.Position[2]))
	return true
}

func (w *worldsHandler) commandRegion(args []string) bool {
	usage := "/cb region pos1|pos2|clear|create <name>"
	if len(args) == 0 {
		return w.commandError("Choose pos1, pos2, clear or create", usage)
	}
	if args[0] == "create" {
		name := strings.Join(args[1:], " ")
		if e := validateCaptureName(name); e != nil {
			return w.commandError(e.Error(), usage)
		}
		w.worldStateMu.Lock()
		ws := w.worldState
		selection := w.commandRegionSelection
		valid := ws != nil && selection.worldID == ws.ID && selection.first != nil && selection.second != nil
		id := ""
		if ws != nil {
			id = ws.ID
		}
		w.worldStateMu.Unlock()
		if !valid {
			return w.commandError("Set pos1 and pos2 in this world first", usage)
		}
		a, b := *selection.first, *selection.second
		region, e := messages.RequestWorldRegion(messages.CreateWorldRegion{WorldID: id, Name: name, Move: false, Bounds: messages.ChunkBounds{MinX: min(a[0], b[0]), MinZ: min(a[1], b[1]), MaxX: max(a[0], b[0]), MaxZ: max(a[1], b[1])}})
		if e != nil {
			return w.commandError(e.Error(), usage)
		}
		w.session.SendMessage("§aRegion created: §f" + region.Name + "§r §7(copy; complete chunk columns)§r")
		return true
	}
	if len(args) != 1 {
		return w.commandError("Unexpected arguments", usage)
	}
	if args[0] == "clear" {
		w.worldStateMu.Lock()
		w.commandRegionSelection = commandRegionSelection{}
		w.worldStateMu.Unlock()
		w.session.SendMessage("§7Region selection cleared.§r")
		return true
	}
	if args[0] != "pos1" && args[0] != "pos2" {
		return w.commandError("Unknown region action", usage)
	}
	if !w.serverState.haveStartGame {
		return w.commandError("Wait until the player has joined the world", usage)
	}
	pos := w.session.PlayerSnapshot().Position
	for _, v := range pos {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || math.Abs(float64(v)) > 30_000_000 {
			return w.commandError("Player position is not valid", usage)
		}
	}
	p := [2]int32{int32(math.Floor(float64(pos[0]) / 16)), int32(math.Floor(float64(pos[2]) / 16))}
	found := false
	w.currentWorld(func(ws *worldstate.World) {
		found = true
		if w.commandRegionSelection.worldID != ws.ID {
			w.commandRegionSelection = commandRegionSelection{worldID: ws.ID}
		}
		if args[0] == "pos1" {
			w.commandRegionSelection.first = &p
		} else {
			w.commandRegionSelection.second = &p
		}
	})
	if !found {
		return w.commandError("No active world capture", usage)
	}
	w.session.SendMessage(fmt.Sprintf("§a%s set §7at chunk §f%d, %d§r", args[0], p[0], p[1]))
	return true
}

type commandRegionSelection struct {
	worldID       string
	first, second *[2]int32
}

func (w *worldsHandler) commandCaptureStatus(args []string) bool {
	if len(args) != 1 || args[0] != "status" {
		return w.commandError("Use capture status", "/cb capture status")
	}
	w.commandStatus(nil)
	w.captureSettingsMu.Lock()
	saveEntities, saveCustomEntities, saveInventories, saveImage := w.settings.SaveEntities, w.settings.SaveCustomEntities, w.settings.SaveInventories, w.settings.SaveImage
	w.captureSettingsMu.Unlock()
	on := func(v bool) string {
		if v {
			return "§aon§r"
		}
		return "§7off§r"
	}
	w.session.SendMessage(fmt.Sprintf("§a§lCapture channels§r\n§7Terrain: §aon§r\n§7Entities: %s §8· §7Custom entities: %s\n§7Inventories: %s §8· §7Map PNG: %s\n%s", on(saveEntities), on(saveCustomEntities), on(saveInventories), on(saveImage), w.session.CaptureStatusText()))
	return true
}
