package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var envCandidates = []string{
	"../../.env.local",
	".env.local",
}

func main() {
	for _, path := range envCandidates {
		_ = godotenv.Overload(path)
	}

	fmt.Println("--- Google Sheets Auth Verification (Go Engine) ---")

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	refreshToken := strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN"))
	sheetID := strings.TrimSpace(os.Getenv("GOOGLE_SHEETS_ID"))

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		fmt.Fprintln(os.Stderr, "❌ Missing credentials in .env.local")
		fmt.Println("Please run `bunx nx run wealth-management-engine:run --args=\"cmd/google_oauth_setup/main.go\"` to configure them.")
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
			fmt.Println("   - Run google_oauth_setup to get a new token.")
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
			fmt.Println("⚠️ GOOGLE_SHEETS_ID not found in .env.local. Skipping sheet verification.")
		}

		fmt.Println("\n✨ Your Google Sheets integration is healthy!")
	} else {
		fmt.Fprintln(os.Stderr, "\n❌ Authentication failed: No token returned")
		os.Exit(1)
	}
}
