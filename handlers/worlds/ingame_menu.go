package worlds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/internal/ingamefeedback"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
)

type menuAction struct {
	text string
	run  func()
}

func (w *worldsHandler) commandMenu(args []string) bool {
	if len(args) != 0 {
		return w.commandError("This command takes no arguments", "/cb menu")
	}
	return w.showMenu("CraftBox", w.menuDashboardText(), []menuAction{
		{"Quick actions\nPosition, map and recording shortcuts", func() { w.quickMenu() }},
		{"Capture & world\nStatus, world options and saving", func() { w.captureMenu() }},
		{"Map & position\nMap visibility, live position and zoom", func() { w.mapMenu() }},
		{"Points & notes\nMarkers, recording marks, notes and cameras", func() { w.pointsMenu() }},
		{"Blueprints\nSelect corners and save structures", func() { w.blueprintMenu() }},
		{"Regions\nSelect and create a separate map", func() { w.regionMenu() }},
		{"Entity exclusions\nManage which entities are saved", func() { w.exclusionsMenu(0) }},
		{"Connection & session\nConnection details and player information", func() { w.connectionMenu() }},
		{"Interface settings\nHotbar, model picking and connection controls", func() { w.feedbackMenu() }},
		{"Help", func() { w.helpMenu(0) }},
	})
}

func (w *worldsHandler) feedbackMenu() {
	settings := proxy.IngameInterfaceSettings()
	w.showForm(map[string]any{"type": "custom_form", "title": "CraftBox interface", "content": []any{
		map[string]any{"type": "label", "text": "Customize CraftBox notifications and menus. Server messages and command replies are unchanged. Settings are shared with the desktop app."},
		map[string]any{"type": "toggle", "text": "Show CraftBox hotbar messages", "default": settings.Enabled},
		map[string]any{"type": "toggle", "text": "Confirm disconnecting from the in-game menu", "default": settings.ConfirmDisconnect},
		map[string]any{"type": "toggle", "text": "Show F8 model-picking hints", "default": settings.ModelPickFeedback},
		map[string]any{"type": "toggle", "text": "Bring CraftBox to the front after F8", "default": settings.ModelPickFocus},
		map[string]any{"type": "toggle", "text": "Use colored menu text", "default": settings.MenuColors},
	}}, func(data json.RawMessage) {
		var desired ingamefeedback.Settings
		if !menuFields(data, nil, &desired.Enabled, &desired.ConfirmDisconnect, &desired.ModelPickFeedback, &desired.ModelPickFocus, &desired.MenuColors) {
			return
		}
		if err := proxy.SetIngameInterfaceSettings(desired); err != nil {
			w.session.SendMessage("Could not save interface settings: " + err.Error())
			return
		}
		if !desired.Enabled {
			w.session.ClearHotbarFeedback()
		}
		w.commandMenu(nil)
	})
}

func (w *worldsHandler) menuBack() menuAction {
	return menuAction{"Back to CraftBox", func() { w.commandMenu(nil) }}
}

func (w *worldsHandler) showForm(form any, submit func(json.RawMessage)) bool {
	if w.session == nil {
		return false
	}
	worldID := ""
	w.currentWorld(func(ws *worldstate.World) { worldID = ws.ID })
	if err := w.session.ShowLocalForm(presentMenuForm(form, proxy.IngameInterfaceSettings()), func(data json.RawMessage) {
		w.submitMenuForm(worldID, submit, data)
	}); err != nil {
		return w.commandError("Could not open menu: "+err.Error(), "/cb menu")
	}
	return true
}

func (w *worldsHandler) submitMenuForm(worldID string, submit func(json.RawMessage), data json.RawMessage) {
	w.menuWorldMu.Lock()
	defer w.menuWorldMu.Unlock()
	currentID := ""
	w.currentWorld(func(ws *worldstate.World) { currentID = ws.ID })
	if worldID != currentID {
		w.commandError("The active capture changed; reopen the menu before applying changes", "/cb menu")
		return
	}
	if submit != nil {
		submit(data)
	}
}

func (w *worldsHandler) showMenu(title, content string, actions []menuAction) bool {
	buttons := make([]map[string]string, len(actions))
	settings := proxy.IngameInterfaceSettings()
	for i, action := range actions {
		buttons[i] = map[string]string{"text": menuActionText(action.text, i, settings)}
	}
	return w.showForm(map[string]any{"type": "form", "title": "§8" + title + "§r", "content": content, "buttons": buttons}, func(data json.RawMessage) {
		var selection int
		if json.Unmarshal(data, &selection) != nil || selection < 0 || selection >= len(actions) {
			return
		}
		actions[selection].run()
	})
}

func (w *worldsHandler) inputMenu(title, label string, save func(string) bool) {
	w.inputMenuReturn(title, label, "", "", save, func() { w.commandMenu(nil) })
}

func (w *worldsHandler) inputMenuReturn(title, label, value, problem string, save func(string) bool, back func()) {
	content := []any{
		map[string]any{"type": "label", "text": "§8" + problem + "Enter your text below. Closing this form cancels the action."},
		map[string]any{"type": "input", "text": "§8" + label, "placeholder": "Enter text", "default": value},
		map[string]any{"type": "toggle", "text": "§8Return to the previous menu after saving", "default": true},
	}
	w.showForm(map[string]any{
		"type": "custom_form", "title": "§8" + title + "§r",
		"content": content,
	}, func(data json.RawMessage) {
		var value string
		var returnBack bool
		if !menuFields(data, nil, &value, &returnBack) {
			return
		}
		if !save(strings.TrimSpace(value)) {
			w.inputMenuReturn(title, label, value, "The action could not be completed. See the chat message for details.\n\n", save, back)
		} else if returnBack {
			back()
		}
	})
}

// Decode all fields before applying any action. Labels submit null; actual
// controls must have a non-null value of the expected type.
func menuFields(data json.RawMessage, fields ...any) bool {
	var values []json.RawMessage
	if json.Unmarshal(data, &values) != nil || len(values) != len(fields) {
		return false
	}
	for i, field := range fields {
		if field == nil {
			continue
		}
		if bytes.Equal(bytes.TrimSpace(values[i]), []byte("null")) || json.Unmarshal(values[i], field) != nil {
			return false
		}
	}
	return true
}

func (w *worldsHandler) captureMenu() {
	name := "No active world capture"
	w.currentWorld(func(ws *worldstate.World) { name = ws.Name })
	w.showMenu("Capture & world", name, []menuAction{
		{"Capture status & channels", func() { w.captureStatusMenu() }},
		{"World options\nRename and choose the void generator", func() { w.worldOptionsMenu() }},
		{"Save capture & start a new one", func() {
			w.menuConfirm("Save current capture?", name+"\n\nSave this world and start a fresh capture in a new world tab.", func() { w.saveMenuCapture(); w.captureMenu() }, func() { w.captureMenu() })
		}},
		w.menuBack(),
	})
}

// Called only by submitMenuForm while holding menuWorldMu.
func (w *worldsHandler) saveMenuCapture() {
	found, chunks := false, 0
	w.currentWorld(func(ws *worldstate.World) { found, chunks = true, len(ws.StoredChunks) })
	if !found || chunks == 0 {
		w.commandError("No captured chunks to save yet", "/cb menu")
		return
	}
	w.session.SendMessage("§7Saving this capture and starting a fresh world tab...§r")
	w.saveAndReset(false, nil)
}

func (w *worldsHandler) captureStatusMenu() {
	body := "No active world capture."
	w.currentWorld(func(ws *worldstate.World) {
		body = fmt.Sprintf("World: %s\nCaptured chunks: %d\nVoid generator: %s", ws.Name, len(ws.StoredChunks), menuOnOff(ws.VoidGen))
	})
	w.captureSettingsMu.Lock()
	s := w.settings
	w.captureSettingsMu.Unlock()
	body += fmt.Sprintf("\n\nCapture channels\nTerrain: on\nEntities: %s\nCustom entities: %s\nInventories: %s\nMap PNG: %s\n\n%s", menuOnOff(s.SaveEntities), menuOnOff(s.SaveCustomEntities), menuOnOff(s.SaveInventories), menuOnOff(s.SaveImage), w.session.CaptureStatusText())
	w.showMenu("Capture status", body, []menuAction{{"Refresh status", func() { w.captureStatusMenu() }}, {"World options", func() { w.worldOptionsMenu() }}, {"Back to capture", func() { w.captureMenu() }}, w.menuBack()})
}

func (w *worldsHandler) mapMenu() {
	if w.mapUI == nil {
		w.commandError("The capture map is not ready", "/cb menu")
		return
	}
	state := func(enabled bool) string {
		if enabled {
			return "on"
		}
		return "off"
	}
	w.showMenu("Map & position", fmt.Sprintf("Map: %s\nPosition marker: %s\nZoom: %d", state(w.mapUI.Enabled()), state(w.mapUI.MarkerEnabled()), w.mapUI.Zoom()), []menuAction{
		{"Map settings\nVisibility, position marker and zoom", func() { w.mapSettingsMenu() }},
		{"Add named map marker", func() { w.markerMenu() }},
		{"Browse saved map markers", func() { w.pointBrowserMenu("marker", "", 0) }},
		{"Player position & orientation", func() { w.playerMenu() }},
		w.menuBack(),
	})
}

func (w *worldsHandler) pointMenu(kind, title string) {
	label := "Name (1-100 characters)"
	if kind == "note" {
		label = "Note (1-100 characters)"
	}
	w.inputMenuReturn(title, label, "", "", func(value string) bool { return w.commandPoint(kind, []string{value}) }, func() { w.pointsMenu() })
}

func (w *worldsHandler) markerMenu() {
	icons := make([]string, 0, len(captureMarkerIcons))
	for icon := range captureMarkerIcons {
		icons = append(icons, icon)
	}
	slices.Sort(icons)
	w.showForm(map[string]any{
		"type": "custom_form", "title": "§8Add map marker§r",
		"content": []any{
			map[string]any{"type": "input", "text": "§8Marker name (1-100 characters)", "placeholder": "Home base"},
			map[string]any{"type": "dropdown", "text": "§8Marker type", "options": icons, "default": slices.Index(icons, "pin")},
		},
	}, func(data json.RawMessage) {
		var name string
		var icon int
		if !menuFields(data, &name, &icon) || icon < 0 || icon >= len(icons) {
			return
		}
		if w.commandPoint("marker", []string{icons[icon], strings.TrimSpace(name)}) {
			w.pointsMenu()
		} else {
			w.menuNotice("Marker not saved", "The marker could not be saved. Check the chat message for details.", func() { w.markerMenu() }, func() { w.pointsMenu() })
		}
	})
}

func (w *worldsHandler) blueprintMenu() {
	w.showMenu("Blueprints", w.blueprintSelectionText(), []menuAction{
		{"Set first corner here", func() {
			if w.commandStructure([]string{"pos1"}) {
				w.blueprintMenu()
			}
		}},
		{"Set second corner here", func() {
			if w.commandStructure([]string{"pos2"}) {
				w.blueprintMenu()
			}
		}},
		{"Refresh selection status", func() { w.blueprintMenu() }},
		{"Enter corner coordinates", func() { w.cornerMenu() }},
		{"Save selection", func() {
			w.inputMenuReturn("Save blueprint", "Blueprint name", "", "", func(name string) bool { return w.commandStructure([]string{"save", name}) }, func() { w.blueprintMenu() })
		}},
		{"Clear selection", func() {
			w.menuConfirm("Clear blueprint selection?", "Reset both blueprint corners? Saved blueprints remain available.", func() { w.commandStructure([]string{"clear"}); w.blueprintMenu() }, func() { w.blueprintMenu() })
		}},
		{"Cancel running blueprint save", func() {
			w.menuConfirm("Cancel blueprint save?", "Stop the current blueprint save operation?", func() { w.commandStructure([]string{"cancel"}); w.blueprintMenu() }, func() { w.blueprintMenu() })
		}},
		{"Browse saved blueprints\nDetails, search and open in the desktop", func() { w.blueprintBrowserMenu("", 0) }},
		{"Open Structure Studio in CraftBox", func() { w.commandStructure([]string{"open"}) }},
		w.menuBack(),
	})
}

func (w *worldsHandler) pointsMenu() {
	w.showMenu("Points & notes", "Save your current position. Camera points also keep your viewing direction. Names and notes can contain spaces.", []menuAction{
		{"Browse saved points\nSearch markers, notes and camera positions", func() { w.pointBrowserMenu("", "", 0) }},
		{"Map marker\nChoose a name and marker type", func() { w.markerMenu() }},
		{"Recording mark\nRemember position and recording time", func() { w.pointMenu("mark", "Recording mark") }},
		{"Note\nAttach text to your current position", func() { w.pointMenu("note", "Note") }},
		{"Camera position\nKeep position, yaw and pitch", func() { w.pointMenu("camera", "Camera position") }},
		w.menuBack(),
	})
}

func (w *worldsHandler) mapSettingsMenu() {
	if w.mapUI == nil {
		return
	}
	levels := []int{1, 2, 4, 8, 16}
	w.showForm(map[string]any{
		"type": "custom_form", "title": "§8Map settings§r",
		"content": []any{
			map[string]any{"type": "label", "text": "§8These options affect your live map. Saved map images remain unchanged."},
			map[string]any{"type": "toggle", "text": "§8Show map", "default": w.mapUI.Enabled()},
			map[string]any{"type": "toggle", "text": "§8Show current position", "default": w.mapUI.MarkerEnabled()},
			map[string]any{"type": "step_slider", "text": "§8Map zoom", "steps": []string{"1x", "2x", "4x", "8x", "16x"}, "default": max(0, slices.Index(levels, w.mapUI.Zoom()))},
		},
	}, func(data json.RawMessage) {
		if w.applyMapMenu(data) {
			w.mapMenu()
		}
	})
}

func (w *worldsHandler) applyMapMenu(data json.RawMessage) bool {
	var enabled, position bool
	var zoom int
	if !menuFields(data, nil, &enabled, &position, &zoom) || zoom < 0 || zoom > 4 || w.mapUI == nil {
		return false
	}
	w.mapUI.SetEnabled(enabled)
	w.mapUI.SetMarkerEnabled(position)
	w.mapUI.SetZoom([]int{1, 2, 4, 8, 16}[zoom])
	return true
}

func (w *worldsHandler) worldOptionsMenu() {
	name, void, found := "", false, false
	w.currentWorld(func(ws *worldstate.World) { name, void, found = ws.Name, ws.VoidGen, true })
	if !found {
		w.commandError("No active world capture", "/cb menu")
		return
	}
	w.showForm(map[string]any{
		"type": "custom_form", "title": "§8World options§r",
		"content": []any{
			map[string]any{"type": "label", "text": "§8Configure the local saved world. This does not change the server world."},
			map[string]any{"type": "input", "text": "§8World name", "default": name, "placeholder": "My capture"},
			map[string]any{"type": "toggle", "text": "§8Void generator for uncaptured terrain", "default": void},
		},
	}, func(data json.RawMessage) {
		var newName string
		var newVoid bool
		if !menuFields(data, nil, &newName, &newVoid) {
			return
		}
		newName = strings.TrimSpace(newName)
		if err := validateCaptureName(newName); err != nil {
			w.commandError(err.Error(), "/cb menu")
			return
		}
		if !w.setWorldName(newName) {
			return
		}
		w.setVoidGen(newVoid)
		w.captureMenu()
	})
}

func (w *worldsHandler) cornerMenu() {
	pos, err := structureFeetPosition(w.session.PlayerSnapshot().Position)
	if err != nil {
		w.commandError(err.Error(), "/cb menu")
		return
	}
	w.showForm(map[string]any{
		"type": "custom_form", "title": "§8Blueprint corner§r",
		"content": []any{
			map[string]any{"type": "dropdown", "text": "§8Corner", "options": []string{"First corner", "Second corner"}, "default": 0},
			map[string]any{"type": "toggle", "text": "§8Use my position when submitting", "default": true},
			map[string]any{"type": "input", "text": "§8X (used for manual coordinates)", "default": strconv.Itoa(int(pos[0]))},
			map[string]any{"type": "input", "text": "§8Y (used for manual coordinates)", "default": strconv.Itoa(int(pos[1]))},
			map[string]any{"type": "input", "text": "§8Z (used for manual coordinates)", "default": strconv.Itoa(int(pos[2]))},
		},
	}, func(data json.RawMessage) {
		var corner int
		var current bool
		var x, y, z string
		if !menuFields(data, &corner, &current, &x, &y, &z) || corner < 0 || corner > 1 {
			return
		}
		args := []string{[]string{"pos1", "pos2"}[corner]}
		if !current {
			args = append(args, strings.TrimSpace(x), strings.TrimSpace(y), strings.TrimSpace(z))
		}
		if w.commandStructure(args) {
			w.blueprintMenu()
		}
	})
}

func (w *worldsHandler) regionMenu() {
	w.showMenu("Regions", w.regionSelectionText(), []menuAction{
		{"Set first corner here", func() {
			if w.commandRegion([]string{"pos1"}) {
				w.regionMenu()
			}
		}},
		{"Set second corner here", func() {
			if w.commandRegion([]string{"pos2"}) {
				w.regionMenu()
			}
		}},
		{"Refresh selection status", func() { w.regionMenu() }},
		{"Browse created regions", func() { w.regionBrowserMenu(0) }},
		{"Create region copy", func() {
			w.inputMenuReturn("Create region", "Region name", "", "", func(name string) bool { return w.commandRegion([]string{"create", name}) }, func() { w.regionMenu() })
		}},
		{"Clear region selection", func() {
			w.menuConfirm("Clear region selection?", "Reset the selected region corners? Created region copies remain available.", func() { w.commandRegion([]string{"clear"}); w.regionMenu() }, func() { w.regionMenu() })
		}},
		w.menuBack(),
	})
}

func (w *worldsHandler) exclusionsMenu(page int) {
	patterns := w.excludedMobs()
	pages := max(1, (len(patterns)+7)/8)
	page = max(0, min(page, pages-1))
	body := fmt.Sprintf("Page %d/%d\nExclusions affect entities in the saved world.\n", page+1, pages)
	if len(patterns) == 0 {
		body += "\nNo exclusions configured."
	} else {
		body += "\n" + strings.Join(patterns[page*8:min(page*8+8, len(patterns))], "\n")
	}
	actions := []menuAction{
		{"Browse server entity types\nSearch the catalog and exclude an entity", func() { w.entityBrowserMenu("", 0) }},
		{"Add exclusion\nEntity ID or wildcard pattern", func() {
			w.inputMenuReturn("Exclude entities", "IDs or patterns separated by spaces", "", "", func(value string) bool { return w.commandExclude(strings.Fields(value)) }, func() { w.exclusionsMenu(0) })
		}},
	}
	if len(patterns) > 0 {
		actions = append(actions, menuAction{"Remove one exclusion", func() {
			w.showForm(map[string]any{"type": "custom_form", "title": "§8Remove exclusion§r", "content": []any{
				map[string]any{"type": "dropdown", "text": "§8Exclusion to remove", "options": patterns, "default": 0},
			}}, func(data json.RawMessage) {
				var index int
				if menuFields(data, &index) && index >= 0 && index < len(patterns) {
					w.commandInclude([]string{patterns[index]})
					w.exclusionsMenu(page)
				}
			})
		}}, menuAction{"Clear all exclusions", func() {
			w.showForm(map[string]any{"type": "modal", "title": "§8Clear exclusions?§r", "content": "Include all entity types again in the saved world?", "button1": "§9Clear all§r", "button2": "§8Cancel§r"}, func(data json.RawMessage) {
				var confirm bool
				if json.Unmarshal(data, &confirm) == nil && confirm {
					w.commandInclude([]string{"all"})
					w.exclusionsMenu(0)
				}
			})
		}})
	}
	if page > 0 {
		actions = append(actions, menuAction{"Previous page", func() { w.exclusionsMenu(page - 1) }})
	}
	if page+1 < pages {
		actions = append(actions, menuAction{"Next page", func() { w.exclusionsMenu(page + 1) }})
	}
	actions = append(actions, w.menuBack())
	w.showMenu("Entity exclusions", body, actions)
}

func (w *worldsHandler) helpMenu(page int) {
	pages := []struct{ title, content string }{
		{"Capture & world", "All CraftBox actions affect your local capture.\n\nWorld options change the saved name and void generator. Saving finishes the current capture and begins a new one without disconnecting.\n\nCapture status shows recorded chunks and enabled capture channels.\n\nCommands\n/cb menu\n/cb status\n/cb name <name>\n/cb void [on|off|toggle]\n/cb save\n/cb connection"},
		{"Map & points", "Map settings control visibility, the live position marker and zoom. Saved map images are not changed by these display options.\n\nNamed map markers, notes and recording marks save your position. Camera points also save the viewing direction.\n\nCommands\n/cb map [on|off|toggle]\n/cb position [on|off|toggle]\n/cb zoom [1|2|4|8|16]\n/cb marker <type> <name>\n/cb note <text>\n/cb mark <name>\n/cb camera add <name>"},
		{"Blueprints, regions & entities", "Blueprints use exact block corners. Set each corner at your position or enter coordinates, then save the selection.\n\nRegions copy complete chunk columns between two selected corners.\n\nEntity exclusions filter which entities are saved. Use entity IDs or wildcard patterns; removing an exclusion includes those entities again.\n\nCommands\n/cb structure pos1|pos2 [x y z]\n/cb structure save <name>\n/cb region pos1|pos2|clear\n/cb region create <name>\n/cb exclude <id/pattern>\n/cb include <pattern|all>"},
		{"Browsing & navigation", "Quick actions show your current position and common map/recording controls.\n\nSaved points can be searched by name and filtered by type. Point details include precise coordinates, recording time and camera orientation.\n\nThe blueprint library lists saved projects alphabetically. Search by name, inspect dimensions, then open the chosen blueprint in CraftBox.\n\nThe entity catalog comes from the connected server. Search for an ID to add or remove an exact exclusion. Existing wildcard rules still apply.\n\nLists use six items per page. Clear the search text to show all results. Use Back to return to the previous section."},
	}
	page = max(0, min(page, len(pages)-1))
	var actions []menuAction
	if page > 0 {
		actions = append(actions, menuAction{"Previous page", func() { w.helpMenu(page - 1) }})
	}
	if page+1 < len(pages) {
		actions = append(actions, menuAction{"Next page", func() { w.helpMenu(page + 1) }})
	}
	actions = append(actions, w.menuBack())
	w.showMenu(fmt.Sprintf("Help %d/%d: %s", page+1, len(pages), pages[page].title), pages[page].content, actions)
}
