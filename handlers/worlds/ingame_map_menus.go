package worlds

import (
	"fmt"
	"strings"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/df-mc/dragonfly/server/block/cube"
)

func (w *worldsHandler) navigationMenu() {
	if w.mapUI == nil {
		return
	}
	body := "Set both region corners to show capture coverage."
	w.currentWorld(func(ws *worldstate.World) {
		if bounds, ok := w.commandRegionSelection.bounds(); ok && w.commandRegionSelection.worldID == ws.ID {
			var count int
			for _, p := range w.mapUI.coveragePositions(ws.ID) {
				if bounds.Contains(p[0], p[1]) {
					count++
				}
			}
			body = locale.Format("Region: %d/%d chunks", count, coverageArea(bounds)) + "\n" + locale.Text("The amber outline marks the region, orange tiles are missing chunks and the blue marker is your destination.")
		}
	})
	w.showMenu("Capture navigation", body, []menuAction{
		{"Toggle coverage overlay", func() { w.commandMapNavigation([]string{"coverage"}); w.navigationMenu() }},
		{"Navigate to next missing chunk", func() { w.navigateToGap() }},
		{"Choose a saved marker", func() { w.pointBrowserMenu("marker", "", 0) }},
		{"Clear map destination", func() { w.commandMapNavigation([]string{"clear"}); w.navigationMenu() }},
		{"Select region corners", func() { w.regionMenu() }},
		{"Back to map", func() { w.mapMenu() }},
		w.menuBack(),
	})
}

func mapWallStatus(wall []worldstate.MapFrame, known map[int64]*worldstate.Map) (complete int, missing []worldstate.MapFrame) {
	for _, frame := range wall {
		if known[frame.MapID].PixelsComplete() {
			complete++
		} else {
			missing = append(missing, frame)
		}
	}
	return
}

func (w *worldsHandler) mapArtMenu(page int) {
	var walls [][]worldstate.MapFrame
	w.currentWorld(func(ws *worldstate.World) { walls = ws.CapturedMapWalls() })
	known := w.knownMapsSnapshot()
	start, end, page, pages := menuPage(len(walls), page)
	body := locale.Format("Captured map groups: %d | Page %d/%d", len(walls), page+1, pages)
	if len(walls) == 0 {
		body = "No map frames have been captured in this world."
	}
	actions := []menuAction{{"Refresh", func() { w.mapArtMenu(page) }}}
	for _, wall := range walls[start:end] {
		complete, _ := mapWallStatus(wall, known)
		p := wall[0].Pos
		actions = append(actions, menuAction{locale.Format("Frame group at %d, %d, %d\n%d/%d maps complete", p[0], p[1], p[2], complete, len(wall)), func() {
			w.mapWallMenu(p, page, 0)
		}})
	}
	actions = menuNavigation(actions, page, pages, w.mapArtMenu, menuAction{"Back to map", func() { w.mapMenu() }})
	w.showMenu("Map art check", body, actions)
}

func (w *worldsHandler) mapWallAt(anchor cube.Pos) (result []worldstate.MapFrame) {
	w.currentWorld(func(ws *worldstate.World) {
		for _, wall := range ws.CapturedMapWalls() {
			for _, frame := range wall {
				if frame.Pos == anchor {
					result = wall
					return
				}
			}
		}
	})
	return
}

func (w *worldsHandler) mapWallMenu(anchor cube.Pos, listPage, page int) {
	wall := w.mapWallAt(anchor)
	if len(wall) == 0 {
		w.mapArtMenu(listPage)
		return
	}
	complete, missing := mapWallStatus(wall, w.knownMapsSnapshot())
	start, end, page, pages := menuPage(len(missing), page)
	lines := []string{locale.Format("%d/%d maps complete", complete, len(wall))}
	visible := missing[start:end]
	actions := []menuAction{{"Refresh", func() { w.mapWallMenu(anchor, listPage, page) }}}
	if len(visible) > 0 {
		lines = append(lines, locale.Format("Missing frames | Page %d/%d", page+1, pages))
		for _, frame := range visible {
			p := frame.Pos
			lines = append(lines, fmt.Sprintf("X: %d · Y: %d · Z: %d", p[0], p[1], p[2]))
		}
		actions = append(actions, menuAction{"Retry missing maps on this page", func() {
			var requested int
			_, current := mapWallStatus(w.mapWallAt(anchor), w.knownMapsSnapshot())
			from, to, _, _ := menuPage(len(current), page)
			for _, frame := range current[from:to] {
				if w.requestMap(frame.MapID, true) {
					requested++
				}
			}
			if requested == 0 {
				w.session.SendMessage("No new map requests sent. Wait a moment and refresh.")
			} else {
				w.session.SendActionBarOrMessage(locale.Format("Requested %d missing maps", requested))
			}
			w.mapWallMenu(anchor, listPage, page)
		}})
	}
	actions = menuNavigation(actions, page, pages, func(p int) { w.mapWallMenu(anchor, listPage, p) }, menuAction{"Back to map art", func() { w.mapArtMenu(listPage) }})
	w.showMenu("Map frame details", strings.Join(lines, "\n"), actions)
}
