package main

import (
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sirupsen/logrus"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type consoleWriter struct{ w io.Writer }

func (w consoleWriter) Write(b []byte) (int, error) { _, _ = w.w.Write(b); return len(b), nil }

var finishLogging = func() {}

func setupLogging(isDebug bool) {
	store, err := diagnostics.OpenStore(utils.PathData("logs"))
	if err != nil {
		fallback, e := os.UserCacheDir()
		if e == nil {
			store, err = diagnostics.OpenStore(filepath.Join(fallback, "CraftBox", "logs"))
		}
	}
	if err != nil {
		logrus.WithError(err).Error("Persistent diagnostics unavailable")
		return
	}
	diagnostics.SetStore(store)
	role := "main"
	if os.Getenv("CRAFT_BOX_MAP_BRIDGE") != "" {
		role = "map-window"
	}
	manager, err := store.Start(diagnostics.Options{Version: utils.Version, Role: role})
	if err != nil {
		store.ReportError(err)
		finishLogging = func() { store.Close(); diagnostics.SetStore(nil) }
		logrus.WithError(err).Error("Could not start diagnostic session")
		return
	}
	diagnostics.SetCurrent(manager)
	slog.SetDefault(slog.New(diagnostics.NewSlogHandler()))
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	logrus.SetOutput(consoleWriter{originalStdout})
	logrus.SetReportCaller(true)
	logrus.AddHook(&diagnostics.Hook{Manager: manager})
	logrus.SetLevel(logrus.DebugLevel)
	if store.Policy().Level == "trace" {
		logrus.SetLevel(logrus.TraceLevel)
	}
	if isDebug {
		logrus.SetLevel(logrus.TraceLevel)
		p := store.Policy()
		p.Level = "trace"
		_ = store.SetPolicy(p)
	}

	if info, e := os.Lstat(utils.PathData("CraftBox.log")); e == nil && info.Mode().IsRegular() {
		if legacy, e := os.Open(utils.PathData("CraftBox.log")); e == nil {
			prefix := make([]byte, 32)
			_, _ = legacy.ReadAt(prefix, 0)
			if !strings.HasPrefix(string(prefix), "Persistent logs are stored in ") {
				manager.Record("info", "diagnostics", "legacy.import", "Preserving the last 128 KiB of the previous legacy log", map[string]any{"originalBytes": info.Size()})
				output := manager.OutputWriter()
				_, _ = io.Copy(output, io.NewSectionReader(legacy, max(int64(0), info.Size()-128*1024), 128*1024))
				output.Flush()
			}
			legacy.Close()
		}
	}

	_ = os.WriteFile(utils.PathData("CraftBox.log"), []byte("Persistent logs are stored in "+filepath.Join(store.Path(), manager.ID())+"\nOpen Preferences > Diagnostics & logs to view and manage them.\n"), 0600)
	stopCrash, err := startCrashRecorder(manager)
	if err != nil {
		manager.Record("error", "diagnostics", "crash.recorder.unavailable", err.Error(), nil)
		stopCrash = func() {}
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		manager.Record("error", "diagnostics", "output.redirect.failed", err.Error(), nil)
		finishLogging = func() { stopCrash(); manager.Close(); store.Close(); diagnostics.SetCurrent(nil) }
		return
	}
	lines := manager.OutputWriter()
	copied := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.MultiWriter(lines, consoleWriter{originalStdout}), reader)
		lines.Flush()
		reader.Close()
		close(copied)
	}()
	os.Stdout = writer
	redirectStderr(writer)
	finishLogging = func() {
		stopCrash()
		os.Stdout = originalStdout
		redirectStderr(originalStderr)
		writer.Close()
		select {
		case <-copied:
		case <-time.After(time.Second):
			manager.Record("warning", "diagnostics", "output.pending", "Output is still being flushed", nil)
		}
		manager.Close()
		store.Close()
		diagnostics.SetCurrent(nil)
		diagnostics.SetStore(nil)
	}
	status, e := store.Status()
	if e == nil {
		for _, s := range status.Sessions {
			if !s.Active && !s.Clean {
				manager.Record("warning", "diagnostics", "previous.interrupted", "Previous session did not exit normally", map[string]any{"previousSession": s.ID, "status": s.Status})
			}
		}
	}
}

type logTransport struct{ rt http.RoundTripper }

func (t *logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	logrus.Tracef("Request %s", diagnostics.Redact(req.URL.String()))
	return t.rt.RoundTrip(req)
}
