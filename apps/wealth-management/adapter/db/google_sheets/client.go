package google_sheets

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// callTimeout bounds one call including quota waits and retries, so a
// starved quota or a hung connection fails instead of blocking forever.
const callTimeout = 2 * time.Minute

type SheetsClient struct {
	service       *sheets.Service
	spreadsheetID string
	log           *logger.Logger
	// writeMu serialises first-empty-row writes, which read then write (WM-DATA-4).
	writeMu sync.Mutex
	// reads paces read requests to the Sheets quota (WM-NFR-3).
	reads *tokenBucket
	retry retryPolicy
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

	client := NewSheetsClientWithService(service, config.SpreadsheetID, log)
	if config.ReadsPerMinute > 0 {
		client.reads = newTokenBucket(config.ReadsPerMinute)
	}
	return client, nil
}

// NewSheetsClientWithService uses the default read quota.
func NewSheetsClientWithService(service *sheets.Service, spreadsheetID string, log *logger.Logger) *SheetsClient {
	return &SheetsClient{
		service:       service,
		spreadsheetID: spreadsheetID,
		log:           log,
		reads:         newTokenBucket(DefaultReadsPerMinute),
		retry:         defaultRetryPolicy(),
	}
}

// read runs a read call within the quota, retrying throttling and server
// errors.
func (c *SheetsClient) read(operation string, call func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return c.retry.do(ctx, c, operation, readCall, func() error {
		if err := c.reads.Wait(ctx); err != nil {
			return err
		}
		return call(ctx)
	})
}

// write runs a write call, retrying only throttling (see callKind).
func (c *SheetsClient) write(operation string, call func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return c.retry.do(ctx, c, operation, writeCall, func() error { return call(ctx) })
}

func (c *SheetsClient) ReadSheet(rangeValue string) ([][]string, error) {
	var response *sheets.ValueRange
	err := c.read("values.get", func(ctx context.Context) error {
		var err error
		response, err = c.service.Spreadsheets.Values.Get(c.spreadsheetID, rangeValue).
			ValueRenderOption("UNFORMATTED_VALUE").
			Context(ctx).
			Do()
		return err
	})
	if err != nil {
		return nil, err
	}
	return normalizeRows(response.Values), nil
}

// ReadSheets reads several ranges with one values.batchGet request, which
// costs one request against the quota (WM-NFR-3).
func (c *SheetsClient) ReadSheets(ranges []string) ([][][]string, error) {
	if len(ranges) == 0 {
		return [][][]string{}, nil
	}
	var response *sheets.BatchGetValuesResponse
	err := c.read("values.batchGet", func(ctx context.Context) error {
		var err error
		response, err = c.service.Spreadsheets.Values.BatchGet(c.spreadsheetID).
			Ranges(ranges...).
			ValueRenderOption("UNFORMATTED_VALUE").
			Context(ctx).
			Do()
		return err
	})
	if err != nil {
		return nil, err
	}
	if len(response.ValueRanges) != len(ranges) {
		return nil, fmt.Errorf("batchGet returned %d ranges for %d requested", len(response.ValueRanges), len(ranges))
	}
	result := make([][][]string, 0, len(ranges))
	for _, valueRange := range response.ValueRanges {
		result = append(result, normalizeRows(valueRange.Values))
	}
	return result, nil
}

// normalizeRows turns cells into strings; empty cells become "".
func normalizeRows(values [][]any) [][]string {
	rows := make([][]string, 0, len(values))
	for _, row := range values {
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
	return rows
}

func (c *SheetsClient) UpdateRow(rangeValue string, values []any) error {
	return c.write("values.update", func(ctx context.Context) error {
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, rangeValue, &sheets.ValueRange{
			Values: [][]any{values},
		}).
			ValueInputOption("USER_ENTERED").
			Context(ctx).
			Do()
		return err
	})
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
	err = c.write("values.update", func(ctx context.Context) error {
		_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, fmt.Sprintf("%s!A%d", sheetName, targetRow), &sheets.ValueRange{
			Values: rows,
		}).
			ValueInputOption("USER_ENTERED").
			Context(ctx).
			Do()
		return err
	})
	if err != nil {
		return 0, err
	}
	return targetRow, nil
}

// SpreadsheetTitle is the document title (not a tab name).
func (c *SheetsClient) SpreadsheetTitle() (string, error) {
	var response *sheets.Spreadsheet
	err := c.read("spreadsheets.get", func(ctx context.Context) error {
		var err error
		response, err = c.service.Spreadsheets.Get(c.spreadsheetID).Fields("properties.title").Context(ctx).Do()
		return err
	})
	if err != nil {
		return "", err
	}
	if response.Properties == nil {
		return "", nil
	}
	return response.Properties.Title, nil
}

// SheetTitles lists the tab names of the spreadsheet.
func (c *SheetsClient) SheetTitles() ([]string, error) {
	var response *sheets.Spreadsheet
	err := c.read("spreadsheets.get", func(ctx context.Context) error {
		var err error
		response, err = c.service.Spreadsheets.Get(c.spreadsheetID).Fields("sheets.properties.title").Context(ctx).Do()
		return err
	})
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

// CreateSheet adds a tab and writes its header row.
func (c *SheetsClient) CreateSheet(title string, header []any) error {
	err := c.write("spreadsheets.batchUpdate", func(ctx context.Context) error {
		_, err := c.service.Spreadsheets.BatchUpdate(c.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{AddSheet: &sheets.AddSheetRequest{Properties: &sheets.SheetProperties{Title: title}}}},
		}).Context(ctx).Do()
		return err
	})
	if err != nil {
		return err
	}
	if len(header) == 0 {
		return nil
	}
	return c.UpdateRow(fmt.Sprintf("'%s'!A1", strings.ReplaceAll(title, "'", "''")), header)
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
