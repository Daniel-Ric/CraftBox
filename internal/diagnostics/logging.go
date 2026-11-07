package diagnostics

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
	"sync"
)

type Hook struct{ Manager *Manager }

func (*Hook) Levels() []logrus.Level { return logrus.AllLevels }
func (h *Hook) Fire(e *logrus.Entry) error {
	fields := map[string]any{}
	for k, v := range e.Data {
		fields[k] = v
	}
	component := "application"
	if part, ok := e.Data["part"]; ok {
		component = fmt.Sprint(part)
	}
	if e.Caller != nil {
		fields["caller"] = fmt.Sprintf("%s:%d", filepath.Base(e.Caller.File), e.Caller.Line)
		fields["function"] = e.Caller.Function
	}
	message := e.Message
	if strings.HasPrefix(strings.ToLower(message), "payload:") {
		message = "[packet payload omitted from persistent diagnostics]"
	}
	h.Manager.Record(e.Level.String(), component, "log", message, fields)
	return nil
}

type LineWriter struct {
	mu      sync.Mutex
	manager *Manager
	buffer  []byte
}

func (m *Manager) OutputWriter() *LineWriter { return &LineWriter{manager: m} }
func (w *LineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	length := len(p)
	for len(p) > 0 {
		n := min(len(p), 16384-len(w.buffer))
		w.buffer = append(w.buffer, p[:n]...)
		p = p[n:]
		for {
			index := bytes.IndexByte(w.buffer, '\n')
			if index < 0 {
				break
			}
			w.emit(w.buffer[:index], false)
			w.buffer = append(w.buffer[:0], w.buffer[index+1:]...)
		}
		if len(w.buffer) >= 16384 {
			w.emit(w.buffer, true)
			w.buffer = w.buffer[:0]
		}
	}
	return length, nil
}
func (w *LineWriter) emit(line []byte, fragment bool) {
	text := strings.TrimRight(string(line), "\r")
	if text != "" {
		w.manager.Record("info", "process", "process.output", text, map[string]any{"fragment": fragment})
	}
}
func (w *LineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buffer) > 0 {
		w.emit(w.buffer, false)
		w.buffer = nil
	}
}
