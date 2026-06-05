package openai

import (
	"apps/ai-gateway/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func (p *Provider) authMode() string {
	if p.useCodex() {
		return "oauth"
	}
	if p.apiKey != "" {
		return "api_key"
	}
	return "oauth"
}

func (p *Provider) refreshOAuth() *config.OpenAIOAuth {
	if oauth, source := config.ReloadOpenAIOAuthSource(); oauth != nil {
		p.mu.Lock()
		p.oauth = oauth
		p.source = source
		p.mu.Unlock()
		return oauth
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.oauth
}

func (p *Provider) refreshAccessToken(ctx context.Context, force bool) (*config.OpenAIOAuth, error) {
	oauth := p.refreshOAuth()
	if oauth == nil || oauth.RefreshToken == "" {
		return oauth, nil
	}
	if !force && oauth.AccessToken != "" {
		return oauth, nil
	}

	clientID := os.Getenv(envOpenAIOAuthClientID)
	if clientID == "" {
		return oauth, nil
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", oauth.RefreshToken)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return oauth, err
	}
	httpReq.Header.Set(headerContentType, "application/x-www-form-urlencoded")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return oauth, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return oauth, fmt.Errorf("openai oauth refresh error %d: %s", resp.StatusCode, body)
	}

	var refreshed oauthRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshed); err != nil {
		return oauth, fmt.Errorf("failed to decode openai oauth refresh: %w", err)
	}
	if refreshed.AccessToken == "" {
		return oauth, fmt.Errorf("openai oauth refresh did not return access_token")
	}

	next := *oauth
	if refreshed.AccountID != "" {
		next.AccountID = refreshed.AccountID
	}
	if refreshed.ChatGPTAccountID != "" {
		next.AccountID = refreshed.ChatGPTAccountID
	}
	if refreshed.IDToken != "" {
		next.IDToken = refreshed.IDToken
	}
	next.AccessToken = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		next.RefreshToken = refreshed.RefreshToken
	}

	p.mu.Lock()
	p.oauth = &next
	source := p.source
	p.mu.Unlock()

	if source != "" {
		_ = config.SaveOpenAIOAuthSource(source, &next)
	}

	return &next, nil
}

func (p *Provider) setAuthHeaders(httpReq *http.Request) error {
	if p.useCodex() {
		oauth, err := p.refreshAccessToken(httpReq.Context(), false)
		if err != nil {
			return err
		}
		if oauth != nil {
			if oauth.AccessToken != "" {
				httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+oauth.AccessToken)
			} else if oauth.IDToken != "" {
				httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+oauth.IDToken)
			}
			if oauth.AccountID != "" {
				httpReq.Header.Set(headerChatGPTAcctID, oauth.AccountID)
			}
		}
		return nil
	}

	if p.apiKey != "" {
		httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+p.apiKey)
		return nil
	}

	return nil
}
