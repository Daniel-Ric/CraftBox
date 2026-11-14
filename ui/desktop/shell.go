//go:build gui && !android

package desktop

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/assets"
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/updater"
	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var frontend embed.FS

type GUI struct{ app *App }

func restoreBundledAssets() {
	if exe, err := os.Executable(); err == nil {
		diagnostics.Record("info", "assets", "assets.restore.started", "Checking bundled installation assets", nil)
		if err = assets.Restore(filepath.Join(filepath.Dir(exe), "assets", "vanilla-3d")); err != nil {
			diagnostics.Record("warning", "assets", "assets.restore.failed", "Using embedded assets; installation files could not be restored", map[string]any{"error": err.Error()})
		} else {
			diagnostics.Record("info", "assets", "assets.restored", "Bundled assets checked and missing files restored", nil)
		}
	}
}

func (g *GUI) Init() error {
	g.app = NewApp()
	if account := auth.Auth.Account(); account != nil {
		g.app.state.Account = account.Name()
	}
	messages.SetEventHandler(g.app.HandleEvent)
	auth.Auth.SetHandler(&messages.AuthHandler{})
	utils.ErrorHandler = g.app.reportError
	return nil
}
func (g *GUI) Start(ctx context.Context, cancel context.CancelCauseFunc) error {
	assets, err := fs.Sub(frontend, "frontend/dist")
	if err != nil {
		return err
	}
	hook := &desktopLog{app: g.app}
	return wails.Run(&options.App{
		Logger: diagnosticFrameworkLogger{},
		Title:  "CraftBox", Width: 1240, Height: 820, MinWidth: 760, MinHeight: 600,
		BackgroundColour: &options.RGBA{R: 28, G: 34, B: 40, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets}, Bind: []interface{}{g.app},
		OnStartup: func(wctx context.Context) {
			diagnostics.Record("info", "desktop", "window.started", "Main workspace opened", nil)

			go restoreBundledAssets()
			g.app.mu.Lock()
			g.app.ctx = wctx
			g.app.emit = func(name string, data any) { wruntime.EventsEmit(wctx, name, data) }
			g.app.mu.Unlock()
			logrus.AddHook(hook)
			go func() {
				select {
				case <-ctx.Done():
					g.app.StopCommand()
					wruntime.Quit(wctx)
				case <-wctx.Done():
				}
			}()
		},
		OnBeforeClose: func(wctx context.Context) bool {
			diagnostics.Record("info", "desktop", "window.close.requested", "Window close requested", nil)
			return g.app.shouldPreventWindowClose()
		},
		OnShutdown: func(context.Context) {
			g.app.closeDiscordPresence()
			diagnostics.Record("info", "desktop", "window.shutdown", "Main workspace shutdown", nil)
			g.app.closeTerminals()
			g.app.CancelGLBExport()
			g.app.CancelWorldExport()
			g.app.mu.Lock()
			g.app.emit = nil
			loginDone := g.app.loginDone
			g.app.mu.Unlock()
			g.app.CancelLogin()
			g.app.StopCommand()
			g.app.mu.Lock()
			done := g.app.done
			g.app.mu.Unlock()
			if done != nil {
				<-done
			}
			g.app.flushAssets()
			g.app.closePacketInspector()
			g.app.closeMarketplace()
			if loginDone != nil {
				<-loginDone
			}
			cancel(context.Canceled)
			g.app.closeMapWindowHost()
			g.app.closeScenes()
			g.app.CancelRenderImport()
			g.app.CloseRenderWorld("")
		},
	})
}
func (a *App) PickFile() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Select file"})
}

func (a *App) OpenRenderWorld(kind string) (RenderWorld, error) {
	return a.OpenRenderWorldForTab("legacy", kind)
}
func (a *App) OpenRenderWorldForTab(tabID, kind string) (RenderWorld, error) {
	var path string
	var err error
	switch kind {
	case "folder":
		path, err = wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open Bedrock world folder"})
	case "archive":
		path, err = wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Open Minecraft world", Filters: []wruntime.FileFilter{{DisplayName: "Bedrock world", Pattern: "*.mcworld;*.zip"}}})
	default:
		return RenderWorld{}, fmt.Errorf("Invalid file selection")
	}
	if err != nil || path == "" {
		return RenderWorld{}, err
	}
	return a.importRenderWorldForTab(a.ctx, tabID, path, nil)
}

func (a *App) SaveRenderPNG(name, data string) (string, error) {
	raw, err := decodeRenderPNG(data)
	if err != nil {
		return "", err
	}
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune("<>:\"/\\|?*", r) {
			return '_'
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "Map render"
	}
	name = strings.TrimSuffix(name, filepath.Ext(name)) + ".png"
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Save map render", DefaultFilename: name, Filters: []wruntime.FileFilter{{DisplayName: "PNG image", Pattern: "*.png"}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".png") {
		path += ".png"
	}
	if err = os.WriteFile(path, raw, 0644); err != nil {
		return "", err
	}
	return path, nil
}
func (a *App) SaveCinematicFile(name, kind, data string) (string, error) {
	raw, err := decodeCinematicFile(kind, data)
	if err != nil {
		return "", err
	}
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune("<>:\"/\\|?*", r) {
			return '_'
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "Camera sequence"
	}
	name += "." + kind
	title := "Save cinematic"
	if kind == "json" && strings.HasSuffix(strings.TrimSuffix(name, ".json"), ".render-style") {
		title = "Save render preset"
	}
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: title, DefaultFilename: name, Filters: []wruntime.FileFilter{{DisplayName: strings.ToUpper(kind), Pattern: "*." + kind}}})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), "."+kind) {
		path += "." + kind
	}
	if err = writeExportFile(path, raw); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) OpenURL(value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return err
	}
	if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "minecraft" {
		return errors.New("This address cannot be opened")
	}
	if u.Scheme == "minecraft" {
		return openMinecraftURL(a.ctx, u.String())
	}
	wruntime.BrowserOpenURL(a.ctx, u.String())
	return nil
}
func (a *App) OpenMinecraft() error {
	link, err := minecraftJoinForState(a.Snapshot())
	if err != nil {
		return err
	}
	return a.OpenURL(link)
}
func (a *App) OpenResultFolder(value string) error {
	absolute, err := resultFolder(value)
	if err != nil {
		return err
	}
	return openResultDirectory(absolute)
}
func (a *App) CheckUpdate() (string, error) {
	result, err := a.CheckUpdateDetails("stable")
	if err != nil {
		return "", err
	}
	if result.Release == nil {
		return "", nil
	}
	return result.Release.Version, nil
}
func (a *App) CheckUpdateDetails(channel string) (*updater.ReleaseCheck, error) {
	a.mu.Lock()
	a.updateCheckSequence++
	sequence := a.updateCheckSequence
	a.pendingUpdate = nil
	a.mu.Unlock()
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := updater.CheckRelease(ctx, channel, utils.Version)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if sequence != a.updateCheckSequence {
		a.mu.Unlock()
		return result, nil
	}
	a.pendingUpdate = nil
	a.state.UpdateVersion = ""
	if result.Available && result.Release != nil {
		a.state.UpdateVersion = result.Release.Version
		if result.Release.Installable {
			a.pendingUpdate = result.Release
		}
	}
	a.mu.Unlock()
	a.publish()
	return result, nil
}
func (a *App) InstallUpdate() error {
	if a.Snapshot().Running {
		return errors.New("End the session first")
	}
	a.mu.Lock()
	update := a.pendingUpdate
	a.mu.Unlock()
	if update == nil {
		return errors.New("Check for a compatible update before installing")
	}
	if err := updater.DoUpdate(update); err != nil {
		return err
	}
	if err := updater.Restart(); err != nil {
		return err
	}
	wruntime.Quit(a.ctx)
	return nil
}

type desktopLog struct {
	app *App
}

func (*desktopLog) Levels() []logrus.Level { return logrus.AllLevels }
func (h *desktopLog) Fire(entry *logrus.Entry) error {
	value := LogEntry{Time: entry.Time.Format(time.RFC3339), Level: entry.Level.String(), Message: entry.Message}
	if part, ok := entry.Data["part"]; ok {
		value.Source = fmt.Sprint(part)
	}
	h.app.appendLog(value)
	return nil
}
