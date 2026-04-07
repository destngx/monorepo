package domain

import "time"

type AccountType string

const (
	AccountTypeActiveUse      AccountType = "active use"
	AccountTypeRarelyUse      AccountType = "rarely use"
	AccountTypeLongHolding    AccountType = "long holding"
	AccountTypeDeprecated     AccountType = "deprecated"
	AccountTypeNegativeActive AccountType = "negative active use"
	AccountTypeBank           AccountType = "bank"
	AccountTypeCrypto         AccountType = "crypto"
	AccountTypeCash           AccountType = "cash"
	AccountTypeInvestment     AccountType = "investment"
)

type Currency string

const (
	CurrencyVND  Currency = "VND"
	CurrencyUSD  Currency = "USD"
	CurrencyUSDT Currency = "USDT"
)

type CategoryType string

const (
	CategoryTypeIncome    CategoryType = "income"
	CategoryTypeExpense   CategoryType = "expense"
	CategoryTypeNonBudget CategoryType = "non-budget"
)

type GoalType string

const (
	GoalTypeSavingsTarget    GoalType = "SAVINGS_TARGET"
	GoalTypePurchaseGoal     GoalType = "PURCHASE_GOAL"
	GoalTypeDebtPayoff       GoalType = "DEBT_PAYOFF"
	GoalTypeInvestmentTarget GoalType = "INVESTMENT_TARGET"
	GoalTypeIncomeGoal       GoalType = "INCOME_GOAL"
	GoalTypeNetWorth         GoalType = "NET_WORTH_MILESTONE"
)

type GoalStatus string

const (
	GoalStatusOnTrack  GoalStatus = "ON_TRACK"
	GoalStatusAtRisk   GoalStatus = "AT_RISK"
	GoalStatusOffTrack GoalStatus = "OFF_TRACK"
)

type NotificationStatus string

const (
	NotificationPending NotificationStatus = "pending"
	NotificationDone    NotificationStatus = "done"
)

type Account struct {
	Name           string      `json:"name"`
	DueDate        *string     `json:"dueDate"`
	GoalAmount     *float64    `json:"goalAmount"`
	GoalProgress   *float64    `json:"goalProgress"`
	ClearedBalance float64     `json:"clearedBalance"`
	Balance        float64     `json:"balance"`
	Type           AccountType `json:"type"`
	Currency       Currency    `json:"currency"`
	Note           *string     `json:"note"`
}

type Transaction struct {
	ID              string        `json:"id"`
	AccountName     string        `json:"accountName"`
	Date            time.Time     `json:"date"`
	ReferenceNumber *string       `json:"referenceNumber"`
	Payee           string        `json:"payee"`
	Tags            []string      `json:"tags"`
	Memo            *string       `json:"memo"`
	Category        string        `json:"category"`
	CategoryType    *CategoryType `json:"categoryType,omitempty"`
	Cleared         bool          `json:"cleared"`
	Payment         *float64      `json:"payment"`
	Deposit         *float64      `json:"deposit"`
	AccountBalance  float64       `json:"accountBalance"`
	ClearedBalance  float64       `json:"clearedBalance"`
	RunningBalance  float64       `json:"runningBalance"`
}

type TransactionCreateInput struct {
	AccountName     string   `json:"accountName"`
	Date            string   `json:"date"`
	ReferenceNumber *string  `json:"referenceNumber,omitempty"`
	Payee           string   `json:"payee"`
	Tags            []string `json:"tags"`
	Memo            *string  `json:"memo,omitempty"`
	Category        string   `json:"category"`
	Cleared         bool     `json:"cleared"`
	Payment         *float64 `json:"payment,omitempty"`
	Deposit         *float64 `json:"deposit,omitempty"`
}

type AccountCreateInput struct {
	Name     string      `json:"name"`
	Type     AccountType `json:"type"`
	Balance  *float64    `json:"balance,omitempty"`
	Currency Currency    `json:"currency,omitempty"`
	Note     *string     `json:"note,omitempty"`
}


type Category struct {
	Name string       `json:"name"`
	Type CategoryType `json:"type"`
}

type BudgetItem struct {
	Category         string             `json:"category"`
	CategoryType     *CategoryType      `json:"categoryType,omitempty"`
	MonthlyLimit     float64            `json:"monthlyLimit"`
	YearlyLimit      float64            `json:"yearlyLimit"`
	MonthlyLimits    map[string]float64 `json:"monthlyLimits,omitempty"`
	MonthlySpent     float64            `json:"monthlySpent"`
	YearlySpent      float64            `json:"yearlySpent"`
	MonthlyRemaining float64            `json:"monthlyRemaining"`
	YearlyRemaining  float64            `json:"yearlyRemaining"`
	Note             string             `json:"note,omitempty"`
}

type Milestone struct {
	Percentage float64 `json:"percentage"`
	Label      string  `json:"label"`
	AchievedAt *string `json:"achievedAt,omitempty"`
}

type Contribution struct {
	ID            string  `json:"id"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	SourceAccount string  `json:"sourceAccount"`
	Note          *string `json:"note,omitempty"`
}

type Goal struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Type             GoalType       `json:"type"`
	Emoji            string         `json:"emoji"`
	TargetAmount     float64        `json:"targetAmount"`
	CurrentAmount    float64        `json:"currentAmount"`
	Deadline         string         `json:"deadline"`
	Status           GoalStatus     `json:"status"`
	LinkedAccountID  string         `json:"linkedAccountId"`
	ContributionType string         `json:"contributionType"`
	StreakCount      int            `json:"streakCount"`
	Milestones       []Milestone    `json:"milestones"`
	History          []Contribution `json:"history"`
}

type Loan struct {
	Name             string  `json:"name"`
	MonthlyDebt      float64 `json:"monthlyDebt"`
	MonthlyPaid      float64 `json:"monthlyPaid"`
	MonthlyRemaining float64 `json:"monthlyRemaining"`
	Extra            string  `json:"extra"`
	YearlyDebt       float64 `json:"yearlyDebt"`
	YearlyPaid       float64 `json:"yearlyPaid"`
	YearlyRemaining  float64 `json:"yearlyRemaining"`
}

type EmailNotification struct {
	ID        string             `json:"id"`
	Timestamp string             `json:"timestamp"`
	From      string             `json:"from"`
	Subject   string             `json:"subject"`
	Content   string             `json:"content"`
	Status    NotificationStatus `json:"status"`
	RowNumber int                `json:"rowNumber"`
}

type InvestmentHoldingSection struct {
	Headers  []string         `json:"headers"`
	Holdings []map[string]any `json:"holdings"`
}

type InvestmentAssets struct {
	Crypto InvestmentHoldingSection `json:"crypto"`
	Funds  InvestmentHoldingSection `json:"funds"`
}

type SyncResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Deleted int    `json:"deleted,omitempty"`
}

type InitResult struct {
	Ready  bool `json:"ready"`
	Loaded struct {
		Prompts   int `json:"prompts"`
		Knowledge int `json:"knowledge"`
	} `json:"loaded"`
	Error string `json:"error,omitempty"`
}
