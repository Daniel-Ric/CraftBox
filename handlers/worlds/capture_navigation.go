package worlds

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type mapDestination struct {
	name string
	x, z float64
	gap  *protocol.ChunkPos
}

type streamedCoverage struct {
	minY, size int32
	received   int32
	sections   [4]uint64
}

func (m *MapUI) recordCoverageLevelChunk(pos protocol.ChunkPos, minY, size int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size == 0 {
		m.markColumnReceived(pos)
		return
	}
	if _, complete := m.receivedColumns[pos]; !complete && size > 0 && size <= 256 {
		if previous, exists := m.streamedColumns[pos]; !exists || previous.minY != minY || previous.size != size {
			m.streamedColumns[pos] = streamedCoverage{minY: minY, size: size}
		}
	}
}

func (m *MapUI) recordCoverageSubChunks(pos protocol.ChunkPos, entries []int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	column, exists := m.streamedColumns[pos]
	if !exists {
		return
	}
	for _, y := range entries {
		i := y - column.minY
		if i >= 0 && i < column.size {
			bit := uint64(1) << (i % 64)
			if column.sections[i/64]&bit == 0 {
				column.sections[i/64] |= bit
				column.received++
			}
		}
	}
	if column.received == column.size {
		m.markColumnReceived(pos)
	} else {
		m.streamedColumns[pos] = column
	}
}

func (m *MapUI) markColumnReceived(pos protocol.ChunkPos) {
	delete(m.streamedColumns, pos)
	if _, known := m.receivedColumns[pos]; known {
		return
	}
	m.receivedColumns[pos] = struct{}{}
	if m.coverageValid && m.coverageBounds.Contains(pos[0], pos[1]) {
		m.coverageCount++
	}
	m.SchedRedraw()
}

func (m *MapUI) coveragePositions(worldID string) []protocol.ChunkPos {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.worldID != worldID {
		return nil
	}
	chunks := make([]protocol.ChunkPos, 0, len(m.receivedColumns))
	for pos := range m.receivedColumns {
		chunks = append(chunks, pos)
	}
	return chunks
}

func (s commandRegionSelection) bounds() (messages.ChunkBounds, bool) {
	if s.first == nil || s.second == nil {
		return messages.ChunkBounds{}, false
	}
	a, b := *s.first, *s.second
	return messages.ChunkBounds{MinX: min(a[0], b[0]), MinZ: min(a[1], b[1]), MaxX: max(a[0], b[0]), MaxZ: max(a[1], b[1])}, true
}

func coverageArea(b messages.ChunkBounds) int64 {
	return (int64(b.MaxX) - int64(b.MinX) + 1) * (int64(b.MaxZ) - int64(b.MinZ) + 1)
}

func captureCoverage(b messages.ChunkBounds, positions []protocol.ChunkPos, origin protocol.ChunkPos) (int, *protocol.ChunkPos) {
	captured := make(map[protocol.ChunkPos]struct{})
	for _, p := range positions {
		if b.Contains(p[0], p[1]) {
			captured[p] = struct{}{}
		}
	}
	if int64(len(captured)) == coverageArea(b) {
		return len(captured), nil
	}
	start := protocol.ChunkPos{min(max(origin[0], b.MinX), b.MaxX), min(max(origin[1], b.MinZ), b.MaxZ)}
	queue := []protocol.ChunkPos{start}
	seen := map[protocol.ChunkPos]struct{}{start: {}}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		if _, ok := captured[p]; !ok {
			return len(captured), &p
		}
		for _, step := range [...]protocol.ChunkPos{{0, 1}, {1, 0}, {0, -1}, {-1, 0}} {
			next := protocol.ChunkPos{p[0] + step[0], p[1] + step[1]}
			if _, ok := seen[next]; !ok && b.Contains(next[0], next[1]) {
				seen[next] = struct{}{}
				queue = append(queue, next)
			}
		}
	}
	return len(captured), nil
}

func (w *worldsHandler) syncMapCoverage() {
	w.currentWorld(func(ws *worldstate.World) {
		bounds, valid := w.commandRegionSelection.bounds()
		valid = valid && w.commandRegionSelection.worldID == ws.ID
		w.mapUI.setCoverageBounds(ws.ID, bounds, valid)
	})
}

func (w *worldsHandler) commandMapNavigation(args []string) bool {
	const usage = "/cb navigate coverage|next|clear|target <marker>"
	if w.mapUI == nil {
		return w.commandError("The capture map is not ready", usage)
	}
	switch {
	case len(args) == 1 && args[0] == "coverage":
		w.syncMapCoverage()
		enabled, available := w.mapUI.toggleCoverage()
		if !available {
			return w.commandError("Set both region corners in this world first", usage)
		}
		if enabled {
			w.mapUI.SetEnabled(true)
		}
		w.session.SendActionBarOrMessage(locale.Format("Coverage overlay: %s", locale.Text(menuOnOff(enabled))))
	case len(args) == 1 && args[0] == "next":
		return w.navigateToGap()
	case len(args) == 1 && args[0] == "clear":
		w.mapUI.clearDestination()
		w.session.SendMessage("Map destination cleared")
	case len(args) >= 2 && args[0] == "target":
		name := strings.Join(args[1:], " ")
		var matches []messages.CapturePoint
		w.currentWorld(func(ws *worldstate.World) {
			for _, point := range messages.ReadCapturePoints(ws.Folder) {
				if point.Kind == "marker" && strings.EqualFold(point.Name, name) {
					matches = append(matches, point)
				}
			}
			if len(matches) == 1 {
				w.navigateToPoint(ws.ID, matches[0])
			}
		})
		if len(matches) != 1 {
			return w.commandError("Choose a matching saved marker from the menu", usage)
		}
	default:
		return w.commandError("Choose a map navigation action", usage)
	}
	return true
}

func (w *worldsHandler) navigateToGap() bool {
	var bounds messages.ChunkBounds
	var id string
	var chunks []protocol.ChunkPos
	w.currentWorld(func(ws *worldstate.World) {
		if selection, valid := w.commandRegionSelection.bounds(); valid && w.commandRegionSelection.worldID == ws.ID {
			bounds, id, chunks = selection, ws.ID, w.mapUI.coveragePositions(ws.ID)
		}
	})
	if id == "" {
		return w.commandError("Set both region corners in this world first", "/cb region pos1")
	}
	pos := w.session.PlayerSnapshot().Position
	count, gap := captureCoverage(bounds, chunks, positionChunk(pos))
	if gap == nil {
		w.mapUI.clearDestination()
		w.session.SendMessage("All chunks in this region are captured")
		return true
	}
	w.mapUI.setCoverageBounds(id, bounds, true)
	w.mapUI.setDestination(id, mapDestination{name: "Missing chunk", x: float64(gap[0])*16 + 8, z: float64(gap[1])*16 + 8, gap: gap}, true)
	w.session.SendMessage(locale.Format("Region: %d/%d chunks · Next missing chunk: %d, %d", count, coverageArea(bounds), gap[0], gap[1]))
	return true
}

func (w *worldsHandler) navigateToPoint(id string, point messages.CapturePoint) {
	w.mapUI.setDestination(id, mapDestination{name: point.Name, x: float64(point.Position[0]), z: float64(point.Position[2])}, false)
}

func positionChunk(pos mgl32.Vec3) protocol.ChunkPos {
	return protocol.ChunkPos{int32(math.Floor(float64(pos[0]) / 16)), int32(math.Floor(float64(pos[2]) / 16))}
}

func (m *MapUI) setCoverageBounds(id string, bounds messages.ChunkBounds, valid bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.worldID != id {
		return
	}
	if !valid || m.coverageBounds != bounds {
		if m.destination.gap != nil {
			m.destination = mapDestination{}
		}
	}
	if m.coverageValid != valid || m.coverageBounds != bounds {
		m.coverageCount = 0
		if valid {
			for pos := range m.receivedColumns {
				if bounds.Contains(pos[0], pos[1]) {
					m.coverageCount++
				}
			}
		}
	}
	m.coverageBounds, m.coverageValid = bounds, valid
	m.coverageEnabled = m.coverageEnabled && valid
	m.SchedRedraw()
}

func (m *MapUI) toggleCoverage() (bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.coverageValid {
		m.coverageEnabled = !m.coverageEnabled
		m.SchedRedraw()
	}
	return m.coverageEnabled, m.coverageValid
}

func (m *MapUI) setDestination(id string, destination mapDestination, coverage bool) {
	m.mu.Lock()
	if m.worldID != id {
		m.mu.Unlock()
		return
	}
	m.destination = destination
	m.coverageEnabled = m.coverageEnabled || (coverage && m.coverageValid)
	m.mu.Unlock()
	m.SetEnabled(true)
	m.SchedRedraw()
}

func (m *MapUI) clearDestination() {
	m.mu.Lock()
	m.destination = mapDestination{}
	m.mu.Unlock()
	m.SchedRedraw()
}

func (m *MapUI) drawNavigation(position mgl32.Vec3, scale float64) {
	origin := image.Pt(int(position.X()), int(position.Z()))
	project := func(x, z float64) image.Point {
		return image.Pt(int(math.Floor((x-float64(origin.X))*scale))+64, int(math.Floor((z-float64(origin.Y))*scale))+64)
	}
	if m.coverageEnabled {
		b := m.coverageBounds
		lower := positionChunk(mgl32.Vec3{float32(float64(origin.X) - 65/scale), 0, float32(float64(origin.Y) - 65/scale)})
		upper := positionChunk(mgl32.Vec3{float32(float64(origin.X) + 65/scale), 0, float32(float64(origin.Y) + 65/scale)})
		size := int(scale * 16)
		missing := image.NewUniform(color.RGBA{R: 164, G: 87, B: 54, A: 255})
		for z := max(b.MinZ, lower[1]); z <= min(b.MaxZ, upper[1]); z++ {
			for x := max(b.MinX, lower[0]); x <= min(b.MaxX, upper[0]); x++ {
				p := project(float64(x)*16, float64(z)*16)
				if _, captured := m.receivedColumns[protocol.ChunkPos{x, z}]; !captured {
					draw.Draw(m.mapImage, image.Rect(p.X, p.Y, p.X+size, p.Y+size), missing, image.Point{}, draw.Src)
				}
			}
		}
		a, end := project(float64(b.MinX)*16, float64(b.MinZ)*16), project((float64(b.MaxX)+1)*16, (float64(b.MaxZ)+1)*16)
		line := image.NewUniform(color.RGBA{R: 255, G: 219, B: 108, A: 255})
		for _, rect := range [...]image.Rectangle{image.Rect(a.X, a.Y, end.X, a.Y+1), image.Rect(a.X, end.Y-1, end.X, end.Y), image.Rect(a.X, a.Y, a.X+1, end.Y), image.Rect(end.X-1, a.Y, end.X, end.Y)} {
			draw.Draw(m.mapImage, rect, line, image.Point{}, draw.Src)
		}
	}
	if m.destination.name != "" {
		p := project(m.destination.x, m.destination.z)
		p = image.Pt(min(max(p.X, 3), 124), min(max(p.Y, 3), 124))
		draw.Draw(m.mapImage, image.Rect(p.X-2, p.Y-2, p.X+3, p.Y+3), image.NewUniform(color.RGBA{R: 75, G: 223, B: 244, A: 255}), image.Point{}, draw.Src)
	}
}

func (m *MapUI) showNavigationStatus() {
	if !m.Enabled() {
		return
	}
	m.mu.Lock()
	destination := m.destination
	hit := false
	if destination.gap != nil {
		_, hit = m.receivedColumns[*destination.gap]
	}
	if hit {
		m.destination = mapDestination{}
		m.SchedRedraw()
	}
	bounds, coverage, count := m.coverageBounds, m.coverageEnabled, m.coverageCount
	m.mu.Unlock()
	status := ""
	if destination.name != "" && !hit {
		player := m.w.session.PlayerSnapshot()
		dx, dz := destination.x-float64(player.Position.X()), destination.z-float64(player.Position.Z())
		angle := math.Atan2(-dx, dz)*180/math.Pi - float64(player.Yaw)
		if !math.IsNaN(angle) && !math.IsInf(angle, 0) {
			arrow := [...]string{"↑", "↗", "→", "↘", "↓", "↙", "←", "↖"}[(int(math.Round(math.Mod(angle, 360)/45))+8)%8]
			name := destination.name
			if destination.gap != nil {
				name = locale.Text(name)
			}
			status = locale.Format("%s · %.0f m %s", name, math.Hypot(dx, dz), arrow)
		}
	}
	if coverage {
		if status != "" {
			status += " · "
		}
		status += locale.Format("Region: %d/%d chunks", count, coverageArea(bounds))
	}
	if status != "" {
		m.w.session.SendActionBar(status)
	}
}

func (m *MapUI) navigationActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Enabled() && (m.coverageEnabled || m.destination.name != "")
}
