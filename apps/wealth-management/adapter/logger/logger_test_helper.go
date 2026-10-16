package logger

import (
	"bytes"
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

// NewRecordingTestLogger writes JSON lines to the returned buffer, so a
// test can check what was logged. Not safe for concurrent logging.
func NewRecordingTestLogger(t *testing.T) (*Logger, *bytes.Buffer) {
	t.Helper()
	var buffer bytes.Buffer
	return newLogger(&buffer, "debug", false), &buffer
}
