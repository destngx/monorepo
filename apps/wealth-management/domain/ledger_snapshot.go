package domain

// LedgerSnapshot is the sheet data most reports start from, read together
// so a cold cache costs one Sheets request (WM-NFR-3). Money lent derives
// from the transactions (D30).
type LedgerSnapshot struct {
	Accounts     []Account
	Transactions []Transaction
}
