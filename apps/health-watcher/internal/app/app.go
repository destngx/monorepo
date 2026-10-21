// Package app wires the engine from its configuration. main and the
// commands share it.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
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
	"apps/health-watcher/port"
	"apps/health-watcher/service"
)

// Engine is the wired engine.
type Engine struct {
	Store    *sqlite.Store
	Services fiberAdapter.Services
	// Pipeline is the background reader; Start runs it.
	Pipeline *service.Pipeline
	// Settings holds the engine settings from the database (HW-OPS-9).
	Settings *service.Engine
	// ImportedEnv says the retired environment variables were saved as the
	// first settings at this start.
	ImportedEnv bool
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
	defaultBin, err := config.DefaultMacOCRBin()
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	settings := &service.Engine{Deps: deps, Log: log, DefaultMacOCRBin: defaultBin, LocalAddrs: localAddrs,
		PingAI: func(ctx context.Context, s domain.Settings) error { return newReader(s).Ping(ctx) }}
	current, imported, err := settings.Load(ctx, cfg.Legacy)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	pipeline := &service.Pipeline{Deps: deps, Masker: imaging.Masker{Quality: 85}, Catalog: domain.DefaultCatalog,
		WorkDir: cfg.WorkDir(), Log: log}
	records := &service.Records{Deps: deps}
	inbox := &service.Inbox{Deps: deps, Records: records, Pipeline: pipeline}
	settings.Apply = func(s domain.Settings) {
		bin := s.MacOCRBin
		if bin == "" {
			bin = defaultBin
		}
		var reader port.Reader
		if s.AIEnabled() {
			reader = newReader(s)
		}
		pipeline.Configure(macocr.New(bin), reader, s.Workers)
		inbox.SetLimits(service.Limits{MaxFileBytes: int64(s.MaxFileMB) << 20, MaxBatchBytes: int64(s.MaxBatchMB) << 20})
	}
	settings.Apply(current)
	return &Engine{
		SetupCodePath: setupPath,
		Store:         store,
		Pipeline:      pipeline,
		Settings:      settings,
		ImportedEnv:   imported,
		Services: fiberAdapter.Services{
			Accounts: accounts,
			Tenants:  &service.Tenants{Deps: deps, Accounts: accounts},
			Records:  records,
			Inbox:    inbox,
			Clinical: &service.Clinical{Deps: deps, Catalog: domain.DefaultCatalog},
			Engine:   settings,
		},
	}, nil
}

func newReader(s domain.Settings) *openai.Reader {
	return openai.New(openai.Config{BaseURL: s.AIBaseURL, APIKey: s.AIAPIKey, Model: domain.AIModel, Effort: domain.AIEffort,
		Timeout: time.Duration(s.AITimeoutSeconds) * time.Second}, nil)
}

// localAddrs lists the addresses of this Mac's network interfaces.
func localAddrs() ([]netip.Addr, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(addrs))
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if ip, ok := netip.AddrFromSlice(n.IP); ok {
				out = append(out, ip.Unmap())
			}
		}
	}
	return out, nil
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

// HTTP builds the Fiber app for the engine. Its body limit is the batch
// limit at start; a larger one waits for the next start.
func (e *Engine) HTTP(log *slog.Logger) *fiber.App {
	// Multipart framing adds a little to the batch; 1 MiB covers it.
	return fiberAdapter.New(e.Services, fiberAdapter.Options{
		Log:        log,
		BodyLimit:  (e.Settings.Current().MaxBatchMB + 1) << 20,
		UIBuildDir: config.UIBuildDir,
	})
}

// Close closes the store.
func (e *Engine) Close() error { return e.Store.Close() }
