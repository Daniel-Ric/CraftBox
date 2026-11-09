package desktop

import (
	"encoding/json"
	"errors"
	"html"
	"io"
	"os"
	"path"
	"strings"
	"time"

	ce "github.com/Daniel-Ric/CraftBox/internal/commandexplorer"
	pi "github.com/Daniel-Ric/CraftBox/internal/packetinspector"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) commandExplorer() *ce.Store {
	a.packetMu.Lock()
	defer a.packetMu.Unlock()
	if a.commandArchive == nil {
		a.commandArchive = ce.New(utils.PathData("server-explorer"))
	}
	return a.commandArchive
}
func (a *App) GetCommandExplorerSessions() ([]ce.Session, error) { return a.commandExplorer().List() }
func (a *App) GetCommandExplorer(id string) (ce.Document, error) { return a.commandExplorer().Get(id) }
func (a *App) DeleteCommandExplorerSessions(ids []string) error {
	return a.commandExplorer().Delete(ids)
}

func (a *App) ExtractCommandExplorer(capture string) (ce.Session, error) {
	page, err := a.packetInspector().Store.Query(pi.Query{Capture: capture, Limit: 1})
	if err != nil {
		return ce.Session{}, err
	}
	m, err := a.commandExplorer().Begin(page.Capture.Name, capture, page.Capture.Protocol)
	if err != nil {
		return m, err
	}
	err = a.packetInspector().Store.Export(pi.Query{Capture: capture}, false, func(p pi.Packet) error {
		if a.ctx != nil {
			if err := a.ctx.Err(); err != nil {
				return err
			}
		}
		if p.Redacted || (p.Stage != "received" && p.Stage != "imported") {
			return nil
		}
		var pk packet.Packet
		if len(p.Raw) > 0 {
			pk, _ = pi.Decode(p.PacketID, p.Raw, p.ShieldID)
		}
		if pk == nil {
			pk, _ = pi.DecodeJSON(p.PacketID, p.Payload)
		}
		if pk == nil {
			return nil
		}
		return a.commandExplorer().ObserveEvidence(m.ID, pk, p.Direction == "serverbound", time.UnixMilli(p.Time), p.ID)
	})
	endErr := a.commandExplorer().End(m.ID)
	if err != nil {
		_ = a.commandExplorer().Delete([]string{m.ID})
		return m, err
	}
	if endErr != nil {
		return m, endErr
	}
	d, err := a.commandExplorer().Get(m.ID)
	return d.Session, err
}

type CommandExplorerExport struct {
	Session string   `json:"session"`
	Format  string   `json:"format"`
	IDs     []string `json:"ids"`
}

func exportCommandExplorer(d ce.Document, request CommandExplorerExport) ([]byte, error) {
	selected := map[string]bool{}
	for _, name := range request.IDs {
		selected[name] = true
	}
	commands := []ce.Command{}
	for _, command := range d.Commands {
		if len(selected) == 0 || selected[command.Name] {
			commands = append(commands, command)
		}
	}
	d.Commands = commands
	events := []ce.Event{}
	for _, event := range d.Events {
		switch event.Kind {
		case "commands":
			var snapshot struct {
				Commands []ce.Command `json:"commands"`
			}
			if json.Unmarshal(event.Data, &snapshot) != nil {
				continue
			}
			filtered := []ce.Command{}
			for _, command := range snapshot.Commands {
				if len(selected) == 0 || selected[command.Name] {
					filtered = append(filtered, command)
				}
			}
			snapshot.Commands = filtered
			event.Data, _ = json.Marshal(snapshot)
			events = append(events, event)
		case "enum":
			if len(selected) == 0 {
				events = append(events, event)
			}
		case "command", "output":
			if len(selected) == 0 {
				events = append(events, event)
			}
		}
	}
	data, _ := json.Marshal(map[string]any{"commands": commands})
	if len(events) >= 30000 {
		events = events[len(events)-29999:]
	}
	events = append(events, ce.Event{ID: "current-definitions", Kind: "commands", Direction: "clientbound", Time: d.Session.Updated, Data: data})
	d.Events = events
	d.Format = "craftbox-command-archive"
	d.Session.Commands, d.Session.Events = len(commands), len(events)
	if request.Format == "json" {
		return json.MarshalIndent(d, "", "  ")
	}
	var output strings.Builder
	if request.Format == "html" {
		output.WriteString("<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width\"><title>CraftBox command reference</title><style>body{font:15px system-ui;max-width:1000px;margin:40px auto;padding:0 24px}article{border-bottom:1px solid #8885;padding:16px 0}pre{white-space:pre-wrap;overflow-wrap:anywhere}</style><h1>" + html.EscapeString(d.Session.Name) + "</h1>")
		for _, c := range commands {
			output.WriteString("<article><h2>/" + html.EscapeString(c.Name) + "</h2><p>" + html.EscapeString(c.Description) + "</p>")
			raw, _ := json.MarshalIndent(c, "", "  ")
			output.WriteString("<pre>" + html.EscapeString(string(raw)) + "</pre></article>")
		}
		output.WriteString("</html>")
	} else if request.Format == "markdown" {
		output.WriteString("# " + strings.ReplaceAll(d.Session.Name, "\n", " ") + "\n\n")
		for _, c := range commands {
			output.WriteString("## /" + c.Name + "\n\n" + c.Description + "\n\n")
			for _, overload := range c.Overloads {
				output.WriteString("    /" + c.Name)
				for _, p := range overload {
					left, right := "<", ">"
					if p.Optional {
						left, right = "[", "]"
					}
					output.WriteString(" " + left + p.Name + right)
				}
				output.WriteString("\n")
			}
			output.WriteString("\n")
		}
	} else {
		return nil, errors.New("Choose JSON, Markdown or HTML")
	}
	return []byte(output.String()), nil
}

func (a *App) ExportCommandExplorer(request CommandExplorerExport) (string, error) {
	d, err := a.commandExplorer().Get(request.Session)
	if err != nil {
		return "", err
	}
	raw, err := exportCommandExplorer(d, request)
	if err != nil {
		return "", err
	}
	ext := map[string]string{"json": "json", "markdown": "md", "html": "html"}[request.Format]
	name, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export command reference", DefaultFilename: "command-reference." + ext, Filters: []wruntime.FileFilter{{DisplayName: strings.ToUpper(ext), Pattern: "*." + ext}}})
	if err != nil || name == "" {
		return "", err
	}
	if !strings.EqualFold(path.Ext(name), "."+ext) {
		name += "." + ext
	}
	return name, writeExportFile(name, raw)
}

func (a *App) ImportCommandExplorer() (ce.Session, error) {
	name, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Import command archive", Filters: []wruntime.FileFilter{{DisplayName: "CraftBox command archive", Pattern: "*.json"}}})
	if err != nil || name == "" {
		return ce.Session{}, err
	}
	file, err := os.Open(name)
	if err != nil {
		return ce.Session{}, err
	}
	defer file.Close()
	var document ce.Document
	if err = json.NewDecoder(io.LimitReader(file, 128<<20)).Decode(&document); err != nil {
		return ce.Session{}, err
	}
	return a.commandExplorer().Import(document)
}
