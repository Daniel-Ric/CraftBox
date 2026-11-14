package worlds

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func validateCaptureName(name string) error {
	if !utf8.ValidString(name) {
		return fmt.Errorf("The world name must be valid UTF-8")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("Enter a world name")
	}
	if utf8.RuneCountInString(name) > 80 {
		return fmt.Errorf("Use at most 80 characters for the world name")
	}
	if strings.ContainsAny(name, "<>:\"/\\|?*§") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("The world name contains unsupported characters")
	}
	if strings.Trim(name, ". ") == "" || strings.HasSuffix(name, ".") || strings.TrimSpace(name) != name {
		return fmt.Errorf("Use a name without leading/trailing spaces or trailing dots")
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
		return fmt.Errorf("This name is reserved; choose another world name")
	}
	return nil
}

func (w *worldsHandler) commandError(message, usage string) bool {
	text := "§c" + strings.TrimRight(message, ".") + ".§r"
	if usage != "" {
		text += "\n§7Usage: §e" + usage + "§r"
	}
	w.session.SendMessage(text)
	return false
}
func (w *worldsHandler) registerCommands(session *proxy.Session) {
	w.commandStarted = time.Now()
	w.captureSettingsMu.Lock()
	w.commandCompletionPrefix = "craftbox_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	w.commandEntityTypes = map[string]struct{}{}
	w.commandMetadataReady = false
	w.captureSettingsMu.Unlock()
	session.SetCommandCompletions(w.completeCommands, w.publishCommandSuggestions)
	add := func(name, description string, exec func([]string) bool) {
		session.AddCommand(exec, protocol.Command{Name: name, Description: description, Overloads: []protocol.CommandOverload{{Parameters: []protocol.CommandParameter{{Name: "arguments", Type: protocol.CommandArgValid | protocol.CommandArgTypeRawText, Optional: true}}}}})
	}
	add("cb", "CraftBox capture, map and help commands", w.commandRoot)
	add("cbhelp", "Show CraftBox command help", w.commandHelp)
	add("blueprint", "CraftBox Structure Studio selections and snapshots", w.commandStructure)
	add("setname", "Name the current CraftBox world", func(args []string) bool { return w.setWorldName(strings.Join(args, " ")) })
	add("generator", "Choose void, flat or normal terrain generation", w.commandGenerator)
	add("void", "Toggle the saved world's void generator", w.commandVoid)
	add("exclude-mob", "Exclude entity types from the saved world", w.commandExclude)
	add("include-mob", "Remove entity exclusion patterns", w.commandInclude)
	add("save-world", "Save the captured world and start a new capture", w.commandSave)
}
func (w *worldsHandler) commandRoot(args []string) bool {
	if len(args) == 0 {
		return w.commandHelp(nil)
	}
	rest := args[1:]
	switch strings.ToLower(args[0]) {
	case "menu":
		return w.commandMenu(rest)
	case "position":
		return w.commandPosition(rest)
	case "help":
		return w.commandHelp(rest)
	case "mark", "note", "marker":
		return w.commandPoint(strings.ToLower(args[0]), rest)
	case "camera":
		if len(rest) < 2 || rest[0] != "add" {
			return w.commandError("Use camera add and a name", "/cb camera add <name>")
		}
		return w.commandPoint("camera", rest[1:])
	case "region":
		return w.commandRegion(rest)
	case "structure", "blueprint":
		return w.commandStructure(rest)
	case "capture":
		return w.commandCaptureStatus(rest)
	case "name":
		return w.setWorldName(strings.Join(rest, " "))
	case "generator":
		return w.commandGenerator(rest)
	case "void":
		return w.commandVoid(rest)
	case "save":
		return w.commandSave(rest)
	case "exclude":
		return w.commandExclude(rest)
	case "include":
		return w.commandInclude(rest)
	case "exclusions":
		return w.commandExclusions(rest)
	case "map":
		return w.commandMap(rest)
	case "zoom":
		return w.commandZoom(rest)
	case "status":
		return w.commandStatus(rest)
	case "connection":
		if len(rest) != 0 {
			return w.commandError("This command takes no arguments", "/cb connection")
		}
		w.session.SendMessage(w.session.ConnectionDetailsText())
		return true
	case "version":
		if len(rest) > 0 {
			return w.commandError("This command takes no arguments", "/cb version")
		}
		version := utils.Version
		if version == "" {
			version = "development"
		}
		w.session.SendMessage("§7Version: §f" + version)
		return true
	default:
		return w.commandError("Unknown CraftBox command", "/cb help [1|2|3]")
	}
}
func (w *worldsHandler) commandHelp(args []string) bool {
	page := 1
	if len(args) > 1 {
		return w.commandError("Choose help page 1, 2 or 3", "/cb help [1|2|3]")
	}
	if len(args) == 1 {
		n, e := strconv.Atoi(args[0])
		if e != nil || n < 1 || n > 3 {
			return w.commandError("Choose help page 1, 2 or 3", "/cb help [1|2|3]")
		}
		page = n
	}
	lines := [][]string{
		{"/cb menu ?7? Open the ingame menu", "/cb help [1|2|3] §7» Show command help", "/setname <world name> §7» Rename this capture", "/generator <void|flat|normal> §7» Choose terrain generation", "/save-world §7» Save and start a fresh capture", "/cb status §7» Show capture and map status", "/cb connection \u00a77\u00bb Show current connection details", "/cb version §7» Show the CraftBox version"},
		{"/cb map [on|off|toggle] §7» Show or hide your local map", "/cb position [on|off|toggle] \u00a77\u00bb Show your current position", "/cb zoom [1|2|4|8|16] §7» Set or show map zoom", "/exclude-mob <id/pattern> ... §7» Exclude entity types", "/include-mob <pattern|all> ... §7» Remove exclusions", "/cb exclusions [page] §7» List entity exclusions", "/cb name|void|save|exclude|include §7» Command aliases"},
		{"/cb mark <name> §7» Save position and recording time", "/cb marker <icon> <name> §7» Place a named 2D map marker", "/cb note <text> §7» Save a note at this position", "/cb region pos1|pos2|clear §7» Select region corners", "/cb region create <name> §7» Create a region copy", "/cb camera add <name> §7» Save position and viewing direction", "/cb capture status §7» Show capture channels"},
	}
	text := fmt.Sprintf("§a§lCraftBox help §r§7(%d/3)§r", page)
	for _, line := range lines[page-1] {
		text += "\n§8» §e" + line + "§r"
	}
	text += "\n§7These commands affect your local capture, not the server.§r"
	w.session.SendMessage(text)
	return true
}
func parseCommandToggle(args []string, current bool) (bool, bool) {
	if len(args) == 0 {
		return !current, true
	}
	if len(args) != 1 {
		return current, false
	}
	switch strings.ToLower(args[0]) {
	case "on":
		return true, true
	case "off":
		return false, true
	case "toggle":
		return !current, true
	}
	return current, false
}
func (w *worldsHandler) commandVoid(args []string) bool {
	found, current := false, false
	w.currentWorld(func(world *worldstate.World) { found = true; current = world.VoidGen })
	if !found {
		return w.commandError("No active world capture", "/cb status")
	}
	next, ok := parseCommandToggle(args, current)
	if !ok {
		return w.commandError("Use on, off or toggle", "/void [on|off|toggle]")
	}
	return w.setVoidGen(next)
}
func (w *worldsHandler) commandMap(args []string) bool {
	if w.mapUI == nil {
		return w.commandError("The capture map is not ready", "/cb status")
	}

	next, ok := parseCommandToggle(args, w.mapUI.Enabled())
	if !ok {
		return w.commandError("Use on, off or toggle", "/cb map [on|off|toggle]")
	}
	w.mapUI.SetEnabled(next)
	state := "§7hidden"
	if next {
		state = "§ashown"
	}
	w.session.SendMessage("§7Local map: " + state + "§r")
	return true
}
func (w *worldsHandler) commandPosition(args []string) bool {
	if w.mapUI == nil {
		return w.commandError("The capture map is not ready", "/cb status")
	}
	next, ok := parseCommandToggle(args, w.mapUI.MarkerEnabled())
	if !ok {
		return w.commandError("Use on, off or toggle", "/cb position [on|off|toggle]")
	}
	w.mapUI.SetMarkerEnabled(next)
	state := "hidden"
	if next {
		state = "shown"
	}
	text := "§7Position marker: §f" + state + "§r"
	if next && !w.mapUI.Enabled() {
		text += "\n§7Show the map with §e/cb map on§r"
	}
	w.session.SendMessage(text)
	return true
}
func (w *worldsHandler) commandZoom(args []string) bool {
	if w.mapUI == nil {
		return w.commandError("The capture map is not ready", "/cb status")
	}
	if len(args) > 1 {
		return w.commandError("Choose zoom 1, 2, 4, 8 or 16", "/cb zoom [1|2|4|8|16]")
	}
	if len(args) == 1 {
		n, e := strconv.Atoi(args[0])
		if e != nil || !slices.Contains([]int{1, 2, 4, 8, 16}, n) {
			return w.commandError("Choose zoom 1, 2, 4, 8 or 16", "/cb zoom [1|2|4|8|16]")
		}
		w.mapUI.SetZoom(n)
	}
	w.session.SendMessage(fmt.Sprintf("§7Map zoom: §f%d§r", w.mapUI.Zoom()))
	return true
}
func (w *worldsHandler) commandStatus(args []string) bool {
	if len(args) > 0 {
		return w.commandError("This command takes no arguments", "/cb status")
	}
	text := "§7No active world capture.§r"
	w.currentWorld(func(world *worldstate.World) {
		state := "§7off"
		if world.VoidGen {
			state = "§aon"
		}
		text = fmt.Sprintf("§a§lCapture status§r\n§7World: §f%s§r\n§7Captured chunks: §f%d§r\n§7Void generator: %s§r", world.Name, len(world.StoredChunks), state)
	})
	if w.mapUI != nil {
		state := "hidden"
		if w.mapUI.Enabled() {
			state = "shown"
		}
		text += fmt.Sprintf("\n§7Local map: §f%s §8· §7Zoom: §f%d§r", state, w.mapUI.Zoom())
	}
	w.session.SendMessage(text)
	return true
}
func (w *worldsHandler) commandSave(args []string) bool {
	if len(args) > 0 {
		return w.commandError("This command takes no arguments", "/save-world")
	}
	found, chunks := false, 0
	w.currentWorld(func(world *worldstate.World) { found = true; chunks = len(world.StoredChunks) })
	if !found {
		return w.commandError("No active world capture", "/cb status")
	}
	if chunks == 0 {
		return w.commandError("No captured chunks to save yet", "/cb status")
	}
	w.session.SendMessage("§7Saving this capture and starting a fresh world tab…§r")
	w.SaveAndReset(false, nil)
	return true
}

var entityPattern = regexp.MustCompile(`^[a-z0-9_.-]+:[a-z0-9_./*?\[\]-]+$`)

func commandPatterns(args []string) ([]string, error) {
	if len(args) == 0 || len(args) > 32 {
		return nil, fmt.Errorf("Enter 1 to 32 entity IDs or patterns")
	}
	result := []string{}
	for _, arg := range args {
		value := strings.ToLower(arg)
		if !strings.Contains(value, ":") {
			value = "minecraft:" + value
		}
		if len(value) > 128 || !entityPattern.MatchString(value) {
			return nil, fmt.Errorf("Use entity IDs such as minecraft:zombie or minecraft:*")
		}
		if _, e := path.Match(value, "minecraft:zombie"); e != nil {
			return nil, fmt.Errorf("The entity pattern has invalid brackets")
		}
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result, nil
}
func (w *worldsHandler) commandExclude(args []string) bool {
	patterns, e := commandPatterns(args)
	if e != nil {
		return w.commandError(e.Error(), "/exclude-mob <id/pattern> ...")
	}
	w.captureSettingsMu.Lock()
	next := slices.Clone(w.settings.ExcludedMobs)
	for _, value := range patterns {
		if strings.ContainsAny(value, "*?[") {
			continue
		}
		if _, known := w.commandEntityTypes[value]; !known {
			ready := len(w.commandEntityTypes) > 0
			w.captureSettingsMu.Unlock()
			if !ready {
				return w.commandError("Entity types are not available yet; wait for the server catalog", "/exclude-mob <id/pattern> ...")
			}
			return w.commandError("Unknown entity type: "+value+". Choose an autocomplete suggestion", "/exclude-mob <id/pattern> ...")
		}
	}
	for _, value := range patterns {
		if !slices.Contains(next, value) {
			next = append(next, value)
		}
	}
	if len(next) > 128 {
		w.captureSettingsMu.Unlock()
		return w.commandError("At most 128 exclusion patterns are supported", "/cb exclusions")
	}
	w.settings.ExcludedMobs = next
	w.captureSettingsMu.Unlock()
	w.updateExclusionSuggestions()
	w.session.SendMessage("§7Excluded from saved entities: §e" + strings.Join(patterns, "§7, §e") + "§r")
	return true
}
func (w *worldsHandler) commandInclude(args []string) bool {
	if len(args) > 1 && slices.ContainsFunc(args, func(value string) bool { return strings.EqualFold(value, "all") }) {
		return w.commandError("Use all on its own", "/include-mob <pattern|all> ...")
	}
	if len(args) == 1 && strings.EqualFold(args[0], "all") {
		w.captureSettingsMu.Lock()
		w.settings.ExcludedMobs = nil
		w.captureSettingsMu.Unlock()
		w.updateExclusionSuggestions()
		w.session.SendMessage("§aEntity exclusions cleared.§r")
		return true
	}
	patterns, e := commandPatterns(args)
	if e != nil {
		return w.commandError(e.Error(), "/include-mob <pattern|all> ...")
	}
	w.captureSettingsMu.Lock()
	before := len(w.settings.ExcludedMobs)
	w.settings.ExcludedMobs = slices.DeleteFunc(slices.Clone(w.settings.ExcludedMobs), func(value string) bool { return slices.Contains(patterns, value) })
	removed := before - len(w.settings.ExcludedMobs)
	w.captureSettingsMu.Unlock()
	if removed == 0 {
		return w.commandError("No matching exclusion is configured", "/cb exclusions")
	}
	w.updateExclusionSuggestions()
	w.session.SendMessage(fmt.Sprintf("§aRemoved %d entity exclusion(s).§r", removed))
	return true
}
func (w *worldsHandler) commandExclusions(args []string) bool {
	patterns := w.excludedMobs()
	page := 1
	pages := max(1, (len(patterns)+7)/8)
	if len(args) > 1 {
		return w.commandError("Choose a valid exclusions page", "/cb exclusions [page]")
	}
	if len(args) == 1 {
		n, e := strconv.Atoi(args[0])
		if e != nil || n < 1 || n > pages {
			return w.commandError(fmt.Sprintf("Choose a page from 1 to %d", pages), "/cb exclusions [page]")
		}
		page = n
	}
	text := fmt.Sprintf("§a§lEntity exclusions §r§7(%d/%d)§r", page, pages)
	if len(patterns) == 0 {
		text += "\n§7No entity types are excluded.§r"
	} else {
		for _, value := range patterns[(page-1)*8 : min(page*8, len(patterns))] {
			text += "\n§8» §e" + value + "§r"
		}
	}
	w.session.SendMessage(text)
	return true
}

func (w *worldsHandler) excludedMobs() []string {
	w.captureSettingsMu.Lock()
	defer w.captureSettingsMu.Unlock()
	return slices.Clone(w.settings.ExcludedMobs)
}

func (w *worldsHandler) commandGenerator(args []string) bool {
	if len(args) != 1 {
		return w.commandError("Choose void, flat or normal", "/generator <void|flat|normal>")
	}
	generator, err := worldstate.ParseGenerator(args[0])
	if err != nil {
		return w.commandError(err.Error(), "/generator <void|flat|normal>")
	}
	changed := false
	w.currentWorld(func(state *worldstate.World) {
		state.Generator = generator
		state.VoidGen = generator == worldstate.GeneratorVoid
		w.settings.Generator = generator
		w.settings.VoidGen = state.VoidGen
		changed = true
	})
	if !changed {
		return w.commandError("No active world capture.", "/cb status")
	}
	w.session.SendMessage("Generator: " + string(generator))
	messages.SendEvent(&messages.EventSetValue{Name: "generator", Value: string(generator)})
	return true
}
