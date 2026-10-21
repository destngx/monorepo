// Command health-watcher is the engine: the API and the built UI in one
// process (docs/health-watcher/REQUIREMENTS.md).
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	log.Info("background reader started", "mac_ocr", cfg.MacOCRBin, "workers", cfg.Workers, "ai", cfg.AI.Enabled(), "model", cfg.AI.Model,
		"effort", cfg.AI.Effort)
	server := engine.HTTP(cfg, log)
	errs := make(chan error, 1)
	go func() { errs <- server.Listen(cfg.Addr()) }()
	log.Info("engine listening", "addr", "http://"+cfg.Addr(), "ui", "http://"+cfg.Addr()+"/ui", "data", cfg.DataDir)

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
