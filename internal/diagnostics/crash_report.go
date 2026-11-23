package diagnostics

import (
	"encoding/json"
	"io"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type CrashReportSession struct {
	Role         string `json:"role"`
	Status       string `json:"status"`
	Version      string `json:"version"`
	Platform     string `json:"platform"`
	LastActivity string `json:"lastActivity"`
	Workspace    string `json:"workspace,omitempty"`
	Stack        string `json:"stack,omitempty"`
	Events       string `json:"events,omitempty"`
}

var (
	stackSymbol  = regexp.MustCompile(`^(?:github\.com/Daniel-Ric/CraftBox/|github\.com/sandertv/gophertunnel/|runtime\.)[a-zA-Z0-9_./*()\[\],-]+$`)
	stackSource  = regexp.MustCompile(`^([a-zA-Z0-9_.-]+\.go:[0-9]+)(?:\s|$)`)
	buildVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][a-zA-Z0-9.-]+)?$`)
)

func crashFrames(text string) string {
	var frames []string
	var function string
	for line := range strings.SplitSeq(text, "\n") {
		if function != "" && strings.HasPrefix(line, "\t") {
			source := path.Base(strings.ReplaceAll(strings.TrimSpace(line), `\`, "/"))
			if match := stackSource.FindStringSubmatch(source); match != nil {
				frames = append(frames, function+" ("+match[1]+")")
				if len(frames) == 32 {
					break
				}
			}
		}
		function = ""
		if i := strings.LastIndexByte(line, '('); i > 0 && i <= 180 && stackSymbol.MatchString(line[:i]) {
			function = strings.TrimPrefix(line[:i], "github.com/Daniel-Ric/CraftBox/")
		}
	}
	return strings.Join(frames, "\n")
}

func allowedDiagnosticEvent(name string) bool {
	switch name {
	case "session.started", "session.finished", "capture.started", "capture.finished",
		"window.started", "window.shutdown", "window.start.failed", "framework.fatal",
		"desktop.error", "frontend.error", "diagnostic.snapshot", "crash.recorder.failed",
		"crash.recorder.unavailable", "crash.recorder.pending", "output.redirect.failed":
		return true
	default:
		return false
	}
}

func safeDiagnosticEvents(text string, stack *strings.Builder) []string {
	var events []string
	for line := range strings.SplitSeq(text, "\n") {
		var entry struct {
			Time, Event string
			Fields      struct{ Stack string }
		}
		if json.Unmarshal([]byte(line), &entry) != nil || !allowedDiagnosticEvent(entry.Event) {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, entry.Time); err == nil {
			events = append(events, t.UTC().Format(time.RFC3339)+"  "+entry.Event)
		}
		if stack != nil && entry.Event == "diagnostic.snapshot" && stack.Len()+len(entry.Fields.Stack) <= 128*1024 {
			stack.WriteString(entry.Fields.Stack)
		}
	}
	return events
}

func (s *Store) CrashReport(ids []string) ([]CrashReportSession, error) {
	lock, err := s.storeLock()
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	sessions, err := s.reportSessions(ids)
	if err != nil {
		return nil, err
	}
	result := make([]CrashReportSession, 0, len(sessions))
	for _, session := range sessions {
		report := CrashReportSession{
			Role: session.Role, Status: session.Status, Version: "unknown",
			Platform: runtime.GOOS + "/" + runtime.GOARCH, LastActivity: session.Started,
		}
		if len(session.Version) <= 80 && buildVersion.MatchString(session.Version) {
			report.Version = session.Version
		}
		if validDesktopWorkspace(session.Workspace) {
			report.Workspace = session.Workspace
		}
		if session.Role == "map-window" {
			report.Workspace = "map"
		}
		root, err := s.root.OpenRoot(session.ID)
		if err != nil {
			return nil, err
		}
		var events []string
		var snapshot strings.Builder
		var snapshotTarget *strings.Builder
		if session.Fatal {
			snapshotTarget = &snapshot
		}
		for _, file := range session.Files {
			switch {
			case file.Name == "crash.log":
				if log, err := root.Open(file.Name); err == nil {
					if raw, err := io.ReadAll(io.LimitReader(log, 128*1024)); err == nil {
						report.Stack = crashFrames(string(raw))
					}
					log.Close()
				}
			case strings.HasSuffix(file.Name, ".jsonl"):
				if log, err := s.Read(session.ID, file.Name); err == nil {
					events = append(events, safeDiagnosticEvents(log.Text, snapshotTarget)...)
				}
			default:
				continue
			}
			if file.Modified > report.LastActivity {
				report.LastActivity = file.Modified
			}
		}
		root.Close()
		if report.Stack == "" {
			report.Stack = crashFrames(snapshot.String())
		}
		if when, err := time.Parse(time.RFC3339Nano, report.LastActivity); err == nil {
			report.LastActivity = when.UTC().Format(time.RFC3339)
		} else {
			report.LastActivity = "unknown"
		}
		report.Events = strings.Join(events[max(0, len(events)-20):], "\n")
		result = append(result, report)
	}
	return result, nil
}
