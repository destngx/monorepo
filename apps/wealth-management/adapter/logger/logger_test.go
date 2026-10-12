package logger

import (
	"context"
	"io"
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
