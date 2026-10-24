// Command herdr-mobile serves a phone-sized view of the agents running in
// herdr on this laptop: their states, their recent output, and a way to answer
// them (docs/herdr-mobile/README.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"apps/herdr-mobile/herdr"
	"apps/herdr-mobile/web"
)

const defaultAddr = "127.0.0.1:9393"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error("herdr-mobile stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	socket, err := socketPath(os.Getenv)
	if err != nil {
		return err
	}
	addr := envOr("HM_ADDR", defaultAddr)
	uiDir := envOr("HM_UI_DIR", "ui/build")
	if _, err := os.Stat(filepath.Join(uiDir, "index.html")); err != nil {
		log.Warn("no built UI; serving the API only (run nx run herdr-mobile:build-ui)", "dir", uiDir)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Handler: web.New(web.Options{
			Herdr:        herdr.New(socket),
			UIDir:        uiDir,
			Log:          log,
			AllowedHosts: splitList(os.Getenv("HM_ALLOWED_HOSTS")),
		}),
		ReadHeaderTimeout: 10 * time.Second,
		// No WriteTimeout: the event stream stays open while the page is.
		IdleTimeout: 2 * time.Minute,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	log.Info("herdr-mobile listening", "url", "http://"+ln.Addr().String()+"/ui/", "herdr", socket)

	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Open event streams never finish on their own, so a timeout here is expected; close them.
	if err := srv.Shutdown(shutdown); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("shut down: %w", err)
	}
	return srv.Close()
}

// socketPath follows herdr's own order: HERDR_SOCKET_PATH, then the named
// session in HERDR_SESSION, then the default session.
func socketPath(getenv func(string) string) (string, error) {
	if p := strings.TrimSpace(getenv("HERDR_SOCKET_PATH")); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the home folder for herdr's socket: %w", err)
	}
	dir := filepath.Join(home, ".config", "herdr")
	if name := strings.TrimSpace(getenv("HERDR_SESSION")); name != "" {
		return filepath.Join(dir, "sessions", name, "herdr.sock"), nil
	}
	return filepath.Join(dir, "herdr.sock"), nil
}

func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.ToLower(strings.TrimSpace(part)); part != "" {
			out = append(out, part)
		}
	}
	return out
}
