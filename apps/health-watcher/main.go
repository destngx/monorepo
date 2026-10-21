// Command health-watcher is the engine: the API and the built UI in one
// process (docs/health-watcher/REQUIREMENTS.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/internal/app"
	"apps/health-watcher/internal/config"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error("engine stopped", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	engine, err := app.Open(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer func() {
		if err := engine.Close(); err != nil {
			log.Error("close database", "error", err)
		}
	}()

	if engine.ImportedEnv {
		log.Info("engine settings imported once from the environment; from now on they live in the database, on the Engine settings page",
			"names", config.Ignored(os.Getenv))
	} else {
		for _, name := range config.Ignored(os.Getenv) {
			log.Warn("environment variable ignored: engine settings live in the database, on the Engine settings page", "name", name)
		}
	}
	if engine.SetupCodePath != "" {
		log.Info("first-run setup: open the UI and enter the code from this file", "file", engine.SetupCodePath)
	}
	// The background reader stops before the database closes.
	readerCtx, stopReader := context.WithCancel(context.Background())
	workers, err := engine.Pipeline.Start(readerCtx)
	if err != nil {
		stopReader()
		return err
	}
	defer func() {
		stopReader()
		workers.Wait()
	}()
	settings := engine.Settings.Current()
	log.Info("background reader started", "workers", settings.Workers, "ai", settings.AIEnabled(), "model", domain.AIModel,
		"effort", domain.AIEffort)
	ln, err := listen(ctx, engine, log)
	if err != nil {
		return err
	}
	server := engine.HTTP(log)
	errs := make(chan error, 1)
	go func() { errs <- server.Listener(ln) }()
	addr := ln.Addr().String()
	log.Info("engine listening", "addr", "http://"+addr, "ui", "http://"+addr+"/ui", "data", cfg.DataDir)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	if err := server.ShutdownWithTimeout(10 * time.Second); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// listenRetries are the waits between tries of the saved address, about
// 30 seconds in all: at boot the Tailscale address can appear after the
// engine starts, and a previous engine may still hold the port.
var listenRetries = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second}

// listen opens the saved address. When that keeps failing it falls back to
// the default, so the Engine settings page stays reachable on the Mac.
func listen(ctx context.Context, engine *app.Engine, log *slog.Logger) (net.Listener, error) {
	saved, fallback := engine.Settings.Current(), domain.DefaultSettings()
	ln, err := net.Listen("tcp", saved.Addr())
	for _, wait := range listenRetries {
		if err == nil || saved.Addr() == fallback.Addr() {
			return ln, err
		}
		log.Warn("cannot listen on the saved address yet; trying again", "addr", saved.Addr(), "error", err, "wait", wait.String())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		ln, err = net.Listen("tcp", saved.Addr())
	}
	if err == nil {
		return ln, nil
	}
	log.Error("cannot listen on the saved address; using the default until it is fixed on the Engine settings page",
		"addr", saved.Addr(), "error", err)
	ln, err = net.Listen("tcp", fallback.Addr())
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", fallback.Addr(), err)
	}
	engine.Settings.StartedOn(fallback.Host, fallback.Port)
	return ln, nil
}
