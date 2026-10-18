package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"strings"
	"sync"
	"sync/atomic"
)

// apiTokenService keeps the write token (WM-SEC-1) in memory for the guard
// and in the settings store across restarts. Unlike other settings a new
// token applies at once, so the browser that set it keeps working.
type apiTokenService struct {
	store    port.SettingsStore
	loopback bool
	token    atomic.Pointer[string]
	// setting serialises Set so the store and memory agree.
	setting sync.Mutex
}

// NewAPITokenService starts with the token in values (the stored settings)
// and host, the address the engine listens on.
func NewAPITokenService(store port.SettingsStore, values map[string]string, host string) port.APITokenService {
	s := &apiTokenService{store: store, loopback: domain.IsLoopbackHost(host)}
	token := strings.TrimSpace(values[domain.APITokenSettingKey])
	s.token.Store(&token)
	return s
}

func (s *apiTokenService) Token() string { return *s.token.Load() }

func (s *apiTokenService) Status() domain.APITokenStatus {
	return domain.APITokenStatus{Set: s.Token() != "", Loopback: s.loopback}
}

func (s *apiTokenService) Set(ctx context.Context, update domain.APITokenUpdate) (domain.APITokenStatus, error) {
	token, err := domain.NormalizeAPIToken(update.Token)
	if err != nil {
		return domain.APITokenStatus{}, err
	}
	s.setting.Lock()
	defer s.setting.Unlock()
	if err := s.store.Save(ctx, map[string]string{domain.APITokenSettingKey: token}); err != nil {
		return domain.APITokenStatus{}, err
	}
	s.token.Store(&token)
	return s.Status(), nil
}
