package domain

// CashbackRule pays Rate percent on payments tagged with Tag, up to CapMonthly per cycle.
type CashbackRule struct {
	Name       string  `json:"name"`
	Tag        string  `json:"tag"`
	Rate       float64 `json:"rate"`
	CapMonthly float64 `json:"capMonthly"`
	// Fallback rules apply to the card's spend that matches no other rule tag.
	Fallback bool `json:"fallback,omitempty"`
}

// CardProgram describes one physical card and its cashback rules.
type CardProgram struct {
	Name         string         `json:"name"`
	Bank         string         `json:"bank"`
	Keyword      string         `json:"keyword"`
	DefaultLimit float64        `json:"defaultLimit"`
	Expiry       string         `json:"expiry"`
	AnnualFee    float64        `json:"annualFee"`
	MonthlyCap   float64        `json:"monthlyCap"`
	StatementDay int            `json:"statementDay"`
	DueDay       int            `json:"dueDay"`
	Rules        []CashbackRule `json:"rules"`
}

// CardBank groups programs sharing a bank-level account and cashback cap.
type CardBank struct {
	Name string `json:"name"`
	// SharedAccount holds spend for every card at this bank; untagged spend on
	// it belongs to DefaultCardKeyword.
	SharedAccount      string        `json:"sharedAccount"`
	DefaultCardKeyword string        `json:"defaultCardKeyword"`
	SharedMonthlyCap   float64       `json:"sharedMonthlyCap"`
	Cards              []CardProgram `json:"cards"`
}

// CardConfig is the full cashback configuration.
type CardConfig struct {
	Banks []CardBank `json:"banks"`
}

// CardHistoryMonth is one month of a card's cashback efficiency (WM-CARD-2).
type CardHistoryMonth struct {
	Month             string  `json:"month"`
	Spent             float64 `json:"spent"`
	PotentialCashback float64 `json:"potentialCashback"`
	ActualRefund      float64 `json:"actualRefund"`
	Efficiency        float64 `json:"efficiency"`
	Missed            float64 `json:"missed"`
}

// CardReport is the computed state of one card.
type CardReport struct {
	Name              string          `json:"name"`
	Bank              string          `json:"bank"`
	Keyword           string          `json:"keyword"`
	AccountName       string          `json:"accountName"`
	Expiry            string          `json:"expiry"`
	Limit             float64         `json:"limit"`
	Usage             float64         `json:"usage"`
	Utilization       float64         `json:"utilization"`
	UtilizationBand   UtilizationBand `json:"utilizationBand"`
	RemainingLimit    float64         `json:"remainingLimit"`
	SpentThisCycle    float64         `json:"spentThisCycle"`
	EstimatedCashback float64         `json:"estimatedCashback"`
	RefundYTD         float64         `json:"refundYtd"`
	AnnualFee         float64         `json:"annualFee"`
	// MonthlyCap is the card's total cashback cap per month; 0 means none (WM-CARD-2).
	MonthlyCap        float64            `json:"monthlyCap"`
	NetEarnYTD        float64            `json:"netEarnYtd"`
	LifetimeFees      float64            `json:"lifetimeFees"`
	TransactionCount  int                `json:"transactionCount"`
	Cycle             CardCycle          `json:"cycle"`
	Rules             []CashbackRule     `json:"rules"`
	History           []CardHistoryMonth `json:"history"`
	CycleTransactions []Transaction      `json:"cycleTransactions"`
}

// CardAccountGroup aggregates the cards billed to one sheet account.
type CardAccountGroup struct {
	Name              string       `json:"name"`
	Shared            bool         `json:"shared"`
	Limit             float64      `json:"limit"`
	Usage             float64      `json:"usage"`
	Utilization       float64      `json:"utilization"`
	RemainingLimit    float64      `json:"remainingLimit"`
	EstimatedCashback float64      `json:"estimatedCashback"`
	RefundYTD         float64      `json:"refundYtd"`
	AnnualFees        float64      `json:"annualFees"`
	NetEarnYTD        float64      `json:"netEarnYtd"`
	Cards             []CardReport `json:"cards"`
}

// CardBankReport groups card accounts by bank.
type CardBankReport struct {
	Name     string             `json:"name"`
	Accounts []CardAccountGroup `json:"accounts"`
}
