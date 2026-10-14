package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"sync/atomic"
)

// settingsService edits the stored engine settings. Services are wired once
// at startup, so saved changes apply when the engine restarts itself.
type settingsService struct {
	store   port.SettingsStore
	restart func()
	// changed records a save the running engine does not use yet.
	changed atomic.Bool
}

// NewSettingsService takes restart, which must return quickly and restart
// the engine after the current response is sent.
func NewSettingsService(store port.SettingsStore, restart func()) port.SettingsService {
	return &settingsService{store: store, restart: restart}
}

func (s *settingsService) View(ctx context.Context) (domain.SettingsView, error) {
	values, err := s.store.Values(ctx)
	if err != nil {
		return domain.SettingsView{}, err
	}
	return domain.BuildSettingsView(values, s.store.Path(), s.changed.Load()), nil
}

func (s *settingsService) Update(ctx context.Context, update domain.SettingsUpdate) (domain.SettingsView, error) {
	current, err := s.store.Values(ctx)
	if err != nil {
		return domain.SettingsView{}, err
	}
	changes, merged, err := domain.ApplySettingsUpdate(current, update)
	if err != nil {
		return domain.SettingsView{}, err
	}
	if err := s.store.Save(ctx, changes); err != nil {
		return domain.SettingsView{}, err
	}
	s.changed.Store(true)
	return domain.BuildSettingsView(merged, s.store.Path(), true), nil
}

func (s *settingsService) Restart(context.Context) error {
	s.restart()
	return nil
}
