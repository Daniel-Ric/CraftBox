//go:build gui && !android

package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MapWindowApp struct {
	mu              sync.Mutex
	refreshMu       sync.Mutex
	ctx             context.Context
	address, token  string
	client          *http.Client
	state           State
	tiles           map[string]Tile
	sequence, focus uint64
	theme           string
}

func (a *MapWindowApp) refresh() error {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.mu.Lock()
	after := a.sequence
	a.mu.Unlock()
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, a.address+"?after="+strconv.FormatUint(after, 10), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	response, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("map bridge status %d", response.StatusCode)
	}
	var update MapWindowSnapshot
	if err := json.NewDecoder(response.Body).Decode(&update); err != nil {
		return err
	}
	a.mu.Lock()
	oldFocus, oldTheme := a.focus, a.theme
	if update.Reset {
		a.tiles = make(map[string]Tile)
	}
	for _, tile := range update.Tiles {
		a.tiles[tileVersionKey(tile)] = tile
	}
	a.state = update.State
	a.sequence = update.Sequence
	a.focus = update.Focus
	a.theme = update.Theme
	a.mu.Unlock()
	if update.Reset {
		wruntime.EventsEmit(a.ctx, "desktop:reset-map", nil)
	}
	if len(update.Tiles) > 0 {
		wruntime.EventsEmit(a.ctx, "desktop:tiles", update.Tiles)
	}
	wruntime.EventsEmit(a.ctx, "desktop:state", update.State)
	if update.Theme != oldTheme {
		wruntime.EventsEmit(a.ctx, "desktop:theme", update.Theme)
	}
	if update.Focus != oldFocus {
		wruntime.WindowUnminimise(a.ctx)
		wruntime.WindowShow(a.ctx)
	}
	return nil
}

func (a *MapWindowApp) GetBootstrap() (Bootstrap, error) {
	if err := a.refresh(); err != nil {
		return Bootstrap{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return Bootstrap{SceneBudgetBytes: availableSceneBudget(), MapWindow: true, Theme: a.theme, Version: utils.Version, Minecraft: protocol.CurrentVersion, State: a.state, Commands: []CommandInfo{}}, nil
}

func (a *MapWindowApp) CreateWorldRegion(request messages.CreateWorldRegion) (messages.WorldRegion, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return messages.WorldRegion{}, err
	}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodPost, strings.TrimSuffix(a.address, "/map-window")+"/world-region", bytes.NewReader(data))
	if err != nil {
		return messages.WorldRegion{}, err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	response, err := a.client.Do(req)
	if err != nil {
		return messages.WorldRegion{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return messages.WorldRegion{}, fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}
	var region messages.WorldRegion
	if err := json.NewDecoder(response.Body).Decode(&region); err != nil {
		return region, err
	}
	return region, a.refresh()
}
func (a *MapWindowApp) EditWorld(request messages.EditWorld) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(a.address, "/map-window")+"/world-metadata", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}
	return a.refresh()
}
func (a *MapWindowApp) Snapshot() State { a.mu.Lock(); defer a.mu.Unlock(); return a.state }

func (a *MapWindowApp) sceneRequest(method, endpoint string, payload any, result any) error {
	var body io.Reader
	if payload != nil {
		data, e := json.Marshal(payload)
		if e != nil {
			return e
		}
		body = bytes.NewReader(data)
	}
	timeout := 120 * time.Second
	if strings.HasPrefix(endpoint, "/composition-") || endpoint == "/scene-export" {
		timeout = 10 * time.Minute
	}
	if endpoint == "/scene-page" {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(a.address, "/map-window")+endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent && result == nil {
		return nil
	}
	if response.StatusCode != 200 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("%s", strings.TrimSpace(string(data)))
	}
	return json.NewDecoder(response.Body).Decode(result)
}
func (a *MapWindowApp) GetScene(id string, after uint64) (scene.Snapshot, error) {
	var result scene.Snapshot
	err := a.sceneRequest("GET", "/scene?worldId="+url.QueryEscape(id)+"&after="+strconv.FormatUint(after, 10), nil, &result)
	return result, err
}

func (a *MapWindowApp) GetScenePage(id string, request scene.PageRequest) (scene.Page, error) {
	var result scene.Page
	err := a.sceneRequest("POST", "/scene-page", map[string]any{"worldId": id, "page": request}, &result)
	return result, err
}

func (a *MapWindowApp) GetMapPage(id string, request scene.MapRequest) (scene.MapPage, error) {
	var result scene.MapPage
	err := a.sceneRequest("POST", "/map-page", map[string]any{"worldId": id, "page": request}, &result)
	return result, err
}

func (a *MapWindowApp) PrepareWorldComposition(request WorldCompositionRequest) (WorldCompositionPreview, error) {
	var result WorldCompositionPreview
	err := a.sceneRequest("POST", "/composition-prepare", map[string]any{"request": request}, &result)
	return result, err
}
func (a *MapWindowApp) SaveWorldComposition(id string, allowConflicts bool) (SavedWorld, error) {
	var result SavedWorld
	err := a.sceneRequest("POST", "/composition-save", map[string]any{"id": id, "allowConflicts": allowConflicts}, &result)
	if err == nil {
		err = a.refresh()
	}
	return result, err
}
func (a *MapWindowApp) CancelComposition() {
	var result CompositionProgress
	_ = a.sceneRequest("POST", "/composition-cancel", map[string]any{}, &result)
}
func (a *MapWindowApp) DiscardComposition(id string) {
	var result bool
	_ = a.sceneRequest("POST", "/composition-discard", map[string]any{"id": id}, &result)
}
func (a *MapWindowApp) GetCompositionProgress() CompositionProgress {
	var result CompositionProgress
	_ = a.sceneRequest("POST", "/composition-progress", map[string]any{}, &result)
	return result
}
func (a *MapWindowApp) ExportScene(id string) (string, error) {
	var result struct {
		Path string `json:"path"`
	}
	err := a.sceneRequest("POST", "/scene-export", map[string]string{"worldId": id}, &result)
	return result.Path, err
}
func (a *MapWindowApp) ExportModel(request ModelExportRequest) (string, error) {
	var result struct {
		Path string `json:"path"`
	}
	err := a.sceneRequest("POST", "/scene-export", request, &result)
	return result.Path, err
}
func (a *MapWindowApp) GetModelExportProgress() ModelExportStatus {
	var result ModelExportStatus
	_ = a.sceneRequest("GET", "/scene-export", nil, &result)
	return result
}
func (a *MapWindowApp) CancelGLBExport() { _ = a.sceneRequest("DELETE", "/scene-export", nil, nil) }
func (a *MapWindowApp) CancelModelExportJob(jobID string) {
	_ = a.sceneRequest("DELETE", "/scene-export?jobId="+url.QueryEscape(jobID), nil, nil)
}
func (a *MapWindowApp) GetMapTiles() []Tile {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Tile, 0, len(a.tiles))
	for _, tile := range a.tiles {
		out = append(out, tile)
	}
	return out
}
func (*MapWindowApp) GetLogs() []LogEntry { return []LogEntry{} }
func (a *MapWindowApp) OpenResultFolder(path string) error {
	return (&App{ctx: a.ctx}).OpenResultFolder(path)
}

func (a *MapWindowApp) ReportFrontendError(message, stack string) error {
	return (&App{}).ReportFrontendError(message, stack)
}

func RunMapWindow() error {
	address, token := os.Getenv("CRAFT_BOX_MAP_BRIDGE"), os.Getenv("CRAFT_BOX_MAP_TOKEN")
	if address == "" || token == "" {
		return fmt.Errorf("missing map window bridge")
	}
	assets, err := fs.Sub(frontend, "frontend/dist")
	if err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	app := &MapWindowApp{address: address, token: token, client: &http.Client{Timeout: 2 * time.Second}, tiles: make(map[string]Tile)}
	var cancel context.CancelFunc
	return wails.Run(&options.App{
		Logger: diagnosticFrameworkLogger{},
		Title:  "CraftBox · World maps", Width: 1280, Height: 760, MinWidth: 720, MinHeight: 420,
		BackgroundColour: &options.RGBA{R: 28, G: 34, B: 40, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets}, Bind: []interface{}{app},
		Windows: &windows.Options{WindowClassName: "CraftBoxWorldMaps", WebviewUserDataPath: filepath.Join(cache, "CraftBox", "map-webview")},
		OnStartup: func(ctx context.Context) {
			app.ctx, cancel = context.WithCancel(ctx)
			go func() {
				ticker := time.NewTicker(200 * time.Millisecond)
				defer ticker.Stop()
				failures := 0
				for {
					select {
					case <-app.ctx.Done():
						return
					case <-ticker.C:
						if err := app.refresh(); err != nil {
							failures++
							if failures >= 3 {
								wruntime.Quit(ctx)
								return
							}
						} else {
							failures = 0
						}
					}
				}
			}()
		},
		OnShutdown: func(context.Context) {
			if cancel != nil {
				cancel()
			}
		},
	})
}
