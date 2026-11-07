//go:build gui && !android

package desktop

import (
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"os"
)

type diagnosticFrameworkLogger struct{}

func (diagnosticFrameworkLogger) Print(s string) {
	diagnostics.Record("info", "webview", "framework.log", s, nil)
}
func (diagnosticFrameworkLogger) Trace(s string) {
	diagnostics.Record("trace", "webview", "framework.log", s, nil)
}
func (diagnosticFrameworkLogger) Debug(s string) {
	diagnostics.Record("debug", "webview", "framework.log", s, nil)
}
func (diagnosticFrameworkLogger) Info(s string) {
	diagnostics.Record("info", "webview", "framework.log", s, nil)
}
func (diagnosticFrameworkLogger) Warning(s string) {
	diagnostics.Record("warning", "webview", "framework.log", s, nil)
}
func (diagnosticFrameworkLogger) Error(s string) {
	diagnostics.Record("error", "webview", "framework.log", s, nil)
}
func (diagnosticFrameworkLogger) Fatal(s string) {
	diagnostics.Record("fatal", "webview", "framework.fatal", s, nil)
	if m := diagnostics.Current(); m != nil {
		m.Snapshot("WebView host fatal error: " + s)
	}
	os.Exit(1)
}
