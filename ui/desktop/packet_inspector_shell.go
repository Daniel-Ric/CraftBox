//go:build gui && !android

package desktop

import (
	pi "github.com/Daniel-Ric/CraftBox/internal/packetinspector"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ImportPacketCapture(path string) (pi.Capture, error) {
	if path == "" {
		var e error
		path, e = wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Import packet capture", Filters: []wruntime.FileFilter{{DisplayName: "Packet captures", Pattern: "*.json;*.pcap2;*.log;*.txt"}}})
		if e != nil || path == "" {
			return pi.Capture{}, e
		}
	}
	return a.packetInspector().Store.Import(path)
}
func (a *App) ExportPacketCapture(q pi.Query, revealSecrets bool) (string, error) {
	path, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export packet capture", DefaultFilename: "capture.json", Filters: []wruntime.FileFilter{{DisplayName: "Packet capture JSON", Pattern: "*.json"}}})
	if e != nil || path == "" {
		return "", e
	}
	e = a.writePacketExport(path, q, revealSecrets)
	return path, e
}
