// Command health-watcher is the engine: the API and the built UI in one
// process (docs/health-watcher/REQUIREMENTS.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"apps/health-watcher/domain"
	"apps/health-watcher/internal/app"
	"apps/health-watcher/internal/config"
	"apps/health-watcher/internal/multilisten"
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
	for _, a := range listenAddrs(ln) {
		log.Info("engine listening", "addr", "http://"+a, "ui", "http://"+a+"/ui", "data", cfg.DataDir)
	}

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

// listenRetries are the waits between tries of an address, about 30
// seconds in all: at boot the Tailscale address can appear after the engine
// starts, and a previous engine may still hold the port.
var listenRetries = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second}

// listen opens the saved address. A host other than loopback, such as the
// Mac's Tailscale address, is opened next to 127.0.0.1 on the same port, so
// the browser on the Mac and the UI dev server keep working. When the saved
// address keeps failing the engine falls back to the default, so the Engine
// settings page stays reachable on the Mac.
func listen(ctx context.Context, engine *app.Engine, log *slog.Logger) (net.Listener, error) {
	saved, fallback := engine.Settings.Current(), domain.DefaultSettings()
	host := netip.MustParseAddr(saved.Host)
	local := saved.Addr()
	if !host.IsLoopback() && !host.IsUnspecified() {
		local = netip.AddrPortFrom(netip.MustParseAddr(fallback.Host), uint16(saved.Port)).String()
	}
	ln, err := listenRetrying(ctx, local, local != fallback.Addr(), log)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		log.Error("cannot listen on the saved address; using the default until it is fixed on the Engine settings page",
			"addr", local, "error", err)
		if ln, err = net.Listen("tcp", fallback.Addr()); err != nil {
			return nil, fmt.Errorf("listen on %s: %w", fallback.Addr(), err)
		}
		engine.Settings.StartedOn(fallback.Host, fallback.Port)
		return ln, nil
	}
	if local == saved.Addr() {
		return ln, nil
	}
	extra, err := listenRetrying(ctx, saved.Addr(), true, log)
	if err != nil {
		if ctx.Err() != nil {
			_ = ln.Close()
			return nil, ctx.Err()
		}
		log.Error("cannot listen on the saved host; the engine answers on this Mac only until it is fixed on the Engine settings page",
			"addr", saved.Addr(), "error", err)
		engine.Settings.StartedOn(fallback.Host, saved.Port)
		return ln, nil
	}
	return multilisten.New(ln, extra), nil
}

// listenRetrying opens addr, trying again after each of listenRetries when
// retry is set.
func listenRetrying(ctx context.Context, addr string, retry bool, log *slog.Logger) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err == nil || !retry {
		return ln, err
	}
	for _, wait := range listenRetries {
		log.Warn("cannot listen on the saved address yet; trying again", "addr", addr, "error", err, "wait", wait.String())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		if ln, err = net.Listen("tcp", addr); err == nil {
			return ln, nil
		}
	}
	return nil, err
}

func listenAddrs(ln net.Listener) []string {
	if m, ok := ln.(*multilisten.Listener); ok {
		var out []string
		for _, a := range m.Addrs() {
			out = append(out, a.String())
		}
		return out
	}
	return []string{ln.Addr().String()}
}
