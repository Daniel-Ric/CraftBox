//go:build gui

package desktop

import (
	"errors"
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/sirupsen/logrus"
)

func diagnosticManager() (*diagnostics.Manager, error) {
	m := diagnostics.Current()
	if m == nil {
		return nil, errors.New("Persistent logging is not available in this process")
	}
	return m, nil
}
func diagnosticStore() (*diagnostics.Store, error) {
	s := diagnostics.AvailableStore()
	if s == nil {
		return nil, errors.New("Persistent log storage is not available")
	}
	return s, nil
}
func (a *App) GetLogStorage() (diagnostics.Status, error) {
	store, e := diagnosticStore()
	if e != nil {
		return diagnostics.Status{}, e
	}
	return store.Status()
}
func (a *App) ReadSavedLog(id, name string) (diagnostics.Preview, error) {
	store, e := diagnosticStore()
	if e != nil {
		return diagnostics.Preview{}, e
	}
	return store.Read(id, name)
}
func (a *App) DeleteLogSession(id string) error {
	store, e := diagnosticStore()
	if e != nil {
		return e
	}
	return store.Delete(id)
}
func (a *App) DeleteClosedLogSessions() error {
	store, e := diagnosticStore()
	if e != nil {
		return e
	}
	return store.DeleteClosed()
}
func (a *App) CleanupSavedLogs() error {
	store, e := diagnosticStore()
	if e != nil {
		return e
	}
	return store.Cleanup()
}
func (a *App) SetLogPolicy(p diagnostics.Policy) error {
	store, e := diagnosticStore()
	if e != nil {
		return e
	}
	if e := store.SetPolicy(p); e != nil {
		return e
	}
	logrus.SetLevel(logrus.DebugLevel)
	if p.Level == "trace" {
		logrus.SetLevel(logrus.TraceLevel)
	}
	diagnostics.Record("info", "diagnostics", "policy.updated", "Log storage policy updated", nil)
	return nil
}
func (a *App) CreateDiagnosticSnapshot() error {
	m, e := diagnosticManager()
	if e != nil {
		return e
	}
	m.Snapshot("User requested a diagnostic snapshot")
	return nil
}
func (a *App) OpenSavedLog(id, name string) error {
	store, e := diagnosticStore()
	if e != nil {
		return e
	}
	path, e := store.FilePath(id, name)
	if e != nil {
		return e
	}
	return diagnostics.OpenPath(path, false)
}
func (a *App) OpenLogFolder() error {
	store, e := diagnosticStore()
	if e != nil {
		return e
	}
	return diagnostics.OpenPath(store.Path(), true)
}

func (a *App) OpenSessionLog(id string) error {
	store, err := diagnosticStore()
	if err != nil {
		return err
	}
	path, err := store.SessionLogSnapshot(id)
	if err != nil {
		return err
	}
	return diagnostics.OpenPath(path, false)
}

func (a *App) ReportFrontendError(message, stack string) error {
	m, e := diagnosticManager()
	if e != nil {
		return e
	}
	if len(message) > 4096 {
		message = message[:4096]
	}
	if len(stack) > 8192 {
		stack = stack[:8192]
	}
	m.Record("error", "frontend", "frontend.error", message, map[string]any{"stack": stack})
	return nil
}
