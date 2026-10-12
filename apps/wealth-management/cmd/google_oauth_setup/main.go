package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var envCandidates = []string{
	"../../.env.local",
	".env.local",
}

func main() {
	for _, path := range envCandidates {
		_ = godotenv.Overload(path)
	}

	reader := bufio.NewReader(os.Stdin)

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		fmt.Print("Enter GOOGLE_CLIENT_ID: ")
		val, _ := reader.ReadString('\n')
		clientID = strings.TrimSpace(val)
	}

	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	if clientSecret == "" {
		fmt.Print("Enter GOOGLE_CLIENT_SECRET: ")
		val, _ := reader.ReadString('\n')
		clientSecret = strings.TrimSpace(val)
	}

	redirectURL := strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL"))
	if redirectURL == "" {
		redirectURL = "http://127.0.0.1:3000"
	}

	if clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "Error: Client ID and Secret are required.")
		os.Exit(1)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/spreadsheets"},
		Endpoint:     google.Endpoint,
	}

	authURL := oauthConfig.AuthCodeURL("wm-google-sheets", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Println("Open this URL and authorize Google Sheets access:")
	fmt.Println(authURL)
	fmt.Println("")
	fmt.Printf("Paste the full redirected URL (starts with %s):\n", redirectURL)

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed reading redirected URL: %v\n", err)
		os.Exit(1)
	}

	code, err := extractCode(strings.TrimSpace(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid redirected URL: %v\n", err)
		os.Exit(1)
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oauth code exchange failed: %v\n", err)
		os.Exit(1)
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		fmt.Fprintln(os.Stderr, "no refresh token returned; revoke app access and retry with consent")
		os.Exit(1)
	}

	envPath := selectEnvPath()
	_ = upsertEnvValue(envPath, "GOOGLE_CLIENT_ID", clientID)
	_ = upsertEnvValue(envPath, "GOOGLE_CLIENT_SECRET", clientSecret)
	if err := upsertEnvValue(envPath, "GOOGLE_REFRESH_TOKEN", token.RefreshToken); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing .env.local: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s with GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REFRESH_TOKEN.\n", envPath)
}

func extractCode(redirectedURL string) (string, error) {
	parsed, err := url.Parse(redirectedURL)
	if err != nil {
		return "", err
	}
	code := strings.TrimSpace(parsed.Query().Get("code"))
	if code == "" {
		return "", errors.New("missing code query parameter")
	}
	return code, nil
}

func selectEnvPath() string {
	for _, path := range envCandidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return envCandidates[0]
}

func upsertEnvValue(path string, key string, value string) error {
	current := ""
	if data, err := os.ReadFile(filepath.Clean(path)); err == nil {
		current = string(data)
	}

	lines := strings.Split(current, "\n")
	updated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = key + "=" + value
			updated = true
			break
		}
	}
	if !updated {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, key+"="+value)
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(filepath.Clean(path), []byte(content), 0600)
}
