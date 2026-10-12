package logger

import (
	"context"
	"log/slog"
	"slices"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapHandler implements slog.Handler on a zap core. The encoder owns the
// message, timestamp and level keys; every other key is written once, the
// last value winning, so log lines stay valid for JSON parsers.
type zapHandler struct {
	core  zapcore.Core
	level slog.Level
	// attrs come from Logger.With, already qualified by their group.
	attrs []slog.Attr
	group string
}

func (h *zapHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *zapHandler) Handle(ctx context.Context, record slog.Record) error {
	var keys []string
	values := map[string]any{}
	add := func(key string, value any) {
		if _, seen := values[key]; !seen {
			keys = append(keys, key)
		}
		values[key] = value
	}
	if requestID := RequestID(ctx); requestID != "" {
		add("request_id", requestID)
	}
	for _, attr := range h.attrs {
		addAttr(add, "", attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		addAttr(add, h.group, attr)
		return true
	})

	fields := make([]zapcore.Field, 0, len(keys))
	for _, key := range keys {
		switch key {
		case "message", "timestamp", "level":
			// Reserved for the encoder.
			continue
		}
		fields = append(fields, zap.Any(key, values[key]))
	}
	entry := zapcore.Entry{Level: zapLevelFromSlog(record.Level), Time: record.Time, Message: record.Message}
	if checked := h.core.Check(entry, nil); checked != nil {
		checked.Write(fields...)
	}
	return nil
}

// addAttr flattens groups into dotted keys ("call.rows").
func addAttr(add func(string, any), prefix string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}
	key := attr.Key
	if prefix != "" && key != "" {
		key = prefix + "." + key
	} else if key == "" {
		key = prefix
	}
	if attr.Value.Kind() == slog.KindGroup {
		for _, member := range attr.Value.Group() {
			addAttr(add, key, member)
		}
		return
	}
	add(key, attr.Value.Any())
}

func (h *zapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = slices.Clone(h.attrs)
	for _, attr := range attrs {
		if h.group != "" {
			attr = slog.Attr{Key: h.group, Value: slog.GroupValue(attr)}
		}
		next.attrs = append(next.attrs, attr)
	}
	return &next
}

func (h *zapHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	if h.group != "" {
		name = h.group + "." + name
	}
	next.group = name
	return &next
}

// zapLevelFromSlog converts slog.Level to zapcore.Level.
func zapLevelFromSlog(level slog.Level) zapcore.Level {
	switch level {
	case slog.LevelDebug:
		return zap.DebugLevel
	case slog.LevelInfo:
		return zap.InfoLevel
	case slog.LevelWarn:
		return zap.WarnLevel
	case slog.LevelError:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}
