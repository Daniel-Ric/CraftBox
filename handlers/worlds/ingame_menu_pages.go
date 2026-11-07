package worlds

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
)

func (w *worldsHandler) menuDashboardText() string {
	body := "No active capture."
	w.currentWorld(func(ws *worldstate.World) {
		body = fmt.Sprintf("§d%s§r\nCaptured chunks: %d", ws.Name, len(ws.StoredChunks))
	})
	if w.session != nil {
		if p, err := structureFeetPosition(w.session.PlayerSnapshot().Position); err == nil {
			body += fmt.Sprintf("\nPosition: %d, %d, %d", p[0], p[1], p[2])
		}
	}
	if w.mapUI != nil {
		body += fmt.Sprintf("\nMap: %s | Position marker: %s | Zoom: %dx", menuOnOff(w.mapUI.Enabled()), menuOnOff(w.mapUI.MarkerEnabled()), w.mapUI.Zoom())
	}
	return body + "\n\nChoose a category. Open again with /cb menu."
}

func menuOnOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func (w *worldsHandler) menuNotice(title, body string, retry, back func()) {
	var actions []menuAction
	if retry != nil {
		actions = append(actions, menuAction{"Try again", retry})
	}
	actions = append(actions, menuAction{"Back", back}, w.menuBack())
	w.showMenu(title, body, actions)
}

func (w *worldsHandler) menuConfirm(title, body string, yes, no func()) {
	w.showForm(map[string]any{"type": "modal", "title": "§8" + title + "§r", "content": body, "button1": "§9Confirm§r", "button2": "§8Cancel§r"}, func(data json.RawMessage) {
		var confirm *bool
		if json.Unmarshal(data, &confirm) != nil || confirm == nil {
			return
		}
		if *confirm {
			yes()
		} else {
			no()
		}
	})
}

func (w *worldsHandler) quickMenu() {
	actions := []menuAction{
		{"Player position & orientation", func() { w.playerMenu() }},
		{"Save recording mark", func() { w.pointMenu("mark", "Recording mark") }},
		{"Save camera position", func() { w.pointMenu("camera", "Camera position") }},
		{"Add note", func() { w.pointMenu("note", "Note") }},
	}
	if w.mapUI != nil {
		actions = append(actions, menuAction{"Toggle map (currently " + menuOnOff(w.mapUI.Enabled()) + ")", func() { w.commandMap([]string{"toggle"}); w.quickMenu() }}, menuAction{"Toggle position marker (currently " + menuOnOff(w.mapUI.MarkerEnabled()) + ")", func() { w.commandPosition([]string{"toggle"}); w.quickMenu() }})
	}
	actions = append(actions, menuAction{"Capture status", func() { w.captureStatusMenu() }}, w.menuBack())
	w.showMenu("Quick actions", w.menuDashboardText(), actions)
}

func (w *worldsHandler) playerMenu() {
	p := w.session.PlayerSnapshot()
	body := "Player position is not available yet."
	if feet, err := structureFeetPosition(p.Position); err == nil {
		body = fmt.Sprintf("Feet block: %d, %d, %d\nReceived position: %.3f, %.3f, %.3f\nChunk: %d, %d\nYaw: %.1f\nPitch: %.1f", feet[0], feet[1], feet[2], p.Position[0], p.Position[1], p.Position[2], int32(math.Floor(float64(p.Position[0])/16)), int32(math.Floor(float64(p.Position[2])/16)), p.Yaw, p.Pitch)
	}
	w.showMenu("Player position", body, []menuAction{{"Refresh", func() { w.playerMenu() }}, {"Save this position as a mark", func() { w.pointMenu("mark", "Recording mark") }}, {"Back to quick actions", func() { w.quickMenu() }}, w.menuBack()})
}

func (w *worldsHandler) connectionMenu() {
	body := w.session.ConnectionDetailsText()
	if !w.commandStarted.IsZero() {
		body += "\nSession duration: " + time.Since(w.commandStarted).Round(time.Second).String()
	}
	version := utils.Version
	if version == "" {
		version = "development"
	}
	body += "\nCraftBox: " + version
	w.showMenu("Connection & session", body, []menuAction{{"Refresh connection details", func() { w.connectionMenu() }}, {"Player position & orientation", func() { w.playerMenu() }}, {"Capture channels", func() { w.captureStatusMenu() }}, w.menuBack()})
}

func (w *worldsHandler) blueprintSelectionText() string {
	w.worldStateMu.Lock()
	defer w.worldStateMu.Unlock()
	s := w.commandStructureSelection
	if w.worldState == nil || s.worldID != w.worldState.ID {
		return "No blueprint selection in this capture.\nSet both block corners to get started."
	}
	body := "First corner: not set\nSecond corner: not set"
	if s.first != nil {
		body = fmt.Sprintf("First corner: %v\nSecond corner: not set", *s.first)
	}
	if s.second != nil {
		body = strings.Replace(body, "Second corner: not set", fmt.Sprintf("Second corner: %v", *s.second), 1)
	}
	if s.first != nil && s.second != nil {
		size := (structure.Bounds{Min: *s.first, Max: *s.second}).Normalized().Size()
		body += fmt.Sprintf("\nSize: %d x %d x %d blocks", size[0], size[1], size[2])
	}
	if s.cancel != nil {
		body += "\nA blueprint save is running."
	}
	return body
}

func (w *worldsHandler) regionSelectionText() string {
	w.worldStateMu.Lock()
	defer w.worldStateMu.Unlock()
	s := w.commandRegionSelection
	if w.worldState == nil || s.worldID != w.worldState.ID {
		return "No region selection in this capture.\nRegions copy complete chunk columns. Set both corners to begin."
	}
	body := "First chunk corner: not set\nSecond chunk corner: not set"
	if s.first != nil {
		body = fmt.Sprintf("First chunk corner: %v\nSecond chunk corner: not set", *s.first)
	}
	if s.second != nil {
		body = strings.Replace(body, "Second chunk corner: not set", fmt.Sprintf("Second chunk corner: %v", *s.second), 1)
	}
	if s.first != nil && s.second != nil {
		a, b := *s.first, *s.second
		x, z := int64(max(a[0], b[0]))-int64(min(a[0], b[0]))+1, int64(max(a[1], b[1]))-int64(min(a[1], b[1]))+1
		body += fmt.Sprintf("\nSize: %d x %d chunks\nFootprint: %d x %d blocks\nTotal columns: %d", x, z, x*16, z*16, x*z)
	}
	return body
}

// All browsers use six entries per page and clamp stale/empty page indices.
func menuPage(total, page int) (start, end, current, pages int) {
	pages = max(1, (total+5)/6)
	current = max(0, min(page, pages-1))
	start = current * 6
	end = min(start+6, total)
	return
}

func menuNavigation(actions []menuAction, page, pages int, show func(int), back menuAction) []menuAction {
	if page > 0 {
		actions = append(actions, menuAction{"Previous page", func() { show(page - 1) }})
	}
	if page+1 < pages {
		actions = append(actions, menuAction{"Next page", func() { show(page + 1) }})
	}
	return append(actions, back)
}

func filterMenuPoints(points []messages.CapturePoint, kind, query string) []messages.CapturePoint {
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]messages.CapturePoint, 0, len(points))
	for _, p := range points {
		if (kind == "" || p.Kind == kind) && (query == "" || strings.Contains(strings.ToLower(p.Name), query)) {
			result = append(result, p)
		}
	}
	slices.SortStableFunc(result, func(a, b messages.CapturePoint) int { return b.Created.Compare(a.Created) })
	return result
}

func (w *worldsHandler) pointBrowserMenu(kind, query string, page int) {
	var points []messages.CapturePoint
	w.currentWorld(func(ws *worldstate.World) { points = messages.ReadCapturePoints(ws.Folder) })
	points = filterMenuPoints(points, kind, query)
	start, end, page, pages := menuPage(len(points), page)
	body := fmt.Sprintf("%d matching points | Page %d/%d\nNewest entries first. Select a point for coordinates and recording time.", len(points), page+1, pages)
	if query != "" {
		body += "\nSearch: " + query
	}
	if kind != "" {
		body += "\nType: " + kind
	}
	if len(points) == 0 {
		body += "\nNo points match this filter."
	}
	actions := []menuAction{{"Search & filter", func() { w.pointFilterMenu(kind, query) }}}
	for _, p := range points[start:end] {
		actions = append(actions, menuAction{p.Name + "\n" + p.Kind, func() {
			body := fmt.Sprintf("%s\nType: %s\nX: %.3f\nY: %.3f\nZ: %.3f\nRecording time: %.1fs\nYaw: %.1f | Pitch: %.1f\nCreated: %s", p.Name, p.Kind, p.Position[0], p.Position[1], p.Position[2], p.Elapsed, p.Yaw, p.Pitch, p.Created.Local().Format("2006-01-02 15:04:05"))
			w.showMenu("Point details", body, []menuAction{{"Back to saved points", func() { w.pointBrowserMenu(kind, query, page) }}, w.menuBack()})
		}})
	}
	actions = menuNavigation(actions, page, pages, func(p int) { w.pointBrowserMenu(kind, query, p) }, menuAction{"Back to points & notes", func() { w.pointsMenu() }})
	w.showMenu("Saved points", body, actions)
}

func (w *worldsHandler) pointFilterMenu(kind, query string) {
	kinds := []string{"", "marker", "mark", "note", "camera"}
	w.showForm(map[string]any{"type": "custom_form", "title": "§8Search saved points§r", "content": []any{
		map[string]any{"type": "input", "text": "§8Name contains", "default": query, "placeholder": "Leave empty for all names"},
		map[string]any{"type": "dropdown", "text": "§8Point type", "options": []string{"All types", "Map markers", "Recording marks", "Notes", "Camera positions"}, "default": max(0, slices.Index(kinds, kind))},
	}}, func(data json.RawMessage) {
		var query string
		var index int
		if menuFields(data, &query, &index) && index >= 0 && index < len(kinds) {
			w.pointBrowserMenu(kinds[index], strings.TrimSpace(query), 0)
		}
	})
}

type menuBlueprint struct {
	id    string
	brief structure.ProjectBrief
}

func readMenuBlueprints(folder, query string) ([]menuBlueprint, int, error) {
	items, err := os.ReadDir(folder)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var found []menuBlueprint
	skipped := 0
	query = strings.ToLower(strings.TrimSpace(query))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		brief, err := structure.ReadProjectBrief(filepath.Join(folder, item.Name(), "recovery.craftbox"))
		if err != nil {
			skipped++
			continue
		}
		if _, err = structure.ValidateSize(brief.Size); err != nil {
			skipped++
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(brief.Name), query) {
			found = append(found, menuBlueprint{item.Name(), brief})
		}
	}
	slices.SortStableFunc(found, func(a, b menuBlueprint) int {
		return strings.Compare(strings.ToLower(a.brief.Name), strings.ToLower(b.brief.Name))
	})
	return found, skipped, nil
}

func (w *worldsHandler) blueprintBrowserMenu(query string, page int) {
	items, skipped, err := readMenuBlueprints(utils.PathData("structures", "sessions"), query)
	if err != nil {
		w.menuNotice("Blueprint library unavailable", err.Error(), func() { w.blueprintBrowserMenu(query, page) }, func() { w.blueprintMenu() })
		return
	}
	start, end, page, pages := menuPage(len(items), page)
	body := fmt.Sprintf("%d blueprints | Page %d/%d", len(items), page+1, pages)
	if skipped > 0 {
		body += fmt.Sprintf("\n%d unreadable projects skipped.", skipped)
	}
	if query != "" {
		body += "\nSearch: " + query
	}
	if len(items) == 0 {
		body += "\nNo saved blueprints match this search."
	}
	actions := []menuAction{{"Search blueprints", func() { w.searchMenu("Search blueprints", query, func(q string) { w.blueprintBrowserMenu(q, 0) }) }}}
	for _, item := range items[start:end] {
		size := item.brief.Size
		actions = append(actions, menuAction{fmt.Sprintf("%s\n%d x %d x %d blocks", item.brief.Name, size[0], size[1], size[2]), func() {
			volume, _ := structure.ValidateSize(size)
			w.showMenu("Blueprint details", fmt.Sprintf("%s\nSize: %d x %d x %d\nVolume: %d blocks", item.brief.Name, size[0], size[1], size[2], volume), []menuAction{
				{"Open this blueprint in CraftBox", func() {
					messages.SendEvent(&messages.EventStructureCreated{ID: item.id, Name: item.brief.Name, Open: true})
					w.menuNotice("Opening in CraftBox", "Switch to CraftBox to view this blueprint in Structure Studio.", nil, func() { w.blueprintBrowserMenu(query, page) })
				}},
				{"Back to blueprint library", func() { w.blueprintBrowserMenu(query, page) }},
			})
		}})
	}
	actions = menuNavigation(actions, page, pages, func(p int) { w.blueprintBrowserMenu(query, p) }, menuAction{"Back to blueprints", func() { w.blueprintMenu() }})
	w.showMenu("Blueprint library", body, actions)
}

func (w *worldsHandler) searchMenu(title, value string, submit func(string)) {
	w.showForm(map[string]any{"type": "custom_form", "title": "§8" + title + "§r", "content": []any{map[string]any{"type": "input", "text": "§8Search text (empty shows all)", "default": value}}}, func(data json.RawMessage) {
		var query string
		if menuFields(data, &query) {
			submit(strings.TrimSpace(query))
		}
	})
}

func (w *worldsHandler) entityBrowserMenu(query string, page int) {
	w.captureSettingsMu.Lock()
	var ids []string
	for id := range w.commandEntityTypes {
		if strings.Contains(strings.ToLower(id), strings.ToLower(query)) {
			ids = append(ids, id)
		}
	}
	w.captureSettingsMu.Unlock()
	slices.Sort(ids)
	start, end, page, pages := menuPage(len(ids), page)
	body := fmt.Sprintf("%d server entity types | Page %d/%d\nChoose an entity to manage its exclusion.", len(ids), page+1, pages)
	if len(ids) == 0 {
		body += "\nNo matches, or the server catalog has not arrived yet."
	}
	if query != "" {
		body += "\nSearch: " + query
	}
	actions := []menuAction{{"Search entity types", func() { w.searchMenu("Search entities", query, func(q string) { w.entityBrowserMenu(q, 0) }) }}}
	for _, id := range ids[start:end] {
		actions = append(actions, menuAction{id, func() {
			patterns := w.excludedMobs()
			configured := slices.Contains(patterns, id)
			body := id + "\nExact exclusion: " + menuOnOff(configured) + "\nWildcard rules may also exclude this type."
			detail := []menuAction{}
			if configured {
				detail = append(detail, menuAction{"Remove exact exclusion", func() { w.commandInclude([]string{id}); w.entityBrowserMenu(query, page) }})
			} else {
				detail = append(detail, menuAction{"Exclude this entity from saved worlds", func() { w.commandExclude([]string{id}); w.entityBrowserMenu(query, page) }})
			}
			detail = append(detail, menuAction{"Back to entity catalog", func() { w.entityBrowserMenu(query, page) }})
			w.showMenu("Entity type", body, detail)
		}})
	}
	actions = menuNavigation(actions, page, pages, func(p int) { w.entityBrowserMenu(query, p) }, menuAction{"Back to exclusions", func() { w.exclusionsMenu(0) }})
	w.showMenu("Server entity catalog", body, actions)
}

func (w *worldsHandler) regionBrowserMenu(page int) {
	w.worldStateMu.Lock()
	regions := slices.Clone(w.regions)
	w.worldStateMu.Unlock()
	start, end, page, pages := menuPage(len(regions), page)
	body := fmt.Sprintf("%d created regions | Page %d/%d", len(regions), page+1, pages)
	if len(regions) == 0 {
		body += "\nNo region copies created in this capture yet."
	}
	var actions []menuAction
	for _, r := range regions[start:end] {
		actions = append(actions, menuAction{r.Name, func() {
			b := r.Bounds
			x, z := int64(b.MaxX)-int64(b.MinX)+1, int64(b.MaxZ)-int64(b.MinZ)+1
			w.showMenu("Region details", fmt.Sprintf("%s\nChunks: %d, %d to %d, %d\nSize: %d x %d chunks\nFootprint: %d x %d blocks", r.Name, b.MinX, b.MinZ, b.MaxX, b.MaxZ, x, z, x*16, z*16), []menuAction{{"Back to created regions", func() { w.regionBrowserMenu(page) }}, w.menuBack()})
		}})
	}
	actions = menuNavigation(actions, page, pages, func(p int) { w.regionBrowserMenu(p) }, menuAction{"Back to regions", func() { w.regionMenu() }})
	w.showMenu("Created regions", body, actions)
}
