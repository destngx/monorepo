package port

import (
	"apps/wealth-management/domain"
	"context"
)

// GoogleOAuth runs the OAuth authorization code flow (with PKCE) against
// Google.
type GoogleOAuth interface {
	// AuthCodeURL is the consent page asking for offline access to scopes.
	AuthCodeURL(client domain.GoogleOAuthClient, scopes []string, state, verifier string) string
	// Exchange trades the code for a refresh token; it fails when Google
	// returns none.
	Exchange(ctx context.Context, client domain.GoogleOAuthClient, code, verifier string) (refreshToken string, err error)
}

// GoogleAuthService connects a Google account from the UI and saves its
// refresh token to the settings.
type GoogleAuthService interface {
	// Start begins a consent. origin is where the UI was opened (the
	// request's Origin header); a loopback origin gets the callback there.
	Start(ctx context.Context, purpose domain.GoogleAuthPurpose, origin string) (domain.GoogleAuthStart, error)
	// Complete finishes a consent started by Start; each state works once.
	Complete(ctx context.Context, state, code string) (domain.GoogleAuthResult, error)
}
