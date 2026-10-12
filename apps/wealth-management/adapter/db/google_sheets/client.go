package google_sheets

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsClient struct {
	service       *sheets.Service
	spreadsheetID string
	log           *logger.Logger
	// writeMu serialises first-empty-row writes, which read then write (WM-DATA-4).
	writeMu sync.Mutex
}

func NewSheetsClient(ctx context.Context, config domain.SheetsConfig, log *logger.Logger) (*SheetsClient, error) {
	log.LogApplicationEvent(ctx, "initializing google sheets client",
		slog.String("spreadsheet_id", config.SpreadsheetID),
		slog.String("component", "google_sheets"),
	)

	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{sheets.SpreadsheetsScope},
		Endpoint:     google.Endpoint,
	}

	tokenSource := oauthConfig.TokenSource(ctx, &oauth2.Token{RefreshToken: config.RefreshToken})
	httpClient := oauth2.NewClient(ctx, tokenSource)

	service, err := sheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		log.LogError(ctx, "failed to create google sheets service", err,
			slog.String("component", "google_sheets"),
		)
		return nil, err
	}

	log.LogApplicationEvent(ctx, "google sheets client initialized successfully",
		slog.String("spreadsheet_id", config.SpreadsheetID),
		slog.String("component", "google_sheets"),
	)

	return &SheetsClient{
		service:       service,
		spreadsheetID: config.SpreadsheetID,
		log:           log,
	}, nil
}

func NewSheetsClientWithService(service *sheets.Service, spreadsheetID string, log *logger.Logger) *SheetsClient {
	return &SheetsClient{
		service:       service,
		spreadsheetID: spreadsheetID,
		log:           log,
	}
}

func (c *SheetsClient) ReadSheet(rangeValue string) ([][]string, error) {
	response, err := c.service.Spreadsheets.Values.Get(c.spreadsheetID, rangeValue).
		ValueRenderOption("UNFORMATTED_VALUE").
		Do()
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(response.Values))
	for _, row := range response.Values {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			if cell == nil {
				cells = append(cells, "")
				continue
			}
			cells = append(cells, toString(cell))
		}
		rows = append(rows, cells)
	}

	return rows, nil
}

func (c *SheetsClient) UpdateRow(rangeValue string, values []any) error {
	_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, rangeValue, &sheets.ValueRange{
		Values: [][]any{values},
	}).
		ValueInputOption("USER_ENTERED").
		Do()
	return err
}

// WriteRowsAtFirstEmptyRow writes rows below the last non-empty cell of the
// anchor column and returns the first sheet row written. The sheet keeps
// formulas in later columns of empty rows, so rows are written in place
// rather than appended.
func (c *SheetsClient) WriteRowsAtFirstEmptyRow(sheetName string, anchorRange string, rows [][]any) (int, error) {
	if len(rows) == 0 {
		return 0, fmt.Errorf("no rows to write")
	}
	startRow, err := anchorStartRow(anchorRange)
	if err != nil {
		return 0, err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	existing, err := c.ReadSheet(anchorRange)
	if err != nil {
		return 0, err
	}
	lastDataIdx := -1
	for i, row := range existing {
		if len(row) > 0 && row[0] != "" {
			lastDataIdx = i
		}
	}
	targetRow := startRow + lastDataIdx + 1
	_, err = c.service.Spreadsheets.Values.Update(c.spreadsheetID, fmt.Sprintf("%s!A%d", sheetName, targetRow), &sheets.ValueRange{
		Values: rows,
	}).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return 0, err
	}
	return targetRow, nil
}

// SheetTitles lists the tab names of the spreadsheet.
func (c *SheetsClient) SheetTitles() ([]string, error) {
	response, err := c.service.Spreadsheets.Get(c.spreadsheetID).Fields("sheets.properties.title").Do()
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(response.Sheets))
	for _, sheet := range response.Sheets {
		if sheet.Properties != nil {
			titles = append(titles, sheet.Properties.Title)
		}
	}
	return titles, nil
}

var anchorRowPattern = regexp.MustCompile(`![A-Za-z]+(\d+)`)

func anchorStartRow(anchorRange string) (int, error) {
	match := anchorRowPattern.FindStringSubmatch(anchorRange)
	if match == nil {
		return 0, fmt.Errorf("anchor range %q has no start row", anchorRange)
	}
	return strconv.Atoi(match[1])
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}
