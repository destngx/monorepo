// Package app wires the engine from its configuration. main and the
// commands share it.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/adapter/ai/openai"
	fiberAdapter "apps/health-watcher/adapter/fiber"
	"apps/health-watcher/adapter/filestore"
	"apps/health-watcher/adapter/imaging"
	"apps/health-watcher/adapter/macocr"
	"apps/health-watcher/adapter/password"
	"apps/health-watcher/adapter/sqlite"
	"apps/health-watcher/domain"
	"apps/health-watcher/internal/config"
	"apps/health-watcher/service"
)

// Engine is the wired engine.
type Engine struct {
	Store    *sqlite.Store
	Services fiberAdapter.Services
	// Pipeline is the background reader; Start runs it.
	Pipeline *service.Pipeline
	// SetupCodePath is the file holding the first-run setup code, or ""
	// once setup is done (HW-AUTH-4).
	SetupCodePath string
}

// Open opens the store and builds the services. log is the background
// reader's logger.
func Open(ctx context.Context, cfg config.Config, log *slog.Logger) (*Engine, error) {
	store, err := sqlite.Open(ctx, cfg.DBPath())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	files, err := filestore.New(cfg.FilesDir())
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	deps := service.Deps{Store: store, Files: files, Hasher: password.New(password.Default), Clock: func() time.Time { return time.Now().UTC() }, Location: loc,
		FileLock: &sync.Mutex{}}
	accounts, err := service.NewAccounts(deps)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	setupPath, err := writeSetupCode(ctx, accounts, cfg.SetupCodePath())
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	if _, err := store.RebuildSearchIndex(ctx, domain.SearchIndexVersion); err != nil {
		_ = store.Close()
		return nil, err
	}
	pipeline := &service.Pipeline{Deps: deps, OCR: macocr.New(cfg.MacOCRBin), Masker: imaging.Masker{Quality: 85},
		Catalog: domain.DefaultCatalog, WorkDir: cfg.WorkDir(), Log: log, Workers: cfg.Workers}
	if cfg.AI.Enabled() {
		pipeline.Reader = openai.New(openai.Config{BaseURL: cfg.AI.BaseURL, APIKey: cfg.AI.APIKey, Model: cfg.AI.Model,
			Effort: cfg.AI.Effort, Timeout: cfg.AI.Timeout}, nil)
	}
	records := &service.Records{Deps: deps}
	limits := service.Limits{MaxFileBytes: int64(cfg.MaxFileMB) << 20, MaxBatchBytes: int64(cfg.MaxBatchMB) << 20}
	return &Engine{
		SetupCodePath: setupPath,
		Store:         store,
		Pipeline:      pipeline,
		Services: fiberAdapter.Services{
			Accounts: accounts,
			Tenants:  &service.Tenants{Deps: deps, Accounts: accounts},
			Records:  records,
			Inbox:    &service.Inbox{Deps: deps, Records: records, Limits: limits, Pipeline: pipeline},
			Clinical: &service.Clinical{Deps: deps, Catalog: domain.DefaultCatalog},
		},
	}, nil
}

// writeSetupCode writes a fresh setup code to path (mode 0600) while no
// user exists, and removes a stale file once setup is done.
func writeSetupCode(ctx context.Context, accounts *service.Accounts, path string) (string, error) {
	code, err := accounts.PrepareSetup(ctx)
	if err != nil {
		return "", err
	}
	if code == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("remove old setup code: %w", err)
		}
		return "", nil
	}
	if err := os.WriteFile(path, []byte(code+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write setup code: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("restrict setup code: %w", err)
	}
	return path, nil
}

// HTTP builds the Fiber app for the engine.
func (e *Engine) HTTP(cfg config.Config, log *slog.Logger) *fiber.App {
	// Multipart framing adds a little to the batch; 1 MiB covers it.
	return fiberAdapter.New(e.Services, fiberAdapter.Options{
		Log:        log,
		BodyLimit:  (cfg.MaxBatchMB + 1) << 20,
		UIBuildDir: cfg.UIBuildDir,
	})
}

// Close closes the store.
func (e *Engine) Close() error { return e.Store.Close() }
