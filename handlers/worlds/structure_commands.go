package worlds

import (
	"context"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/go-gl/mathgl/mgl32"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type structureCommandSelection struct {
	worldID       string
	first, second *structure.Vec3
	cancel        context.CancelFunc
}

// Bedrock movement packets carry the player position 1.62 blocks above the feet.
func structureFeetPosition(position mgl32.Vec3) (structure.Vec3, error) {
	var result structure.Vec3
	for _, v := range position {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || math.Abs(float64(v)) > 30000000 {
			return result, fmt.Errorf("Player position is invalid")
		}
	}
	position[1] -= 1.62
	for i, v := range position {
		result[i] = int32(math.Floor(float64(v)))
	}
	return result, nil
}

func (w *worldsHandler) commandStructure(args []string) bool {
	usage := "/cb structure pos1|pos2 [x y z] / save <name> / status / clear / cancel / open / list"
	if len(args) == 0 {
		w.session.SendMessage("§b§lStructure Studio§r\n§7Select precise block corners with §f/cb structure pos1§7 and §f/cb structure pos2§7. Coordinates are optional.\n§7Then §f/cb structure save <name>§7 creates an independent blueprint.\n§7Use §f/cb structure open§7 to open the Studio; §f/cb structure list§7 lists saved blueprints.\n§7Nothing is placed or changed on the server.§r")
		return true
	}
	action := strings.ToLower(args[0])
	switch action {
	case "open":
		messages.SendEvent(&messages.EventStructureCreated{Open: true})
		w.session.SendMessage("§aStructure Studio opened in CraftBox.§r")
		return true
	case "list":
		items, err := os.ReadDir(utils.PathData("structures", "sessions"))
		if err != nil && !os.IsNotExist(err) {
			return w.commandError(err.Error(), usage)
		}
		lines := []string{"§b§lSaved blueprints§r"}
		count := 0
		for _, item := range items {
			if !item.IsDir() {
				continue
			}
			s, e := structure.ReadProjectBrief(filepath.Join(utils.PathData("structures", "sessions"), item.Name(), "recovery.craftbox"))
			if e == nil {
				lines = append(lines, fmt.Sprintf("§f%s §7(%d × %d × %d)§r", s.Name, s.Size[0], s.Size[1], s.Size[2]))
				count++
				if count >= 20 {
					break
				}
			}
		}
		if count == 0 {
			lines = append(lines, "§7No blueprints saved yet.§r")
		}
		w.session.SendMessage(strings.Join(lines, "\n"))
		return true
	case "clear", "cancel":
		w.worldStateMu.Lock()
		if action == "cancel" {
			if w.commandStructureSelection.cancel != nil {
				w.commandStructureSelection.cancel()
			}
		} else {
			cancel := w.commandStructureSelection.cancel
			w.commandStructureSelection = structureCommandSelection{cancel: cancel}
		}
		w.worldStateMu.Unlock()
		w.session.SendMessage("§7Structure " + action + " requested.§r")
		return true
	case "status":
		w.worldStateMu.Lock()
		s := w.commandStructureSelection
		w.worldStateMu.Unlock()
		text := "§7Set pos1 and pos2 in the current world.§r"
		if s.first != nil && s.second != nil {
			b := (structure.Bounds{Min: *s.first, Max: *s.second}).Normalized()
			text = fmt.Sprintf("§bStructure selection§r\n§7Corner 1: §f%v\n§7Corner 2: §f%v\n§7Size: §f%v§r", b.Min, b.Max, b.Size())
		}
		w.session.SendMessage(text)
		return true
	case "pos1", "pos2":
		if len(args) != 1 && len(args) != 4 {
			return w.commandError("Use no coordinates or exactly x y z", usage)
		}
		var p structure.Vec3
		if len(args) == 4 {
			for i := range p {
				n, e := strconv.ParseInt(args[i+1], 10, 32)
				if e != nil || math.Abs(float64(n)) > 30000000 {
					return w.commandError("Invalid block coordinate", usage)
				}
				p[i] = int32(n)
			}
		} else {
			var err error
			p, err = structureFeetPosition(w.session.PlayerSnapshot().Position)
			if err != nil {
				return w.commandError(err.Error(), usage)
			}
		}
		w.worldStateMu.Lock()
		ws := w.worldState
		if ws == nil {
			w.worldStateMu.Unlock()
			return w.commandError("No active capture", usage)
		}
		if w.commandStructureSelection.worldID != ws.ID {
			cancel := w.commandStructureSelection.cancel
			w.commandStructureSelection = structureCommandSelection{worldID: ws.ID, cancel: cancel}
		}
		if action == "pos1" {
			w.commandStructureSelection.first = &p
		} else {
			w.commandStructureSelection.second = &p
		}
		w.worldStateMu.Unlock()
		w.session.SendMessage(fmt.Sprintf("§aStructure %s set at §f%d, %d, %d§r", action, p[0], p[1], p[2]))
		return true
	case "save":
		name := strings.Join(args[1:], " ")
		if err := structure.ValidateName(name); err != nil {
			return w.commandError(err.Error(), usage)
		}
		w.worldStateMu.Lock()
		ws, s := w.worldState, w.commandStructureSelection
		if s.cancel != nil {
			w.worldStateMu.Unlock()
			return w.commandError("A blueprint is already being saved; use cancel or wait", usage)
		}
		if ws == nil || s.worldID != ws.ID || s.first == nil || s.second == nil {
			w.worldStateMu.Unlock()
			return w.commandError("Set both corners in this world first", usage)
		}
		bounds := (structure.Bounds{Min: *s.first, Max: *s.second}).Normalized()
		if _, err := structure.NewImporter(name, bounds); err != nil {
			w.worldStateMu.Unlock()
			return w.commandError(err.Error(), usage)
		}
		ctx, cancel := context.WithCancel(w.ctx)
		w.commandStructureSelection.cancel = cancel
		w.wg.Add(1)
		w.worldStateMu.Unlock()
		w.session.SendMessage("\u00a77Saving independent blueprint snapshot...\u00a7r")
		go func() {
			defer w.wg.Done()
			defer cancel()
			defer func() { w.worldStateMu.Lock(); w.commandStructureSelection.cancel = nil; w.worldStateMu.Unlock() }()
			saved, e := ws.SaveStructure(ctx, name, bounds)
			if e != nil {
				w.session.SendMessage("§cBlueprint could not be saved: " + e.Error() + "§r")
				return
			}
			messages.SendEvent(&messages.EventStructureCreated{ID: saved.ID, Name: name})
			w.session.SendMessage("§aBlueprint saved: §f" + name + "§r\n§7Open Structure Studio in CraftBox to edit or export it.§r")
		}()
		return true
	default:
		return w.commandError("Unknown structure action", usage)
	}
}
