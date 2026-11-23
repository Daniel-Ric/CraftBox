//go:build gui

package desktop

import (
	"archive/zip"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/pkg/browser"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetStartupRecovery() (diagnostics.StartupRecovery, error) {
	store, err := diagnosticStore()
	if err != nil {
		return diagnostics.StartupRecovery{}, err
	}
	var currentID string
	if manager := diagnostics.Current(); manager != nil {
		currentID = manager.ID()
	}
	return store.StartupRecovery(currentID)
}

func (a *App) SetDiagnosticWorkspace(workspace string, ready bool) error {
	if manager := diagnostics.Current(); manager != nil {
		return manager.SetDesktopWorkspace(workspace, ready)
	}
	return nil
}

func (a *App) SetCrashNotice(ids []string, notice string) error {
	store, err := diagnosticStore()
	if err != nil {
		return err
	}
	return store.SetCrashNotice(ids, notice)
}

func (a *App) GetCrashReport(ids []string) ([]diagnostics.CrashReportSession, error) {
	store, err := diagnosticStore()
	if err != nil {
		return nil, err
	}
	return store.CrashReport(ids)
}

func (a *App) OpenCrashIssue(rawURL string) error {
	if len(rawURL) > 7600 {
		return errors.New("crash issue URL exceeds the supported length")
	}
	link, err := url.Parse(rawURL)
	if err != nil || link.Scheme != "https" || link.Host != "github.com" ||
		link.Path != "/Daniel-Ric/CraftBox/issues/new" || link.User != nil || link.Fragment != "" {
		return errors.New("invalid CraftBox crash issue URL")
	}
	fields, err := url.ParseQuery(link.RawQuery)
	if err != nil || len(fields) != 2 || len(fields["title"]) != 1 || len(fields["body"]) != 1 ||
		fields.Get("title") == "" || fields.Get("body") == "" {
		return errors.New("invalid CraftBox crash issue fields")
	}
	return browser.OpenURL(link.String())
}

func (a *App) ExportCrashReport(report string) (string, error) {
	if len(report) == 0 || len(report) > 128<<10 {
		return "", errors.New("invalid crash report size")
	}
	path, err := saveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title: locale.Format("Save crash report"), DefaultFilename: "CraftBox-crash-report.zip",
		Filters: []wruntime.FileFilter{{DisplayName: locale.Format("ZIP archive"), Pattern: "*.zip"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := writeCrashReport(path, report); err != nil {
		return "", err
	}
	return path, nil
}

func writeCrashReport(path, report string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".craftbox-crash-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	archive := zip.NewWriter(file)
	entry, err := archive.Create("crash-report.md")
	if err == nil {
		_, err = io.WriteString(entry, report)
	}
	if err = errors.Join(err, archive.Close(), file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
