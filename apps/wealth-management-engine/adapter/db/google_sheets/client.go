package google_sheets

import (
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsClient struct {
	service       *sheets.Service
	spreadsheetID string
}

func NewSheetsClient(ctx context.Context, config domain.SheetsConfig) (*SheetsClient, error) {
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
		return nil, err
	}

	return &SheetsClient{
		service:       service,
		spreadsheetID: config.SpreadsheetID,
	}, nil
}

func NewSheetsClientWithService(service *sheets.Service, spreadsheetID string) *SheetsClient {
	return &SheetsClient{
		service:       service,
		spreadsheetID: spreadsheetID,
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

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}
