// Package googleauth runs Google's OAuth authorization code flow.
package googleauth

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuth struct{}

func New() OAuth { return OAuth{} }

func config(client domain.GoogleOAuthClient, scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		RedirectURL:  client.RedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}
}

// AuthCodeURL forces the consent screen so Google always returns a refresh
// token, even for an account that consented before.
func (OAuth) AuthCodeURL(client domain.GoogleOAuthClient, scopes []string, state, verifier string) string {
	return config(client, scopes).AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.S256ChallengeOption(verifier))
}

func (OAuth) Exchange(ctx context.Context, client domain.GoogleOAuthClient, code, verifier string) (string, error) {
	token, err := config(client, nil).Exchange(ctx, code, oauth2.VerifierOption(verifier))
	var rejected *oauth2.RetrieveError
	if errors.As(err, &rejected) {
		return "", fmt.Errorf("%w (%w)", domain.ErrGoogleCodeRejected, err)
	}
	if err != nil {
		return "", fmt.Errorf("google oauth code exchange: %w", err)
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		return "", domain.ErrGoogleNoRefreshToken
	}
	return token.RefreshToken, nil
}
