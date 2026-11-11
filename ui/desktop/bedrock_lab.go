package desktop

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/bedrocklab"
	"github.com/Daniel-Ric/CraftBox/utils"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) bedrockLab() *bedrocklab.Store { return bedrocklab.New(utils.PathData("bedrock-lab")) }
func (a *App) labContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
func (a *App) GetBedrockSnapshots() ([]bedrocklab.Snapshot, error) { return a.bedrockLab().List() }
func (a *App) GetBedrockSnapshot(id string) (bedrocklab.Snapshot, error) {
	return a.bedrockLab().Get(id)
}
func (a *App) QueryBedrockFiles(q bedrocklab.Query) (bedrocklab.Page, error) {
	ctx, cancel := context.WithTimeout(a.labContext(), 45*time.Second)
	defer cancel()
	return a.bedrockLab().Query(ctx, q)
}
func (a *App) PreviewBedrockFile(id, name string) (bedrocklab.Preview, error) {
	return a.bedrockLab().Preview(id, name)
}
func (a *App) ExportBedrockAudio(id, name string) (string, error) {
	p, err := a.bedrockLab().Preview(id, name)
	if err != nil {
		return "", err
	}
	prefix := "data:audio/wav;base64,"
	if !strings.HasPrefix(p.Audio, prefix) {
		return "", errors.New("this sound has no decoded WAV preview")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(p.Audio, prefix))
	if err != nil {
		return "", err
	}
	output, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Save decoded audio", DefaultFilename: utils.MakeValidFilename(strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))) + ".wav", Filters: []wruntime.FileFilter{{DisplayName: "Wave audio", Pattern: "*.wav"}}})
	if err != nil || output == "" {
		return "", err
	}
	return output, os.WriteFile(output, data, 0600)
}
func (a *App) AnalyzeBedrockJSON(before, after string) (bedrocklab.JSONAnalysis, error) {
	if len(before) > 4<<20 || len(after) > 4<<20 {
		return bedrocklab.JSONAnalysis{}, errors.New("JSON field analysis exceeds preview limit")
	}
	return bedrocklab.AnalyzeJSON([]byte(before), []byte(after))
}
func (a *App) CompareBedrockSnapshots(before, after string) ([]bedrocklab.Change, error) {
	return a.bedrockLab().Compare(before, after)
}
func (a *App) CompareBedrockVersions(before, after string) (bedrocklab.Comparison, error) {
	ctx, cancel := context.WithTimeout(a.labContext(), 90*time.Second)
	defer cancel()
	return a.bedrockLab().Comparison(ctx, before, after)
}
func (a *App) ClearBedrockAnalysis() { a.bedrockLab().ClearAnalysis() }
func (a *App) GetBedrockIdentifierReport(before, after string) (bedrocklab.IdentifierReport, error) {
	ctx, cancel := context.WithTimeout(a.labContext(), 90*time.Second)
	defer cancel()
	return a.bedrockLab().IdentifierReport(ctx, before, after)
}
func (a *App) RedecodeBedrockSnapshot(id, schemas string) error {
	return a.labJob(func(ctx context.Context, emit func(bedrocklab.Progress)) (string, error) {
		d, err := a.bedrockLab().Redecode(ctx, id, schemas, emit)
		return d.ID, err
	})
}
func (a *App) GetBedrockLabProgress() bedrocklab.Progress {
	a.labMu.Lock()
	defer a.labMu.Unlock()
	return a.labProgress
}
func (a *App) CancelBedrockLab() {
	a.labMu.Lock()
	defer a.labMu.Unlock()
	if a.labCancel != nil {
		a.labCancel()
	}
}
func (a *App) labJob(work func(context.Context, func(bedrocklab.Progress)) (string, error)) error {
	a.labMu.Lock()
	if a.labProgress.Running {
		a.labMu.Unlock()
		return errors.New("another Archive task is running")
	}
	ctx, cancel := context.WithCancel(a.labContext())
	a.labCancel = cancel
	a.labProgress = bedrocklab.Progress{Running: true, Phase: "Starting"}
	a.labMu.Unlock()
	go func() {
		defer cancel()
		emit := func(p bedrocklab.Progress) { a.labMu.Lock(); a.labProgress = p; a.labMu.Unlock() }
		id, err := work(ctx, emit)
		a.labMu.Lock()
		defer a.labMu.Unlock()
		a.labCancel = nil
		a.labProgress.Running = false
		a.labProgress.Snapshot = id
		a.labProgress.Phase = "Complete"
		if err != nil {
			a.labProgress.Error = err.Error()
			a.labProgress.Phase = "Failed"
			if errors.Is(err, context.Canceled) {
				a.labProgress.Phase = "Cancelled"
			}
		}
	}()
	return nil
}

type BedrockCaptureRequest struct {
	Installation string `json:"installation"`
	Source       string `json:"source"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Channel      string `json:"channel"`
	Schemas      string `json:"schemas"`
}

func (a *App) CaptureBedrockSnapshot(request BedrockCaptureRequest) error {
	return a.labJob(func(ctx context.Context, emit func(bedrocklab.Progress)) (string, error) {
		if request.Installation != "" {
			emit(bedrocklab.Progress{Running: true, Phase: "Detecting Minecraft"})
			installation, e := a.assetInstallation(request.Installation)
			if e != nil {
				return "", e
			}
			request.Source = filepath.Join(installation.Location, "data")
			if st, e := os.Stat(request.Source); e != nil || !st.IsDir() {
				return "", errors.New("Minecraft game data is not accessible; choose a copied data folder instead")
			}
			request.Channel = installation.Channel
			if request.Version == "" {
				request.Version = installation.GameVersion
			}
			if request.Name == "" {
				request.Name = installation.Name + " " + request.Version
			}
		}
		snapshot, e := a.bedrockLab().Capture(ctx, bedrocklab.CaptureRequest{Source: request.Source, Name: request.Name, Version: request.Version, Channel: request.Channel, Schemas: request.Schemas}, emit)
		if e != nil {
			return "", e
		}
		return snapshot.ID, nil
	})
}
func (a *App) SelectBedrockFolder(purpose string) (string, error) {
	title := "Choose Minecraft game data folder"
	if purpose == "schemas" {
		title = "Choose BDS schema folder"
	}
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: title})
}
func (a *App) DeleteBedrockSnapshot(id string) error {
	a.labMu.Lock()
	defer a.labMu.Unlock()
	if a.labProgress.Running {
		return errors.New("wait for the active task to finish")
	}
	return a.bedrockLab().Delete(id)
}
func (a *App) GetBedrockSchemas() ([]bedrocklab.SchemaBundle, error) {
	return a.bedrockLab().SchemaBundles()
}
func (a *App) GetBedrockServerReleases() ([]bedrocklab.ServerRelease, error) {
	return bedrocklab.LatestServerReleases(a.labContext())
}
func (a *App) PrepareBedrockSchemas(request bedrocklab.SchemaToolRequest) error {
	return a.labJob(func(ctx context.Context, emit func(bedrocklab.Progress)) (string, error) {
		_, e := a.bedrockLab().PrepareSchemas(ctx, request, emit)
		return "", e
	})
}
func (a *App) ValidateBedrockSnapshot(id string) (bedrocklab.Validation, error) {
	return a.bedrockLab().Validate(a.labContext(), id)
}
func (a *App) SearchBedrockExecutable(installation, query string) ([]bedrocklab.StringMatch, error) {
	target, e := a.assetInstallation(installation)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(a.labContext(), 45*time.Second)
	defer cancel()
	return bedrocklab.SearchStrings(ctx, filepath.Join(target.Location, "Minecraft.Windows.exe"), query, 4)
}
func (a *App) PrepareBedrockExecutable(installation string) (string, error) {
	target, e := a.assetInstallation(installation)
	if e != nil {
		return "", e
	}
	return bedrocklab.CopyReadableExecutable(a.labContext(), filepath.Join(target.Location, "Minecraft.Windows.exe"), filepath.Join(a.bedrockLab().Root, "executables"))
}
func (a *App) SelectBedrockExecutable() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Choose readable executable", Filters: []wruntime.FileFilter{{DisplayName: "Executable or binary", Pattern: "*.exe;*.dll;*.bin"}}})
}
func (a *App) SearchBedrockBinary(name, query string) ([]bedrocklab.StringMatch, error) {
	ctx, cancel := context.WithTimeout(a.labContext(), 45*time.Second)
	defer cancel()
	return bedrocklab.SearchStrings(ctx, name, query, 4)
}
func (a *App) ExportBedrockSnapshot(id string) (string, error) {
	d, e := a.bedrockLab().Get(id)
	if e != nil {
		return "", e
	}
	name, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export Bedrock snapshot", DefaultFilename: utils.MakeValidFilename(d.Name) + ".zip", Filters: []wruntime.FileFilter{{DisplayName: "Archive snapshot", Pattern: "*.zip"}}})
	if e != nil || name == "" {
		return "", e
	}
	tmp, e := os.CreateTemp(filepath.Dir(name), ".craftbox-bedrock-*")
	if e != nil {
		return "", e
	}
	defer os.Remove(tmp.Name())
	e = a.bedrockLab().Export(a.labContext(), id, tmp)
	closeErr := tmp.Close()
	if e != nil {
		return "", e
	}
	if closeErr != nil {
		return "", closeErr
	}
	if e = os.Rename(tmp.Name(), name); e != nil {
		return "", e
	}
	return name, nil
}
func (a *App) ImportBedrockPack(id, name string) ([]assetlibrary.Item, error) {
	temp, e := os.CreateTemp("", "craftbox-research-pack-*.mcpack")
	if e != nil {
		return nil, e
	}
	defer os.Remove(temp.Name())
	e = a.bedrockLab().ExportPack(a.labContext(), id, name, temp)
	closeErr := temp.Close()
	if e != nil {
		return nil, e
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return a.modelAssetsChanged(a.assets().Import(a.labContext(), "packs", []string{temp.Name()}, "Archive snapshot"))
}
