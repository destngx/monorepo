package service

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"maps"
	"testing"
)

type memorySettingsStore struct {
	values  map[string]string
	saveErr error
	saves   int
}

func (m *memorySettingsStore) Values(context.Context) (map[string]string, error) {
	return maps.Clone(m.values), nil
}

func (m *memorySettingsStore) Save(_ context.Context, changes map[string]string) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saves++
	for key, value := range changes {
		if value == "" {
			delete(m.values, key)
		} else {
			m.values[key] = value
		}
	}
	return nil
}

func (m *memorySettingsStore) Path() string { return "/cfg/settings.db" }

func settingValue(view domain.SettingsView, key string) domain.SettingView {
	for _, setting := range view.Settings {
		if setting.Key == key {
			return setting
		}
	}
	return domain.SettingView{}
}

func TestGivenAValidUpdateWhenSavingThenItPersistsAndARestartIsRequired(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{"AI_API_KEY": "sk-old"}}
	svc := NewSettingsService(store, func() {})
	ctx := context.Background()

	before, err := svc.View(ctx)
	if err != nil || before.RestartRequired || before.Path != "/cfg/settings.db" {
		t.Fatalf("before = %+v err %v", before.RestartRequired, err)
	}
	model := "gpt-main"
	view, err := svc.Update(ctx, domain.SettingsUpdate{Values: map[string]*string{"AI_MODEL": &model}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if store.values["AI_MODEL"] != "gpt-main" || store.values["AI_API_KEY"] != "sk-old" {
		t.Fatalf("stored = %v", store.values)
	}
	if !view.RestartRequired || settingValue(view, "AI_MODEL").Value != "gpt-main" || !settingValue(view, "AI_API_KEY").IsSet {
		t.Fatalf("view = %+v", view)
	}
	if after, _ := svc.View(ctx); !after.RestartRequired {
		t.Fatal("restart flag must stick until the engine restarts")
	}
}

func TestGivenAnInvalidUpdateWhenSavingThenNothingIsStored(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{}}
	svc := NewSettingsService(store, func() {})
	bad := "soon"

	_, err := svc.Update(context.Background(), domain.SettingsUpdate{Values: map[string]*string{"AI_TIMEOUT_SECONDS": &bad}})

	if !errors.Is(err, domain.ErrValidation) || store.saves != 0 {
		t.Fatalf("err %v saves %d", err, store.saves)
	}
	if view, _ := svc.View(context.Background()); view.RestartRequired {
		t.Fatal("a rejected update must not ask for a restart")
	}
}

func TestGivenAStoreFailureWhenSavingThenTheErrorIsReturned(t *testing.T) {
	store := &memorySettingsStore{values: map[string]string{}, saveErr: errors.New("disk full")}
	svc := NewSettingsService(store, func() {})
	value := "x"

	if _, err := svc.Update(context.Background(), domain.SettingsUpdate{Values: map[string]*string{"AI_MODEL": &value}}); err == nil {
		t.Fatal("expected the store error")
	}
}

func TestGivenARestartRequestThenTheRestarterIsCalled(t *testing.T) {
	called := false
	svc := NewSettingsService(&memorySettingsStore{values: map[string]string{}}, func() { called = true })
	if err := svc.Restart(context.Background()); err != nil || !called {
		t.Fatalf("called %v err %v", called, err)
	}
}
