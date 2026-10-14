// Package e2esheet makes and removes the disposable spreadsheets the live
// test harness runs against (REQUIREMENTS §14.3, WM-TEST-1..6). Each one is
// uploaded from a local .xlsx export of the workbook and converted to a
// Google Sheet, so the harness never needs access to the production sheet.
package e2esheet

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	// DriveFileScope only reaches files this OAuth client created, so the
	// test token can never see or delete anything else in the account.
	DriveFileScope   = "https://www.googleapis.com/auth/drive.file"
	spreadsheetsMime = "application/vnd.google-apps.spreadsheet"
	xlsxMime         = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	// StaleAfter is how old a leftover copy must be before the janitor
	// deletes it (WM-TEST-6); younger ones may belong to a running test.
	StaleAfter = 24 * time.Hour
)

// Credentials are the OAuth client and the test-only refresh token minted
// by Connect test account in Settings (WM-TEST-3).
type Credentials struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// Copy is one disposable spreadsheet.
type Copy struct {
	ID   string
	Name string
}

// Drive uploads and deletes test spreadsheets. Every delete re-checks that
// the file is a wm-e2e- copy and not the production spreadsheet.
type Drive struct {
	files        *drive.FilesService
	productionID string
	now          func() time.Time
}

// New authenticates with the test refresh token. productionID is optional:
// when set, a file with that id is never accepted or deleted.
func New(ctx context.Context, credentials Credentials, productionID string) (*Drive, error) {
	if credentials.ClientID == "" || credentials.ClientSecret == "" || credentials.RefreshToken == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_TEST_REFRESH_TOKEN are required")
	}
	config := &oauth2.Config{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{DriveFileScope},
	}
	client := oauth2.NewClient(ctx, config.TokenSource(ctx, &oauth2.Token{RefreshToken: credentials.RefreshToken}))
	return NewWithOptions(ctx, productionID, option.WithHTTPClient(client))
}

// NewWithOptions builds the Drive client from explicit options (tests).
func NewWithOptions(ctx context.Context, productionID string, options ...option.ClientOption) (*Drive, error) {
	service, err := drive.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("drive client: %w", err)
	}
	return &Drive{files: service.Files, productionID: strings.TrimSpace(productionID), now: time.Now}, nil
}

// Upload converts the .xlsx workbook at path into a new Google Sheet named
// "wm-e2e-<runID>-<time>". Tabs, formulas, Prompts and Knowledge come along.
func (d *Drive) Upload(ctx context.Context, runID string, path string) (Copy, error) {
	if !strings.EqualFold(filepath.Ext(path), ".xlsx") {
		return Copy{}, fmt.Errorf("test workbook %q must be an .xlsx file", path)
	}
	workbook, err := os.Open(path)
	if err != nil {
		return Copy{}, fmt.Errorf("open the test workbook: %w", err)
	}
	defer workbook.Close()
	name := domain.TestSheetName(runID, d.now())
	file, err := d.files.Create(&drive.File{Name: name, MimeType: spreadsheetsMime}).
		Media(workbook, googleapi.ContentType(xlsxMime)).
		Fields("id", "name").Context(ctx).Do()
	if err != nil {
		return Copy{}, fmt.Errorf("upload the test workbook: %w", err)
	}
	created := Copy{ID: file.Id, Name: file.Name}
	// WM-TEST-2: never hand the harness anything but a fresh test sheet.
	if err := domain.CheckTestSheet(created.ID, created.Name, d.productionID); err != nil {
		return Copy{}, err
	}
	return created, nil
}

// Delete removes a copy after checking, on Drive, that it still is one.
func (d *Drive) Delete(ctx context.Context, id string) error {
	file, err := d.files.Get(id).Fields("id", "name").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("look up %s before deleting: %w", id, err)
	}
	if err := domain.CheckTestSheet(file.Id, file.Name, d.productionID); err != nil {
		return err
	}
	if err := d.files.Delete(id).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
		return fmt.Errorf("delete %s: %w", file.Name, err)
	}
	return nil
}

// SweepStale deletes this account's wm-e2e- spreadsheets older than
// StaleAfter (WM-TEST-6) and returns their names.
func (d *Drive) SweepStale(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("name contains '%s' and mimeType = '%s' and trashed = false and 'me' in owners", domain.TestSheetPrefix, spreadsheetsMime)
	cutoff := d.now().Add(-StaleAfter)
	var stale []Copy
	err := d.files.List().Q(query).Fields("nextPageToken", "files(id,name,createdTime)").PageSize(100).Context(ctx).Pages(ctx, func(page *drive.FileList) error {
		for _, file := range page.Files {
			created, err := time.Parse(time.RFC3339, file.CreatedTime)
			if err != nil || !created.Before(cutoff) || domain.CheckTestSheet(file.Id, file.Name, d.productionID) != nil {
				continue
			}
			stale = append(stale, Copy{ID: file.Id, Name: file.Name})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list leftover test copies: %w", err)
	}
	var deleted []string
	var failures []error
	for _, file := range stale {
		if err := d.Delete(ctx, file.ID); err != nil {
			failures = append(failures, err)
			continue
		}
		deleted = append(deleted, file.Name)
	}
	return deleted, errors.Join(failures...)
}
