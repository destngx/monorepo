package logger

import (
	"io"
	"log/slog"
	"testing"
)

func NewTestLogger(t *testing.T) *Logger {
	noOpHandler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{})
	return &Logger{
		Logger:       slog.New(noOpHandler),
		colorEnabled: false,
	}
}
