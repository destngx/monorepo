package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// captureStdStreams swaps os.Stdout and os.Stderr for pipes while fn runs and returns what each received.
func captureStdStreams(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()

	fn()

	outW.Close()
	errW.Close()
	outBytes, _ := io.ReadAll(outR)
	errBytes, _ := io.ReadAll(errR)
	return string(outBytes), string(errBytes)
}

// Stdout carries the MCP JSON-RPC stream, so logs must never be written there.
func TestNewLoggerWritesToStderrNotStdout(t *testing.T) {
	stdout, stderr := captureStdStreams(t, func() {
		log := NewLogger("info", false)
		log.LogApplicationEvent(context.Background(), "hello from logger test")
	})

	if stdout != "" {
		t.Fatalf("expected nothing on stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "hello from logger test") {
		t.Fatalf("expected log line on stderr, got %q", stderr)
	}
}

// decodeLines parses each JSON log line into ordered keys and values.
func decodeLines(t *testing.T, output string) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		decoder := json.NewDecoder(strings.NewReader(line))
		if _, err := decoder.Token(); err != nil {
			t.Fatalf("not a JSON object: %q", line)
		}
		fields := map[string]any{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				t.Fatalf("bad key in %q: %v", line, err)
			}
			if _, duplicate := fields[key.(string)]; duplicate {
				t.Fatalf("duplicate key %q in %q", key, line)
			}
			var value any
			if err := decoder.Decode(&value); err != nil {
				t.Fatalf("bad value in %q: %v", line, err)
			}
			fields[key.(string)] = value
		}
		lines = append(lines, fields)
	}
	return lines
}

func TestGivenLogRecordWhenEncodingThenEveryKeyAppearsOnce(t *testing.T) {
	var output bytes.Buffer
	log := newLogger(&output, "info", false)
	ctx := WithRequestID(context.Background(), "req-1")

	log.LogError(ctx, "sheet read failed", errors.New("boom"), slog.String("request_id", "req-1"), slog.String("sheet", "Accounts"), slog.String("sheet", "Budget"))

	line := decodeLines(t, output.String())[0]
	if line["message"] != "sheet read failed" || line["level"] != "error" || line["timestamp"] == nil {
		t.Fatalf("missing core fields: %v", line)
	}
	if line["request_id"] != "req-1" || line["error"] != "boom" || line["log_type"] != "error" || line["sheet"] != "Budget" {
		t.Fatalf("unexpected fields: %v", line)
	}
}

func TestGivenRequestContextWhenLoggingThenAddsRequestID(t *testing.T) {
	var output bytes.Buffer
	log := newLogger(&output, "info", false)

	log.LogApplicationEvent(WithRequestID(context.Background(), "req-2"), "inside a request")
	log.LogApplicationEvent(context.Background(), "outside a request")

	lines := decodeLines(t, output.String())
	if lines[0]["request_id"] != "req-2" {
		t.Fatalf("expected the context request id, got %v", lines[0])
	}
	if _, ok := lines[1]["request_id"]; ok || RequestID(context.Background()) != "" {
		t.Fatalf("no request id expected outside a request, got %v", lines[1])
	}
}

func TestGivenDerivedLoggerWhenLoggingThenKeepsAttrsAndGroups(t *testing.T) {
	var output bytes.Buffer
	log := newLogger(&output, "debug", false)

	log.With(slog.String("component", "sheets")).WithGroup("call").Info("read", slog.Int("rows", 3))

	line := decodeLines(t, output.String())[0]
	if line["component"] != "sheets" || line["call.rows"] != float64(3) || line["level"] != "info" {
		t.Fatalf("derived attrs lost: %v", line)
	}
}

func TestGivenLevelWhenBelowThresholdThenSkipsRecord(t *testing.T) {
	var output bytes.Buffer
	log := newLogger(&output, "warn", false)
	log.Info("hidden")
	log.Warn("shown")
	if lines := decodeLines(t, output.String()); len(lines) != 1 || lines[0]["level"] != "warn" {
		t.Fatalf("expected only the warning, got %v", lines)
	}
}
