package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	clientID     = "01ab8ac9400c4e429b23" // GitHub Copilot for VSCode client ID
	deviceCodeURL = "https://github.com/login/device/code"
	tokenURL      = "https://github.com/login/oauth/access_token"
)

var envCandidates = []string{
	"../../.env.local",
	".env.local",
}

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func main() {
	for _, path := range envCandidates {
		_ = godotenv.Overload(path)
	}

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Println("🤖 GitHub Copilot OAuth Setup (Go Engine)")
	fmt.Println("──────────────────────────────────────────────────")

	// 1. Request device code
	payload := map[string]string{
		"client_id": clientID,
		"scope":     "read:user user:email",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", deviceCodeURL, bytes.NewBuffer(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error requesting device code: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var dcResp DeviceCodeResponse
	json.NewDecoder(resp.Body).Decode(&dcResp)

	fmt.Printf("\n1. Open your browser to: %s\n", dcResp.VerificationURI)
	fmt.Printf("2. Enter the following code: \033[32m\033[1m%s\033[0m\n\n", dcResp.UserCode)
	fmt.Println("Waiting for authorization...")

	// 2. Poll for the access token
	interval := time.Duration(dcResp.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	var token string
	for {
		time.Sleep(interval)

		tokenPayload := map[string]string{
			"client_id":   clientID,
			"device_code": dcResp.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}
		tokenBody, _ := json.Marshal(tokenPayload)

		req, _ = http.NewRequest("POST", tokenURL, bytes.NewBuffer(tokenBody))
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			fmt.Print(".")
			continue
		}
		defer resp.Body.Close()

		var tResp TokenResponse
		json.NewDecoder(resp.Body).Decode(&tResp)

		if tResp.Error != "" {
			if tResp.Error == "authorization_pending" {
				fmt.Print(".")
				continue
			} else if tResp.Error == "slow_down" {
				interval += 5 * time.Second
				continue
			} else {
				fmt.Printf("\nError: %s\n", tResp.Error)
				os.Exit(1)
			}
		}

		if tResp.AccessToken != "" {
			token = tResp.AccessToken
			break
		}
	}

	fmt.Println("\n\n✅ Successfully authenticated with GitHub!")

	envPath := selectEnvPath()
	if err := upsertEnvValue(envPath, "GITHUB_TOKEN", token); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing .env.local: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔑 Saved GITHUB_TOKEN to %s\n", envPath)
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
