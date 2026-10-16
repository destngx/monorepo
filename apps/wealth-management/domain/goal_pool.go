package domain

import (
	"math"
	"time"
)

const (
	// savingsWindowMonths averages expense (buffer) and savings.
	savingsWindowMonths = 3
	// needsWindowMonths averages monthly needs (emergency target).
	needsWindowMonths = 6
)

// PoolRole says how an account takes part in the goal pool.
type PoolRole string

const (
	PoolRoleAsset    PoolRole = "asset"
	PoolRoleCardDebt PoolRole = "cardDebt"
	// PoolRoleCrypto is upside, counted toward goals only once converted (D32).
	PoolRoleCrypto PoolRole = "crypto"
	// PoolRoleArchived is a deprecated account, left out.
	PoolRoleArchived PoolRole = "archived"
	// PoolRoleEmpty is a zero balance.
	PoolRoleEmpty PoolRole = "empty"
)

// PoolAccount is one account's part in the pool.
type PoolAccount struct {
	Name    string   `json:"name"`
	Balance float64  `json:"balance"`
	Role    PoolRole `json:"role"`
}

// GoalPool is the pooled savings balance goals draw on (D31), with the
// parts of the sum so the UI can show the math:
// Total = max(0, Assets - CardDebt - SpendingBuffer).
type GoalPool struct {
	Assets                float64       `json:"assets"`
	CardDebt              float64       `json:"cardDebt"`
	SpendingBuffer        float64       `json:"spendingBuffer"`
	BufferMonths          float64       `json:"bufferMonths"`
	AverageMonthlyExpense float64       `json:"averageMonthlyExpense"`
	Total                 float64       `json:"total"`
	Crypto                float64       `json:"crypto"`
	Archived              float64       `json:"archived"`
	MoneyLent             float64       `json:"moneyLent"`
	Accounts              []PoolAccount `json:"accounts"`
}

// ComputeGoalPool sums positive balances of non-crypto, non-archived
// accounts, then takes off every negative balance (card debt, D17) and the
// spending buffer. Crypto, archived accounts and money lent are reported
// beside it but never counted (D32, D30).
func ComputeGoalPool(accounts []Account, transactions []Transaction, settings GoalPoolSettings, today time.Time) GoalPool {
	pool := GoalPool{BufferMonths: settings.SpendingBufferMonths, Accounts: make([]PoolAccount, 0, len(accounts))}
	for _, account := range accounts {
		role := PoolRoleAsset
		switch {
		case account.Balance < 0:
			role = PoolRoleCardDebt
			pool.CardDebt += -account.Balance
		case account.Balance == 0:
			role = PoolRoleEmpty
		case account.Type == AccountTypeDeprecated:
			role = PoolRoleArchived
			pool.Archived += account.Balance
		case isCryptoAccount(account):
			role = PoolRoleCrypto
			pool.Crypto += account.Balance
		default:
			pool.Assets += account.Balance
		}
		pool.Accounts = append(pool.Accounts, PoolAccount{Name: account.Name, Balance: account.Balance, Role: role})
	}
	pool.MoneyLent, _ = lendingPositions(transactions)
	months := windowMonths(transactions, today, savingsWindowMonths)
	expense := 0.0
	for _, month := range months {
		expense += ComputeCashFlow(transactions, month).Expense
	}
	pool.AverageMonthlyExpense = average(expense, len(months))
	pool.SpendingBuffer = math.Round(settings.SpendingBufferMonths * pool.AverageMonthlyExpense)
	pool.Total = math.Max(0, pool.Assets-pool.CardDebt-pool.SpendingBuffer)
	return pool
}

// MonthlyNeeds is the average monthly spending on needs (fixed + essential
// categories) behind the emergency fund target.
type MonthlyNeeds struct {
	PerMonth float64 `json:"perMonth"`
	Months   int     `json:"months"`
	// FromTotalExpense is set when no category is tagged fixed or essential,
	// so total expense stands in for needs.
	FromTotalExpense bool `json:"fromTotalExpense"`
}

// ComputeMonthlyNeeds averages fixed and essential expense over the last
// six completed months (fewer when the history is shorter).
func ComputeMonthlyNeeds(transactions []Transaction, categories []Category, today time.Time) MonthlyNeeds {
	tagged := false
	for _, category := range categories {
		if category.Nature == NatureFixed || category.Nature == NatureEssential {
			tagged = true
			break
		}
	}
	months := windowMonths(transactions, today, needsWindowMonths)
	total := 0.0
	for _, t := range transactions {
		if !t.Usable() || !t.CountsAsExpense() || !inMonths(EffectiveDate(t), months) {
			continue
		}
		if !tagged || t.CategoryNature == NatureFixed || t.CategoryNature == NatureEssential {
			total += t.PaymentAmount()
		}
	}
	return MonthlyNeeds{PerMonth: average(total, len(months)), Months: len(months), FromTotalExpense: !tagged}
}

// MonthlySavings is the average income - expense of recent completed
// months, never below zero (lending excluded, WM-RULE-3).
type MonthlySavings struct {
	PerMonth float64 `json:"perMonth"`
	Months   int     `json:"months"`
}

// ComputeMonthlySavings averages cash flow over the last three completed
// months (fewer when the history is shorter).
func ComputeMonthlySavings(transactions []Transaction, today time.Time) MonthlySavings {
	months := windowMonths(transactions, today, savingsWindowMonths)
	total := 0.0
	for _, month := range months {
		total += ComputeCashFlow(transactions, month).CashFlow
	}
	return MonthlySavings{PerMonth: math.Max(0, average(total, len(months))), Months: len(months)}
}

// windowMonths returns the first days of the last n completed months,
// oldest first, leaving out months before the first usable transaction so a
// short history is not averaged with empty months.
func windowMonths(transactions []Transaction, today time.Time, n int) []time.Time {
	var first time.Time
	for _, t := range transactions {
		if !t.Usable() {
			continue
		}
		if date := EffectiveDate(t); first.IsZero() || date.Before(first) {
			first = date
		}
	}
	if first.IsZero() {
		return nil
	}
	firstMonth := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
	current := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := []time.Time{}
	for i := n; i >= 1; i-- {
		if month := current.AddDate(0, -i, 0); !month.Before(firstMonth) {
			months = append(months, month)
		}
	}
	return months
}

func inMonths(date time.Time, months []time.Time) bool {
	for _, month := range months {
		if SameMonth(date, month) {
			return true
		}
	}
	return false
}

// average rounds to whole VND; zero months average to zero.
func average(total float64, months int) float64 {
	if months == 0 {
		return 0
	}
	return math.Round(total / float64(months))
}
