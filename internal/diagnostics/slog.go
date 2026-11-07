package diagnostics

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"
)

type scopedAttribute struct {
	groups    []string
	attribute slog.Attr
}
type slogHandler struct {
	groups     []string
	attributes []scopedAttribute
}

func NewSlogHandler() slog.Handler { return &slogHandler{} }

func (*slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	manager := Current()
	if manager == nil {
		return false
	}
	return level >= slog.LevelInfo || manager.store.Policy().Level != "info"
}
func (h *slogHandler) Handle(_ context.Context, record slog.Record) error {
	fields := map[string]any{}
	for _, attribute := range h.attributes {
		putSlogAttribute(fields, attribute.groups, attribute.attribute)
	}
	record.Attrs(func(attribute slog.Attr) bool { putSlogAttribute(fields, h.groups, attribute); return true })
	if record.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{record.PC}).Next()
		fields["caller"] = fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
		fields["function"] = frame.Function
	}
	level := "debug"
	switch {
	case record.Level >= slog.LevelError:
		level = "error"
	case record.Level >= slog.LevelWarn:
		level = "warning"
	case record.Level >= slog.LevelInfo:
		level = "info"
	}
	Record(level, "transport", "transport.log", record.Message, fields)
	return nil
}
func (h *slogHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	next := &slogHandler{groups: slices.Clone(h.groups), attributes: slices.Clone(h.attributes)}
	for _, attribute := range attributes {
		next.attributes = append(next.attributes, scopedAttribute{groups: slices.Clone(h.groups), attribute: attribute})
	}
	return next
}
func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &slogHandler{groups: append(slices.Clone(h.groups), name), attributes: slices.Clone(h.attributes)}
}
func putSlogAttribute(fields map[string]any, groups []string, attribute slog.Attr) {
	attribute.Value = attribute.Value.Resolve()
	if attribute.Equal(slog.Attr{}) {
		return
	}
	for _, group := range groups {
		nested, ok := fields[group].(map[string]any)
		if !ok {
			nested = map[string]any{}
			fields[group] = nested
		}
		fields = nested
	}
	if attribute.Value.Kind() == slog.KindGroup {
		nested := fields
		if attribute.Key != "" {
			nested = map[string]any{}
			fields[attribute.Key] = nested
		}
		for _, child := range attribute.Value.Group() {
			putSlogAttribute(nested, nil, child)
		}
		return
	}
	fields[attribute.Key] = attribute.Value.Any()
}
