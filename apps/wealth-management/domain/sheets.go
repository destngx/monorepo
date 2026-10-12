package domain

type SheetsConfig struct {
	ClientID      string
	ClientSecret  string
	RefreshToken  string
	SpreadsheetID string
	RedirectURL   string
	// ReadsPerMinute paces read requests to the Sheets quota (WM-NFR-3);
	// 0 uses the default of 60.
	ReadsPerMinute int
}
