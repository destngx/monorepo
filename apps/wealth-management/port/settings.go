package port

import (
	"apps/wealth-management/domain"
	"context"
)

// SettingsStore persists engine settings (no env file).
type SettingsStore interface {
	// Values returns every stored setting; unset ones are absent.
	Values(ctx context.Context) (map[string]string, error)
	// Save applies changes in one transaction; an empty value deletes the key.
	Save(ctx context.Context, changes map[string]string) error
	// Path is where the settings live, shown in the UI.
	Path() string
}

type SettingsService interface {
	View(ctx context.Context) (domain.SettingsView, error)
	// Update validates and saves; the running engine keeps its old settings
	// until Restart.
	Update(ctx context.Context, update domain.SettingsUpdate) (domain.SettingsView, error)
	// Restart asks the engine to restart with the saved settings.
	Restart(ctx context.Context) error
}
