package google_sheets

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsClient struct {
	service       *sheets.Service
	spreadsheetID string
	log           *logger.Logger
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

func (c *SheetsClient) AppendRow(rangeValue string, values []any) error {
	_, err := c.service.Spreadsheets.Values.Append(c.spreadsheetID, rangeValue, &sheets.ValueRange{
		Values: [][]any{values},
	}).
		ValueInputOption("USER_ENTERED").
		Do()
	return err
}

func (c *SheetsClient) UpdateRow(rangeValue string, values []any) error {
	_, err := c.service.Spreadsheets.Values.Update(c.spreadsheetID, rangeValue, &sheets.ValueRange{
		Values: [][]any{values},
	}).
		ValueInputOption("USER_ENTERED").
		Do()
	return err
}

func (c *SheetsClient) WriteToFirstEmptyRow(sheetName string, columnRange string, values []any) error {
	rows, err := c.ReadSheet(columnRange)
	if err != nil {
		return err
	}
	lastDataIdx := -1
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		if row[0] != "" {
			lastDataIdx = i
		}
	}
	targetRow := 2 + lastDataIdx + 1
	return c.UpdateRow(fmt.Sprintf("%s!A%d", sheetName, targetRow), values)
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}
