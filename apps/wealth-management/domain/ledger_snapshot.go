package domain

// LedgerSnapshot is the sheet data most reports start from, read together
// so a cold cache costs one Sheets request (WM-NFR-3).
type LedgerSnapshot struct {
	Accounts     []Account
	Loans        []Loan
	Transactions []Transaction
}
