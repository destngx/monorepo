package main

import (
	configAdapter "apps/wealth-management/adapter/config"
	settingsAdapter "apps/wealth-management/adapter/settings"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func main() {
	fmt.Println("--- Google Sheets Auth Verification (Go Engine) ---")

	values, path, err := readSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot read the settings store: %v\n", err)
		os.Exit(1)
	}
	clientID := strings.TrimSpace(values["GOOGLE_CLIENT_ID"])
	clientSecret := strings.TrimSpace(values["GOOGLE_CLIENT_SECRET"])
	refreshToken := strings.TrimSpace(values["GOOGLE_REFRESH_TOKEN"])
	sheetID := strings.TrimSpace(values["GOOGLE_SHEETS_ID"])

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		fmt.Fprintf(os.Stderr, "❌ Missing Google credentials in %s\n", path)
		fmt.Println("Save the OAuth client in Settings, then use Connect Google account.")
		os.Exit(1)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{sheets.SpreadsheetsScope},
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	ctx := context.Background()
	tokenSource := config.TokenSource(ctx, token)

	fmt.Println("🔄 Attempting to refresh access token...")
	newToken, err := tokenSource.Token()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Authentication failed: %v\n", err)
		if strings.Contains(err.Error(), "invalid_grant") {
			fmt.Println("\n💡 REASON: Your refresh token has expired or been revoked.")
			fmt.Println("   - If your GCP project is in \"Testing\" mode, this happens every 7 days.")
			fmt.Println("   - Use Connect Google account in Settings to get a new token.")
		}
		os.Exit(1)
	}

	if newToken.AccessToken != "" {
		fmt.Println("✅ Access token obtained successfully!")

		if sheetID != "" {
			fmt.Printf("🔄 Verifying access to spreadsheet: %s...\n", sheetID)
			srv, err := sheets.NewService(ctx, option.WithTokenSource(tokenSource))
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Unable to retrieve Sheets client: %v\n", err)
				os.Exit(1)
			}

			_, err = srv.Spreadsheets.Get(sheetID).Do()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Unable to access spreadsheet: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Successfully connected to Google Sheets!")
		} else {
			fmt.Println("⚠️ GOOGLE_SHEETS_ID is not set in Settings. Skipping sheet verification.")
		}

		fmt.Println("\n✨ Your Google Sheets integration is healthy!")
	} else {
		fmt.Fprintln(os.Stderr, "\n❌ Authentication failed: No token returned")
		os.Exit(1)
	}
}

// readSettings reads the settings store the engine uses (WM_SETTINGS_PATH or
// the user config directory).
func readSettings() (map[string]string, string, error) {
	path, err := settingsAdapter.ResolvePath(configAdapter.LoadSettingsPath())
	if err != nil {
		return nil, "", err
	}
	store, err := settingsAdapter.Open(path)
	if err != nil {
		return nil, path, err
	}
	defer store.Close()
	values, err := store.Values(context.Background())
	return values, path, err
}
