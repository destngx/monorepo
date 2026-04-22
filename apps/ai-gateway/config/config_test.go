package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReloadOpenAIOAuthSourceReadsAccessToken(t *testing.T) {
	t.Setenv(openAIOAuthPathEnv, filepath.Join(t.TempDir(), "auth.json"))
	t.Setenv(codexAuthPathEnv, "")

	authPath := os.Getenv(openAIOAuthPathEnv)
	if err := os.WriteFile(authPath, []byte(`{
  "auth_mode": "chatgpt",
  "tokens": {
    "account_id": "account-1",
    "access_token": "access-1",
    "refresh_token": "refresh-1"
  }
}`), 0600); err != nil {
		t.Fatal(err)
	}

	oauth, source := ReloadOpenAIOAuthSource()
	if oauth == nil {
		t.Fatal("expected oauth tokens")
	}
	if source != authPath {
		t.Fatalf("expected source %q, got %q", authPath, source)
	}
	if oauth.AccessToken != "access-1" {
		t.Fatalf("expected access token to be loaded")
	}
	if oauth.RefreshToken != "refresh-1" {
		t.Fatalf("expected refresh token to be loaded")
	}
}

func TestReloadOpenAIOAuthSourceReadsTopLevelAccessToken(t *testing.T) {
	t.Setenv(openAIOAuthPathEnv, filepath.Join(t.TempDir(), "auth.json"))
	t.Setenv(codexAuthPathEnv, "")

	authPath := os.Getenv(openAIOAuthPathEnv)
	if err := os.WriteFile(authPath, []byte(`{
  "auth_mode": "chatgpt",
  "access_token": "top-access",
  "tokens": {
    "account_id": "account-1"
  }
}`), 0600); err != nil {
		t.Fatal(err)
	}

	oauth, _ := ReloadOpenAIOAuthSource()
	if oauth == nil {
		t.Fatal("expected oauth tokens")
	}
	if oauth.AccessToken != "top-access" {
		t.Fatalf("expected top-level access token to be loaded")
	}
}

func TestSaveOpenAIOAuthSourcePreservesExistingFields(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{
  "auth_mode": "chatgpt",
  "OPENAI_API_KEY": null,
  "tokens": {
    "account_id": "account-1",
    "id_token": "id-1",
    "access_token": "old-access",
    "refresh_token": "old-refresh"
  },
  "last_refresh": "old"
}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := SaveOpenAIOAuthSource(authPath, &OpenAIOAuth{
		AccessToken: "new-access",
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(data), []string{
		`"auth_mode": "chatgpt"`,
		`"OPENAI_API_KEY": null`,
		`"account_id": "account-1"`,
		`"id_token": "id-1"`,
		`"access_token": "new-access"`,
		`"refresh_token": "old-refresh"`,
	}) {
		t.Fatalf("saved auth did not preserve and update expected fields:\n%s", data)
	}
}

func containsAll(s string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(s, needle) {
			return false
		}
	}
	return true
}
