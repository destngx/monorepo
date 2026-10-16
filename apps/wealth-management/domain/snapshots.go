package domain

import (
	"fmt"
	"time"
)

// Monthly snapshots (ROADMAP R8, P3; TODO §2; WM-NW-1..3). The engine writes
// one row per month (yyyy-mm) to the app-managed `Snapshots` tab: the
// balance sheet, the allocation classes, the month's cash flow and the
// health score with its pillars, as of the month's last day (or today while
// the month runs). Writing a month that already has a row updates that row.
// Net worth history reads the rows, so it keeps the price moves that the
// back-calculated history misses.

// SnapshotMonthsBack is how far back a snapshot may be written: this month
// and last month. Older months can only be typed in by hand.
const SnapshotMonthsBack = 1

// MonthSnapshot is one row of the Snapshots tab. NetWorth is required; the
// other figures are optional so months can be backfilled by hand with the
// net worth alone.
type MonthSnapshot struct {
	// Month is yyyy-mm.
	Month string `json:"month"`
	// AsOf (yyyy-mm-dd) is the day the figures describe: the month's last
	// day, or the day it was written while the month was still running.
	AsOf string `json:"asOf"`
	// NetWorth = assets - liabilities (WM-RULE-1/2).
	NetWorth    float64  `json:"netWorth"`
	Assets      *float64 `json:"assets"`
	Liabilities *float64 `json:"liabilities"`
	// MoneyLent is what people owe the user (receivables, D30).
	MoneyLent *float64 `json:"moneyLent"`
	// Cash, Funds and Crypto are the allocation classes (WM-INV-20).
	Cash   *float64 `json:"cash"`
	Funds  *float64 `json:"funds"`
	Crypto *float64 `json:"crypto"`
	// Income, Expense and Savings (income - expense) are the month's cash
	// flow up to AsOf (WM-RULE-3); SavingsRate is percent of income.
	Income      *float64 `json:"income"`
	Expense     *float64 `json:"expense"`
	Savings     *float64 `json:"savings"`
	SavingsRate *float64 `json:"savingsRate"`
	// HealthScore and its pillars (WM-HEA-1) as of AsOf.
	HealthScore          *float64 `json:"healthScore"`
	SavingsScore         *float64 `json:"savingsScore"`
	EmergencyScore       *float64 `json:"emergencyScore"`
	DebtScore            *float64 `json:"debtScore"`
	DiversificationScore *float64 `json:"diversificationScore"`
	// EmergencyMonths is liquid assets over average monthly expense.
	EmergencyMonths *float64 `json:"emergencyMonths"`
	// WrittenOn (yyyy-mm-dd) is the day the engine wrote the row; blank on
	// rows typed by hand.
	WrittenOn string `json:"writtenOn"`
	Note      string `json:"note"`
	// Row is the sheet row; 0 for a snapshot not read from the tab.
	Row int `json:"row"`
}

// SnapshotIssue names a row left out, with the reason.
type SnapshotIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// SnapshotSheet is the parsed tab; Exists is false until the first write.
type SnapshotSheet struct {
	Exists    bool            `json:"exists"`
	Snapshots []MonthSnapshot `json:"snapshots"`
	Issues    []SnapshotIssue `json:"issues"`
}

// ForMonth returns the snapshot of a month (the first row for it).
func (s SnapshotSheet) ForMonth(month string) (MonthSnapshot, bool) {
	for _, snapshot := range s.Snapshots {
		if snapshot.Month == month {
			return snapshot, true
		}
	}
	return MonthSnapshot{}, false
}

// SnapshotSaveResult is a written snapshot read back; Created is false when
// the month's existing row was updated.
type SnapshotSaveResult struct {
	Snapshot MonthSnapshot `json:"snapshot"`
	Created  bool          `json:"created"`
}

// SnapshotInputs are the figures a snapshot is computed from. Funds and
// CryptoHoldings are the allocation's fund and crypto values (holdings
// valued at today's prices); 0 without a valuation.
type SnapshotInputs struct {
	Accounts       []Account
	Transactions   []Transaction
	Funds          float64
	CryptoHoldings float64
	Month          time.Time
	Today          time.Time
}

// SnapshotMonth parses a yyyy-mm month that may be snapshotted: this month
// or last month. Blank means the close month (CloseMonthFor).
func SnapshotMonth(value string, today time.Time) (time.Time, error) {
	month, err := closeMonth(value, today)
	if err != nil {
		return time.Time{}, err
	}
	current := firstOfMonthOf(today)
	if month.Before(current.AddDate(0, -SnapshotMonthsBack, 0)) {
		return time.Time{}, fmt.Errorf("%w: a snapshot can only be written for this month or last month (%s); older months can be typed into the Snapshots tab by hand", ErrValidation, MonthKey(current.AddDate(0, -SnapshotMonthsBack, 0)))
	}
	return month, nil
}

// SnapshotAsOf is the day a month's figures describe: its last day, or
// today while the month runs.
func SnapshotAsOf(month time.Time, today time.Time) time.Time {
	last := firstOfMonthOf(month).AddDate(0, 1, -1)
	if today.Before(last) {
		return today
	}
	return last
}

// ComputeMonthSnapshot builds a month's snapshot. For a month already over,
// account balances are taken back to its last day by undoing the flows
// booked after it (as the net worth trend does); holdings keep today's
// prices, so price moves since the month end are not undone. Rows dated after
// AsOf are left out of the money lent and the health score.
func ComputeMonthSnapshot(in SnapshotInputs) MonthSnapshot {
	month := firstOfMonthOf(in.Month)
	asOf := SnapshotAsOf(month, in.Today)
	accounts := accountsAsOf(in.Accounts, in.Transactions, asOf, in.Today)
	transactions := transactionsUpTo(in.Transactions, asOf)

	sheet := ComputeBalanceSheet(accounts, transactions)
	allocation := ComputeAllocation(AllocationInputs{
		Accounts: accounts, Transactions: transactions, Funds: in.Funds, CryptoHoldings: in.CryptoHoldings, Now: asOf,
	})
	health := ComputeFinancialHealth(accounts, transactions, allocation, asOf)
	flow := CashFlowToDay(in.Transactions, month, asOf.Day())

	snapshot := MonthSnapshot{
		Month: MonthKey(month), AsOf: asOf.Format(time.DateOnly), NetWorth: sheet.NetWorth,
		Assets: figure(sheet.Assets), Liabilities: figure(sheet.Liabilities), MoneyLent: figure(sheet.Receivables),
		Cash: figure(allocation.Line(AllocationCash).Value), Funds: figure(allocation.Line(AllocationFunds).Value),
		Crypto: figure(allocation.Line(AllocationCrypto).Value),
		Income: figure(flow.Income), Expense: figure(flow.Expense), Savings: figure(flow.CashFlow), SavingsRate: figure(flow.SavingsRate),
		HealthScore: figure(health.Score),
		WrittenOn:   in.Today.Format(time.DateOnly),
	}
	for _, component := range health.Components {
		switch component.Key {
		case "savings":
			snapshot.SavingsScore = figure(component.Score)
		case "emergency":
			snapshot.EmergencyScore = figure(component.Score)
			snapshot.EmergencyMonths = figure(component.Value)
		case "debt":
			snapshot.DebtScore = figure(component.Score)
		case "diversification":
			snapshot.DiversificationScore = figure(component.Score)
		}
	}
	return snapshot
}

// accountsAsOf takes each balance back to the end of asOf by undoing the
// flows booked after it, up to today (booking dates, as BackCalculateSeries).
func accountsAsOf(accounts []Account, transactions []Transaction, asOf time.Time, today time.Time) []Account {
	if !asOf.Before(today) {
		return accounts
	}
	later := map[string]float64{}
	for _, t := range transactions {
		if t.Usable() && t.Date.After(asOf) && !t.Date.After(today) {
			later[t.AccountName] += t.DepositAmount() - t.PaymentAmount()
		}
	}
	result := make([]Account, len(accounts))
	for i, account := range accounts {
		account.Balance -= later[account.Name]
		result[i] = account
	}
	return result
}

// transactionsUpTo keeps the rows dated on or before day.
func transactionsUpTo(transactions []Transaction, day time.Time) []Transaction {
	kept := make([]Transaction, 0, len(transactions))
	for _, t := range transactions {
		if !t.Date.After(day) {
			kept = append(kept, t)
		}
	}
	return kept
}

func firstOfMonthOf(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func figure(value float64) *float64 { return &value }
