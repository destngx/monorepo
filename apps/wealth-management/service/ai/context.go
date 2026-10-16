package ai

import (
	"apps/wealth-management/domain"
	"fmt"
	"math"
	"strings"
	"time"
)

// operationalDirective is the engine's contract with every model (WM-AI-2,
// INT-006): it is code, not prompt content, because the guarantees it states
// (where numbers come from) are enforced here.
const operationalDirective = `## OPERATIONAL DIRECTIVE
- Every figure about the user's money comes from the snapshot above, the task context or tool results. Quote figures exactly; never invent, estimate or recompute ledger totals.
- If data is missing or a tool fails, say so plainly, then answer from general expertise and label it as such.
- Amounts are Vietnamese dong (VND) unless another currency is stated. Identifiers shown as •••• or [name] are masked on purpose.`

// systemPrompt composes identity + live data snapshot + directive + task (WM-AI-2).
func (s *Service) systemPrompt(p *prompts, snapshot snapshot, task string) (string, error) {
	identity, err := p.render(keyIdentity, nil)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.WriteString(identity)
	builder.WriteString("\n\n---\n\n## LIVE DATA SNAPSHOT\n")
	fmt.Fprintf(&builder, "- **Today**: %s\n", snapshot.today.Format("02/01/2006"))
	fmt.Fprintf(&builder, "- **Net worth**: %s (assets %s, liabilities %s)\n", vnd(snapshot.sheet.NetWorth), vnd(snapshot.sheet.Assets), vnd(snapshot.sheet.Liabilities))
	fmt.Fprintf(&builder, "- **Account balances**: %s\n", snapshot.accountsLine(s.masker))
	fmt.Fprintf(&builder, "- **Money lent**: %s\n", snapshot.moneyLentLine())
	fmt.Fprintf(&builder, "- **Budget this month**: %s\n", snapshot.budgetLine())
	builder.WriteString("\n---\n\n")
	builder.WriteString(operationalDirective)
	if task = strings.TrimSpace(task); task != "" {
		builder.WriteString("\n\n---\n\n## TASK-SPECIFIC CONTEXT\n")
		builder.WriteString(task)
	}
	return builder.String(), nil
}

// snapshot is the sheet state shared by every prompt of one request.
type snapshot struct {
	today        time.Time
	accounts     []domain.Account
	transactions []domain.Transaction
	lent         domain.MoneyLent
	sheet        domain.BalanceSheet
	budget       domain.BudgetReport
}

func (s *Service) loadSnapshot() (snapshot, error) {
	accounts, err := s.db.ListAccounts(false)
	if err != nil {
		return snapshot{}, fmt.Errorf("load accounts: %w", err)
	}
	transactions, err := s.db.ListTransactions(false)
	if err != nil {
		return snapshot{}, fmt.Errorf("load transactions: %w", err)
	}
	budget, err := s.analytics.BudgetReport("", false)
	if err != nil {
		return snapshot{}, fmt.Errorf("load budget: %w", err)
	}
	today := s.today()
	return snapshot{
		today:        today,
		accounts:     accounts,
		transactions: usable(transactions),
		lent:         domain.ComputeMoneyLent(transactions, today),
		sheet:        domain.ComputeBalanceSheet(accounts, transactions),
		budget:       budget,
	}, nil
}

func usable(transactions []domain.Transaction) []domain.Transaction {
	out := make([]domain.Transaction, 0, len(transactions))
	for _, t := range transactions {
		if t.Usable() {
			out = append(out, t)
		}
	}
	return out
}

func (s snapshot) accountsLine(masker domain.PIIMasker) string {
	if len(s.accounts) == 0 {
		return "no accounts"
	}
	parts := make([]string, 0, len(s.accounts))
	for _, account := range s.accounts {
		parts = append(parts, masker.Mask(account.Name)+": "+vnd(account.Balance))
	}
	return strings.Join(parts, ", ")
}

// moneyLentLine summarises money lent without naming anyone.
func (s snapshot) moneyLentLine() string {
	facts := s.lent.MoneyLentFacts
	if facts.PeopleOwing == 0 && facts.PeopleYouOwe == 0 {
		return "nothing outstanding"
	}
	line := fmt.Sprintf("owed to you %s by %s", vnd(facts.OwedToYou), people(facts.PeopleOwing))
	if facts.OldestDaysOutstanding != nil {
		line += fmt.Sprintf(" (oldest open %d days)", *facts.OldestDaysOutstanding)
	}
	if facts.PeopleYouOwe > 0 {
		line += fmt.Sprintf("; you owe %s to %s", vnd(facts.YouOwe), people(facts.PeopleYouOwe))
	}
	return line
}

func people(count int) string {
	if count == 1 {
		return "1 person"
	}
	return fmt.Sprintf("%d people", count)
}

func (s snapshot) budgetLine() string {
	parts := []string{}
	for _, line := range s.budget.Lines {
		if line.CountsAsExpense() && line.Limit > 0 {
			parts = append(parts, fmt.Sprintf("%s: %s / %s", line.Category, vnd(line.Spent), vnd(line.Limit)))
		}
	}
	if len(parts) == 0 {
		return "no budget limits set"
	}
	return fmt.Sprintf("spent %s of %s (%.0f%%). %s", vnd(s.budget.TotalSpent), vnd(s.budget.TotalLimit), s.budget.PercentUsed, strings.Join(parts, ", "))
}

// recent returns transactions dated within [from, to], newest first, at most limit (0 = all).
func recent(transactions []domain.Transaction, from time.Time, to time.Time, limit int) []domain.Transaction {
	out := []domain.Transaction{}
	for _, t := range transactions {
		if !t.Date.Before(from) && !t.Date.After(to) {
			out = append(out, t)
		}
	}
	sortNewestFirst(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// txContext is the model view of a transaction; free text is PII-masked.
type txContext struct {
	Date     string   `json:"date"`
	Account  string   `json:"account"`
	Payee    string   `json:"payee"`
	Category string   `json:"category"`
	Payment  float64  `json:"payment,omitempty"`
	Deposit  float64  `json:"deposit,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Memo     string   `json:"memo,omitempty"`
}

func (s *Service) txContexts(transactions []domain.Transaction) []txContext {
	out := make([]txContext, 0, len(transactions))
	for _, t := range transactions {
		memo := ""
		if t.Memo != nil {
			memo = s.masker.Mask(*t.Memo)
		}
		out = append(out, txContext{
			Date:     t.Date.Format(time.DateOnly),
			Account:  s.masker.Mask(t.AccountName),
			Payee:    s.masker.Mask(t.Payee),
			Category: t.Category,
			Payment:  t.PaymentAmount(),
			Deposit:  t.DepositAmount(),
			Tags:     t.Tags,
			Memo:     memo,
		})
	}
	return out
}

type accountContext struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
	Note     string  `json:"note,omitempty"`
}

func (s *Service) accountContexts(accounts []domain.Account) []accountContext {
	out := make([]accountContext, 0, len(accounts))
	for _, account := range accounts {
		note := ""
		if account.Note != nil {
			note = s.masker.Mask(*account.Note)
		}
		out = append(out, accountContext{Name: s.masker.Mask(account.Name), Type: string(account.Type), Balance: account.Balance, Currency: string(domain.CurrencyVND), Note: note})
	}
	return out
}

// lendingContext is the model view of one open lending balance; the
// person's name is PII-masked (D30).
type lendingContext struct {
	Person          string  `json:"person"`
	Outstanding     float64 `json:"outstanding"`
	Lent            float64 `json:"lent"`
	Repaid          float64 `json:"repaid"`
	LastActivity    string  `json:"lastActivity"`
	DaysOutstanding *int    `json:"daysOutstanding,omitempty"`
}

// moneyLentContext is the money-lent totals and the open balances; settled
// people are left out.
type moneyLentContext struct {
	domain.MoneyLentFacts
	Open []lendingContext `json:"open"`
}

func (s *Service) moneyLentContext(lent domain.MoneyLent) moneyLentContext {
	out := moneyLentContext{MoneyLentFacts: lent.MoneyLentFacts, Open: []lendingContext{}}
	for _, person := range lent.People {
		if person.Outstanding == 0 {
			continue
		}
		out.Open = append(out.Open, lendingContext{
			Person:          s.masker.Mask(person.Name),
			Outstanding:     person.Outstanding,
			Lent:            person.Lent,
			Repaid:          person.Repaid,
			LastActivity:    person.LastActivity,
			DaysOutstanding: person.DaysOutstanding,
		})
	}
	return out
}

// factsBlock appends engine-computed figures to an action prompt so the
// model quotes them instead of deriving its own.
func factsBlock(facts any) string {
	return "\n\nFACTS (computed by the engine; quote, do not recompute):\n" + toJSON(facts)
}

// vnd formats an amount like the UI: 1.250.000 ₫.
func vnd(value float64) string {
	rounded := int64(math.Round(value))
	sign := ""
	if rounded < 0 {
		sign = "-"
		rounded = -rounded
	}
	digits := fmt.Sprint(rounded)
	var grouped strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(digit)
	}
	return sign + grouped.String() + " ₫"
}

func amount(value float64) string {
	return fmt.Sprint(int64(math.Round(value)))
}
