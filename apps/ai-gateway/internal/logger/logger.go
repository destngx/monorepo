package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Handler implements slog.Handler with a focus on terminal UX and color.
type Handler struct {
	w            io.Writer
	level        slog.Level
	enableColor  bool
	timeColor    *color.Color
	levelColors  map[slog.Level]*color.Color
	attrKeyColor *color.Color
}

func NewHandler(w io.Writer, level slog.Level, enableColor bool) *Handler {
	h := &Handler{
		w:           w,
		level:       level,
		enableColor: enableColor,
		levelColors: make(map[slog.Level]*color.Color),
	}

	if enableColor {
		h.timeColor = color.New(color.Faint)
		h.levelColors[slog.LevelDebug] = color.New(color.FgCyan)
		h.levelColors[slog.LevelInfo] = color.New(color.FgGreen)
		h.levelColors[slog.LevelWarn] = color.New(color.FgYellow)
		h.levelColors[slog.LevelError] = color.New(color.FgRed)
		h.attrKeyColor = color.New(color.Faint)
	}

	return h
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder

	// 1. Timestamp (UX requirement: include brackets and specific color)
	timeStr := fmt.Sprintf("[%s]", r.Time.Format(time.TimeOnly))
	if h.enableColor {
		timeStr = h.timeColor.Sprint(timeStr)
	}
	sb.WriteString(timeStr)
	sb.WriteByte(' ')

	// Start building the body of the log line
	var bodySb strings.Builder

	// 2. Level
	levelStr := fmt.Sprintf("%-5s", r.Level.String())
	bodySb.WriteString(levelStr)
	bodySb.WriteByte(' ')

	// 3. Message
	bodySb.WriteString(r.Message)

	// 4. Attributes
	r.Attrs(func(a slog.Attr) bool {
		key := a.Key
		// Only color keys if we aren't coloring the whole line
		if h.enableColor && r.Level < slog.LevelWarn {
			key = h.attrKeyColor.Sprint(key)
		}
		fmt.Fprintf(&bodySb, " %s=%v", key, a.Value.Any())
		return true
	})

	body := bodySb.String()

	// 5. Apply full-line color for WARN or ERROR
	if h.enableColor {
		if c, ok := h.levelColors[r.Level]; ok {
			if r.Level >= slog.LevelWarn {
				// Color the entire body
				body = c.Sprint(body)
			} else {
				// Only color the level tag part (first 5 chars)
				parts := strings.SplitN(body, " ", 2)
				if len(parts) == 2 {
					body = c.Sprint(parts[0]) + " " + parts[1]
				}
			}
		}
	}

	sb.WriteString(body)
	sb.WriteByte('\n')

	_, err := io.WriteString(h.w, sb.String())
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Simple implementation for now
	return h
}

func (h *Handler) WithGroup(name string) slog.Handler {
	// Simple implementation for now
	return h
}

// Init initializes the global slog default logger.
func Init(levelStr string, enableColor bool) {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := NewHandler(os.Stdout, level, enableColor)
	slog.SetDefault(slog.New(handler))
}
