package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"
)

// googleAuthTTL bounds how long a consent may take.
const googleAuthTTL = 10 * time.Minute

// maxPendingGoogleAuth bounds the consents waiting for their callback.
const maxPendingGoogleAuth = 16

type pendingGoogleAuth struct {
	purpose  domain.GoogleAuthPurpose
	client   domain.GoogleOAuthClient
	verifier string
	expires  time.Time
}

// googleAuthService connects a Google account from the UI. Each consent gets
// a random state (CSRF) and a PKCE verifier, kept in memory for
// googleAuthTTL and usable once; the minted refresh token is saved through
// the settings service, so the engine reports that a restart applies it.
type googleAuthService struct {
	store           port.SettingsStore
	settings        port.SettingsService
	oauth           port.GoogleOAuth
	defaultRedirect string
	now             func() time.Time

	mu      sync.Mutex
	pending map[string]pendingGoogleAuth
}

// NewGoogleAuthService takes defaultRedirect, the engine's own callback URL.
// The redirect is GOOGLE_REDIRECT_URL when set, else the callback on the
// UI's loopback origin, else defaultRedirect.
func NewGoogleAuthService(store port.SettingsStore, settings port.SettingsService, oauth port.GoogleOAuth, defaultRedirect string) port.GoogleAuthService {
	return newGoogleAuthService(store, settings, oauth, defaultRedirect, time.Now)
}

func newGoogleAuthService(store port.SettingsStore, settings port.SettingsService, oauth port.GoogleOAuth, defaultRedirect string, now func() time.Time) *googleAuthService {
	return &googleAuthService{store: store, settings: settings, oauth: oauth, defaultRedirect: defaultRedirect, now: now, pending: map[string]pendingGoogleAuth{}}
}

func (s *googleAuthService) Start(ctx context.Context, purpose domain.GoogleAuthPurpose, origin string) (domain.GoogleAuthStart, error) {
	values, err := s.store.Values(ctx)
	if err != nil {
		return domain.GoogleAuthStart{}, err
	}
	client := domain.GoogleOAuthClient{
		ClientID:     strings.TrimSpace(values["GOOGLE_CLIENT_ID"]),
		ClientSecret: strings.TrimSpace(values["GOOGLE_CLIENT_SECRET"]),
		RedirectURL:  strings.TrimSpace(values["GOOGLE_REDIRECT_URL"]),
	}
	if client.ClientID == "" || client.ClientSecret == "" {
		return domain.GoogleAuthStart{}, domain.ErrGoogleClientMissing
	}
	if client.RedirectURL == "" {
		client.RedirectURL = s.redirectFor(origin)
	}
	state, err := randomToken()
	if err != nil {
		return domain.GoogleAuthStart{}, err
	}
	verifier, err := randomToken()
	if err != nil {
		return domain.GoogleAuthStart{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for key, consent := range s.pending {
		if !now.Before(consent.expires) {
			delete(s.pending, key)
		}
	}
	if len(s.pending) >= maxPendingGoogleAuth {
		return domain.GoogleAuthStart{}, fmt.Errorf("%w: too many Google sign-ins in progress; finish one or wait %s", domain.ErrValidation, googleAuthTTL)
	}
	s.pending[state] = pendingGoogleAuth{purpose: purpose, client: client, verifier: verifier, expires: now.Add(googleAuthTTL)}

	return domain.GoogleAuthStart{
		AuthURL:     s.oauth.AuthCodeURL(client, purpose.Scopes(), state, verifier),
		RedirectURL: client.RedirectURL,
		Purpose:     purpose,
	}, nil
}

func (s *googleAuthService) Complete(ctx context.Context, state, code string) (domain.GoogleAuthResult, error) {
	consent, err := s.take(state)
	if err != nil {
		return domain.GoogleAuthResult{}, err
	}
	token, err := s.oauth.Exchange(ctx, consent.client, code, consent.verifier)
	if err != nil {
		return domain.GoogleAuthResult{}, err
	}
	key := consent.purpose.SettingKey()
	view, err := s.settings.Update(ctx, domain.SettingsUpdate{Values: map[string]*string{key: &token}})
	if err != nil {
		return domain.GoogleAuthResult{}, err
	}
	return domain.GoogleAuthResult{Purpose: consent.purpose, SettingKey: key, RestartRequired: view.RestartRequired}, nil
}

// redirectFor prefers the loopback origin the UI runs on (the Vite dev
// server forwards /api to the engine), else the engine's own callback.
func (s *googleAuthService) redirectFor(origin string) string {
	if callback, ok := domain.GoogleCallbackForOrigin(origin); ok {
		return callback
	}
	return s.defaultRedirect
}

// take removes the consent for state, so a replayed callback fails.
func (s *googleAuthService) take(state string) (pendingGoogleAuth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	consent, ok := s.pending[state]
	delete(s.pending, state)
	if !ok || !s.now().Before(consent.expires) {
		return pendingGoogleAuth{}, domain.ErrGoogleAuthState
	}
	return consent, nil
}

// randomToken is 32 random bytes, base64url: a CSRF state or a PKCE
// verifier (43 characters, within RFC 7636's 43-128).
func randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate oauth token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
