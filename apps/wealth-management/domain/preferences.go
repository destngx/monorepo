package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Preferences store (WM-SET-6..9, ROADMAP C6): an app-managed `Preferences`
// tab with A Key, B Value (a number, a JSON list of account names for
// safeToSpend.notReadyAccounts and idleCash.emergencyAccounts, or a JSON
// object of per-account minimums for accounts.minimumBalance), C Note. The engine reads it like any
// other tab and writes one row per key the user saves in the app; the tab is
// created with its header on the first save. Hand edits are tolerated: blank
// and header rows are skipped, and a bad row is reported as an issue and
// ignored, so the defaults apply instead of failing every page.

// PreferenceKey names one preference (column A).
type PreferenceKey string

const (
	PrefAllocationTargetCash     PreferenceKey = "allocation.target.cash"
	PrefAllocationTargetFunds    PreferenceKey = "allocation.target.funds"
	PrefAllocationTargetCrypto   PreferenceKey = "allocation.target.crypto"
	PrefGoalEmergencyMonths      PreferenceKey = "goal.emergencyMonths"
	PrefGoalSpendingBufferMonths PreferenceKey = "goal.spendingBufferMonths"
	// PrefSafeToSpendNotReadyAccounts lists the accounts left out of
	// safe-to-spend's ready money because they are slow to withdraw (D33).
	PrefSafeToSpendNotReadyAccounts PreferenceKey = "safeToSpend.notReadyAccounts"
	// PrefIdleCashDayToDayFloat is the day-to-day money (VND) kept ready in
	// everyday accounts before any cash counts as idle (D34); unset means
	// the spending buffer.
	PrefIdleCashDayToDayFloat PreferenceKey = "idleCash.dayToDayFloat"
	// PrefIdleCashEmergencyAccounts lists the accounts that hold the
	// emergency fund (D34): they cover the emergency target first and are
	// never idle.
	PrefIdleCashEmergencyAccounts PreferenceKey = "idleCash.emergencyAccounts"
	// PrefAccountsMinimumBalance maps an account to the balance (VND) to
	// keep in it, e.g. for a bank reward on the average monthly balance
	// (D35): that money is never ready to spend nor idle.
	PrefAccountsMinimumBalance PreferenceKey = "accounts.minimumBalance"
	// PrefIdleCashAccountProducts links an account to the flexible savings
	// product it holds (D41): its current rate is the product's live rate.
	PrefIdleCashAccountProducts PreferenceKey = "idleCash.accountProducts"
)

// PreferenceKind is the type of a key's value (column B).
type PreferenceKind string

const (
	// PreferenceNumber is a plain number within the key's range.
	PreferenceNumber PreferenceKind = "number"
	// PreferenceAccountNames is a JSON list of account names, e.g.
	// ["Bank A","Broker B"]; blank is an empty list.
	PreferenceAccountNames PreferenceKind = "accountNames"
	// PreferenceAccountMinimums is a JSON object of account name to
	// {"minimum": VND, "note": text}, e.g.
	// {"Bank A":{"minimum":5000000,"note":"monthly reward"}}; blank is none.
	PreferenceAccountMinimums PreferenceKind = "accountMinimums"
	// PreferenceAccountProducts is a JSON object of account name to
	// product ID, e.g. {"Zalopay":"zalopay:so-du-sinh-loi"}; blank is none.
	PreferenceAccountProducts PreferenceKind = "accountProducts"
)

const (
	// DefaultEmergencyMonths is the emergency fund target in months of needs (R3).
	DefaultEmergencyMonths = 6.0
	// DefaultSpendingBufferMonths is the spending money kept out of the goal pool (D32).
	DefaultSpendingBufferMonths = 1.0

	maxEmergencyMonths      = 24.0
	maxSpendingBufferMonths = 12.0
	// maxDayToDayFloat bounds the day-to-day float (VND).
	maxDayToDayFloat = 100_000_000_000.0
	// targetSumTolerance absorbs decimal targets such as 33.33 + 33.33 + 33.34.
	targetSumTolerance = 0.01
)

// PreferenceSpec describes a known key: its valid range and the note written
// next to it when the engine adds its row.
type PreferenceSpec struct {
	Key  PreferenceKey  `json:"key"`
	Kind PreferenceKind `json:"kind"`
	// Min and Max bound a number key; 0 for an account list.
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Note string  `json:"note"`
	// Default is nil when an unset key means "not set" (allocation targets).
	Default *float64 `json:"default"`
}

func preferenceDefault(value float64) *float64 { return &value }

// PreferenceSpecs lists every key the engine knows, in tab order.
var PreferenceSpecs = []PreferenceSpec{
	{Key: PrefAllocationTargetCash, Kind: PreferenceNumber, Min: 0, Max: 100, Note: "Target share of cash and deposits, percent; the three targets add up to 100"},
	{Key: PrefAllocationTargetFunds, Kind: PreferenceNumber, Min: 0, Max: 100, Note: "Target share of fund certificates, percent"},
	{Key: PrefAllocationTargetCrypto, Kind: PreferenceNumber, Min: 0, Max: 100, Note: "Target share of crypto, percent"},
	{Key: PrefGoalEmergencyMonths, Kind: PreferenceNumber, Min: 1, Max: maxEmergencyMonths, Note: "Emergency fund target in months of needs", Default: preferenceDefault(DefaultEmergencyMonths)},
	{Key: PrefGoalSpendingBufferMonths, Kind: PreferenceNumber, Min: 0, Max: maxSpendingBufferMonths, Note: "Months of spending kept out of the goal pool", Default: preferenceDefault(DefaultSpendingBufferMonths)},
	{Key: PrefSafeToSpendNotReadyAccounts, Kind: PreferenceAccountNames, Note: "Accounts not ready to use (slow to withdraw), left out of safe to spend; a JSON list of account names"},
	{Key: PrefIdleCashDayToDayFloat, Kind: PreferenceNumber, Min: 0, Max: maxDayToDayFloat, Note: "Day-to-day money (VND) kept ready in everyday accounts before any cash counts as idle; blank = the spending buffer (spending buffer months x average monthly expense)"},
	{Key: PrefIdleCashEmergencyAccounts, Kind: PreferenceAccountNames, Note: "Emergency fund accounts: they cover the emergency fund target first and are never idle; a JSON list of account names"},
	{Key: PrefIdleCashAccountProducts, Kind: PreferenceAccountProducts, Note: "Accounts that hold a flexible savings product (an e-wallet earning balance): their current rate is the product's live rate; a JSON object such as {\"Zalopay\":\"zalopay:so-du-sinh-loi\"}"},
	{Key: PrefAccountsMinimumBalance, Kind: PreferenceAccountMinimums, Max: maxAccountMinimum, Note: "Minimum balance (VND) to keep per account, e.g. for a bank reward on the average monthly balance; never ready to spend nor idle; a JSON object such as {\"Bank A\":{\"minimum\":5000000,\"note\":\"monthly reward\"}}"},
}

// PreferenceKeyNames lists the number keys, for the schemas of
// POST /api/preferences; account lists, products and minimums are set per
// account.
func PreferenceKeyNames() []string {
	keys := make([]string, 0, len(PreferenceSpecs))
	for _, spec := range PreferenceSpecs {
		if spec.Kind == PreferenceNumber {
			keys = append(keys, string(spec.Key))
		}
	}
	return keys
}

// PreferenceSpecFor returns the spec of a known key.
func PreferenceSpecFor(key PreferenceKey) (PreferenceSpec, bool) {
	for _, spec := range PreferenceSpecs {
		if spec.Key == key {
			return spec, true
		}
	}
	return PreferenceSpec{}, false
}

// AllocationTargets are the target shares of the allocation classes, in
// percent; they add up to 100.
type AllocationTargets struct {
	Cash   float64 `json:"cash"`
	Funds  float64 `json:"funds"`
	Crypto float64 `json:"crypto"`
}

// For returns the target of a target class.
func (t AllocationTargets) For(class AllocationClass) float64 {
	switch class {
	case AllocationCash:
		return t.Cash
	case AllocationFunds:
		return t.Funds
	case AllocationCrypto:
		return t.Crypto
	default:
		return 0
	}
}

// Validate checks every target is 0-100 and that they add up to 100.
func (t AllocationTargets) Validate() error {
	var problems []string
	for _, item := range []struct {
		key   PreferenceKey
		value float64
	}{{PrefAllocationTargetCash, t.Cash}, {PrefAllocationTargetFunds, t.Funds}, {PrefAllocationTargetCrypto, t.Crypto}} {
		if math.IsNaN(item.value) || item.value < 0 || item.value > 100 {
			problems = append(problems, fmt.Sprintf("%s must be between 0 and 100", item.key))
		}
	}
	if len(problems) == 0 {
		if sum := t.Cash + t.Funds + t.Crypto; math.Abs(sum-100) > targetSumTolerance {
			problems = append(problems, fmt.Sprintf("allocation targets must add up to 100 (now %s)", strconv.FormatFloat(round2(sum), 'f', -1, 64)))
		}
	}
	return validationError(problems)
}

// GoalPoolSettings drive the goal plan (R3): the emergency fund target in
// months of needs and the spending buffer kept out of the pool.
type GoalPoolSettings struct {
	EmergencyMonths      float64 `json:"emergencyMonths"`
	SpendingBufferMonths float64 `json:"spendingBufferMonths"`
}

// DefaultGoalPoolSettings are used for keys the tab does not set.
func DefaultGoalPoolSettings() GoalPoolSettings {
	return GoalPoolSettings{EmergencyMonths: DefaultEmergencyMonths, SpendingBufferMonths: DefaultSpendingBufferMonths}
}

// SafeToSpendSettings are the safe-to-spend preferences (WM-BILL-5, D33).
type SafeToSpendSettings struct {
	// NotReadyAccounts are the accounts marked not ready to use (slow to
	// withdraw), as saved; names that match no account are reported by
	// safe-to-spend, never fatal.
	NotReadyAccounts []string `json:"notReadyAccounts"`
	// ETag fingerprints the saved value (also when unset), so a change can
	// be refused when the setting changed since it was read.
	ETag string `json:"etag"`
}

// IdleCashSettings are the idle cash preferences (WM-CASH-1, D34).
type IdleCashSettings struct {
	// DayToDayFloat is the money kept ready for everyday spending; nil
	// when unset (the spending buffer is used).
	DayToDayFloat *float64 `json:"dayToDayFloat"`
	// EmergencyAccounts hold the emergency fund, as saved; names that
	// match no account are reported by the idle cash check.
	EmergencyAccounts []string `json:"emergencyAccounts"`
	// ETag fingerprints the saved emergency accounts (also when unset).
	ETag string `json:"etag"`
	// AccountProducts link accounts to flexible savings products (D41).
	AccountProducts AccountProductSettings `json:"accountProducts"`
}

// AccountMinimum is the balance to keep in one account (D35).
type AccountMinimum struct {
	Account string  `json:"account"`
	Minimum float64 `json:"minimum"`
	// Note is free text, e.g. "monthly points reward".
	Note string `json:"note,omitempty"`
}

// AccountMinimumSettings are the saved per-account minimums (D35).
type AccountMinimumSettings struct {
	// Accounts are sorted by name ignoring case; names that match no
	// account are reported by safe to spend, never fatal.
	Accounts []AccountMinimum `json:"accounts"`
	// ETag fingerprints the saved value (also when unset).
	ETag string `json:"etag"`
}

// For returns the minimum of an account, names compared ignoring case and
// surrounding spaces.
func (s AccountMinimumSettings) For(account string) (AccountMinimum, bool) {
	for _, minimum := range s.Accounts {
		if sameAccountName(minimum.Account, account) {
			return minimum, true
		}
	}
	return AccountMinimum{}, false
}

func sameAccountName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// PreferenceEntry is one row of the tab. Row is the sheet row (1-based).
type PreferenceEntry struct {
	Key PreferenceKey `json:"key"`
	// Value is the number of a number key; nil when unset, invalid or a list.
	Value *float64 `json:"value"`
	// Names is the account list of an account list key.
	Names []string `json:"names,omitempty"`
	// Minimums are the per-account minimums of accounts.minimumBalance.
	Minimums []AccountMinimum `json:"minimums,omitempty"`
	// Products are the account links of idleCash.accountProducts.
	Products []AccountProduct `json:"products,omitempty"`
	Note     string           `json:"note"`
	Row      int              `json:"row"`
	// ETag fingerprints column B as read; empty on a row not yet written.
	ETag string `json:"etag,omitempty"`
}

// Preferences is the parsed tab with typed, validated values. Values the tab
// does not set (or sets badly) fall back to their defaults.
type Preferences struct {
	// TabExists is false until the first save creates the tab.
	TabExists bool `json:"tabExists"`
	// AllocationTargets is nil until all three targets are set and add up to 100.
	AllocationTargets *AllocationTargets  `json:"allocationTargets"`
	GoalPool          GoalPoolSettings    `json:"goalPool"`
	SafeToSpend       SafeToSpendSettings `json:"safeToSpend"`
	IdleCash          IdleCashSettings    `json:"idleCash"`
	// AccountMinimums are the minimum balances to keep (D35).
	AccountMinimums AccountMinimumSettings `json:"accountMinimums"`
	// Entries are the rows of known keys, in sheet order.
	Entries []PreferenceEntry `json:"entries"`
	// Issues name rows that were ignored, e.g. a value out of range.
	Issues []string `json:"issues"`
}

// Value returns the value set for a known key, if any.
func (p Preferences) Value(key PreferenceKey) (float64, bool) {
	for _, entry := range p.Entries {
		if entry.Key == key && entry.Value != nil {
			return *entry.Value, true
		}
	}
	return 0, false
}

// preferencesHeaderKey is the A1 header; a row with it is skipped.
const preferencesHeaderKey = "key"

// ParsePreferences reads the tab rows starting at sheet row firstRow. The
// first row for a key wins (it is the one the engine updates); later
// duplicates are reported. Keys are matched case-insensitively.
func ParsePreferences(rows [][]string, firstRow int) Preferences {
	prefs := Preferences{TabExists: true, GoalPool: DefaultGoalPoolSettings(), Entries: []PreferenceEntry{}, Issues: []string{}}
	seen := map[PreferenceKey]bool{}
	for index, row := range rows {
		sheetRow := firstRow + index
		rawKey := strings.TrimSpace(cell(row, 0))
		if rawKey == "" || strings.EqualFold(rawKey, preferencesHeaderKey) {
			continue
		}
		key, ok := knownPreferenceKey(rawKey)
		if !ok {
			prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: unknown key %q, ignored", sheetRow, rawKey))
			continue
		}
		if seen[key] {
			prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %s is set more than once; the first row is used", sheetRow, key))
			continue
		}
		seen[key] = true
		entry := PreferenceEntry{Key: key, Note: strings.TrimSpace(cell(row, 2)), Row: sheetRow, ETag: preferenceETag(cell(row, 1))}
		spec, _ := PreferenceSpecFor(key)
		if spec.Kind == PreferenceAccountMinimums {
			minimums, problems, err := ParseAccountMinimums(cell(row, 1))
			if err != nil {
				prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %s %v; no minimum is set", sheetRow, key, err))
			}
			for _, problem := range problems {
				prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %s %s; that account is ignored", sheetRow, key, problem))
			}
			entry.Minimums = minimums
			prefs.Entries = append(prefs.Entries, entry)
			continue
		}
		if spec.Kind == PreferenceAccountProducts {
			links, problems, err := ParseAccountProducts(cell(row, 1))
			if err != nil {
				prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %s %v; no account is linked", sheetRow, key, err))
			}
			for _, problem := range problems {
				prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %s %s; that account is ignored", sheetRow, key, problem))
			}
			entry.Products = links
			prefs.Entries = append(prefs.Entries, entry)
			continue
		}
		if spec.Kind == PreferenceAccountNames {
			names, err := ParseAccountNames(cell(row, 1))
			if err != nil {
				prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %s %v; no account is marked", sheetRow, key, err))
			}
			entry.Names = names
			prefs.Entries = append(prefs.Entries, entry)
			continue
		}
		value, err := parsePreferenceValue(key, cell(row, 1))
		if err != nil {
			prefs.Issues = append(prefs.Issues, fmt.Sprintf("Preferences row %d: %v; the default is used", sheetRow, err))
		} else {
			entry.Value = value
		}
		prefs.Entries = append(prefs.Entries, entry)
	}
	prefs.resolve()
	return prefs
}

// resolve derives the typed values from the entries.
func (p *Preferences) resolve() {
	p.GoalPool = DefaultGoalPoolSettings()
	if value, ok := p.Value(PrefGoalEmergencyMonths); ok {
		p.GoalPool.EmergencyMonths = value
	}
	if value, ok := p.Value(PrefGoalSpendingBufferMonths); ok {
		p.GoalPool.SpendingBufferMonths = value
	}
	notReady, notReadyETag := p.accountList(PrefSafeToSpendNotReadyAccounts)
	p.SafeToSpend = SafeToSpendSettings{NotReadyAccounts: notReady, ETag: notReadyETag}
	emergency, emergencyETag := p.accountList(PrefIdleCashEmergencyAccounts)
	p.IdleCash = IdleCashSettings{EmergencyAccounts: emergency, ETag: emergencyETag}
	if value, ok := p.Value(PrefIdleCashDayToDayFloat); ok {
		p.IdleCash.DayToDayFloat = &value
	}
	p.IdleCash.AccountProducts = AccountProductSettings{Accounts: []AccountProduct{}, ETag: preferenceETag("")}
	if entry, ok := p.entry(PrefIdleCashAccountProducts); ok {
		p.IdleCash.AccountProducts.Accounts = append(p.IdleCash.AccountProducts.Accounts, entry.Products...)
		if entry.ETag != "" {
			p.IdleCash.AccountProducts.ETag = entry.ETag
		}
	}
	p.AccountMinimums = AccountMinimumSettings{Accounts: []AccountMinimum{}, ETag: preferenceETag("")}
	if entry, ok := p.entry(PrefAccountsMinimumBalance); ok {
		p.AccountMinimums.Accounts = append(p.AccountMinimums.Accounts, entry.Minimums...)
		if entry.ETag != "" {
			p.AccountMinimums.ETag = entry.ETag
		}
	}
	p.AllocationTargets = nil
	cash, hasCash := p.Value(PrefAllocationTargetCash)
	funds, hasFunds := p.Value(PrefAllocationTargetFunds)
	crypto, hasCrypto := p.Value(PrefAllocationTargetCrypto)
	switch {
	case !hasCash && !hasFunds && !hasCrypto:
	case !hasCash || !hasFunds || !hasCrypto:
		p.Issues = append(p.Issues, "Preferences: set all three allocation targets (cash, funds, crypto) or none; targets are ignored")
	default:
		targets := AllocationTargets{Cash: cash, Funds: funds, Crypto: crypto}
		if err := targets.Validate(); err != nil {
			p.Issues = append(p.Issues, fmt.Sprintf("Preferences: %s; targets are ignored", strings.TrimPrefix(err.Error(), ErrValidation.Error()+": ")))
		} else {
			p.AllocationTargets = &targets
		}
	}
}

// accountList returns an account list key's names (never nil) and the ETag
// a change must send: the row's, else the fingerprint of an empty value.
func (p Preferences) accountList(key PreferenceKey) ([]string, string) {
	names, etag := []string{}, preferenceETag("")
	if entry, ok := p.entry(key); ok {
		names = append(names, entry.Names...)
		if entry.ETag != "" {
			etag = entry.ETag
		}
	}
	return names, etag
}

// entry returns the row of a key, if any.
func (p Preferences) entry(key PreferenceKey) (PreferenceEntry, bool) {
	for _, entry := range p.Entries {
		if entry.Key == key {
			return entry, true
		}
	}
	return PreferenceEntry{}, false
}

// preferenceETag fingerprints a column B value as read (trimmed).
func preferenceETag(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])[:16]
}

// ParseAccountNames reads an account list value: blank is an empty list,
// otherwise a JSON list of strings. Names are trimmed, blanks dropped and
// repeats (ignoring case) kept once; the result is sorted ignoring case.
func ParseAccountNames(raw string) ([]string, error) {
	names := []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return names, nil
	}
	var parsed []string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return names, fmt.Errorf("value is not a JSON list of account names such as [\"Bank A\"]")
	}
	return normalizeAccountNames(parsed), nil
}

// EncodeAccountNames writes an account list for column B: a JSON list, or
// "" for no names (the cell is cleared).
func EncodeAccountNames(names []string) (string, error) {
	names = normalizeAccountNames(names)
	if len(names) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(names)
	if err != nil {
		return "", fmt.Errorf("encode account names: %w", err)
	}
	return string(encoded), nil
}

func normalizeAccountNames(raw []string) []string {
	names := []string{}
	for _, name := range raw {
		name = strings.TrimSpace(name)
		if name == "" || slices.ContainsFunc(names, func(kept string) bool { return strings.EqualFold(kept, name) }) {
			continue
		}
		names = append(names, name)
	}
	slices.SortStableFunc(names, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return names
}

// AccountReadinessInput marks one account ready to use or not (D33). ETag
// is the SafeToSpendSettings ETag read with the setting.
type AccountReadinessInput struct {
	Account string `json:"account"`
	Ready   bool   `json:"ready"`
	ETag    string `json:"etag"`
}

// Validate checks an account and an ETag are given.
func (in AccountReadinessInput) Validate() error {
	var problems []string
	if strings.TrimSpace(in.Account) == "" {
		problems = append(problems, "account is required")
	}
	if strings.TrimSpace(in.ETag) == "" {
		problems = append(problems, "etag is required")
	}
	return validationError(problems)
}

// ApplyAccountReadiness returns the preferences after marking the account
// ready (removed from the list, ignoring case) or not ready (added). It
// fails with ErrConflict when the setting changed since the ETag was read.
// A row the change adds has Row 0.
func ApplyAccountReadiness(current Preferences, input AccountReadinessInput) (Preferences, error) {
	if err := input.Validate(); err != nil {
		return Preferences{}, err
	}
	return applyAccountMark(current, PrefSafeToSpendNotReadyAccounts, input.ETag, input.Account, !input.Ready,
		"the not-ready accounts changed since they were read; reload and try again")
}

// EmergencyAccountInput marks one account as an emergency fund account or
// not (D34). ETag is the IdleCashSettings ETag read with the setting.
type EmergencyAccountInput struct {
	Account   string `json:"account"`
	Emergency bool   `json:"emergency"`
	ETag      string `json:"etag"`
}

// Validate checks an account and an ETag are given.
func (in EmergencyAccountInput) Validate() error {
	return AccountReadinessInput{Account: in.Account, ETag: in.ETag}.Validate()
}

// ApplyEmergencyAccount returns the preferences after marking the account
// as an emergency account (added) or not (removed, ignoring case). It fails
// with ErrConflict when the setting changed since the ETag was read.
func ApplyEmergencyAccount(current Preferences, input EmergencyAccountInput) (Preferences, error) {
	if err := input.Validate(); err != nil {
		return Preferences{}, err
	}
	return applyAccountMark(current, PrefIdleCashEmergencyAccounts, input.ETag, input.Account, input.Emergency,
		"the emergency accounts changed since they were read; reload and try again")
}

// applyAccountMark adds (marked) or removes the account in an account list
// key, refusing a stale ETag. A row the change adds has Row 0.
func applyAccountMark(current Preferences, key PreferenceKey, etag, account string, marked bool, conflict string) (Preferences, error) {
	names, currentETag := current.accountList(key)
	if strings.TrimSpace(etag) != currentETag {
		return Preferences{}, fmt.Errorf("%w: %s", ErrConflict, conflict)
	}
	account = strings.TrimSpace(account)
	names = slices.DeleteFunc(names, func(name string) bool { return strings.EqualFold(name, account) })
	if marked {
		names = append(names, account)
	}
	names = normalizeAccountNames(names)

	next := Preferences{TabExists: current.TabExists, Entries: slices.Clone(current.Entries), Issues: []string{}}
	index := slices.IndexFunc(next.Entries, func(entry PreferenceEntry) bool { return entry.Key == key })
	if index >= 0 {
		next.Entries[index].Names = names
	} else {
		spec, _ := PreferenceSpecFor(key)
		next.Entries = append(next.Entries, PreferenceEntry{Key: key, Names: names, Note: spec.Note})
	}
	next.resolve()
	return next, nil
}

func knownPreferenceKey(raw string) (PreferenceKey, bool) {
	for _, spec := range PreferenceSpecs {
		if strings.EqualFold(string(spec.Key), raw) {
			return spec.Key, true
		}
	}
	return "", false
}

// parsePreferenceValue parses column B: blank is "not set", otherwise a
// plain number within the key's range.
func parsePreferenceValue(key PreferenceKey, raw string) (*float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("%s value %q is not a number", key, raw)
	}
	if err := checkPreferenceRange(key, value); err != nil {
		return nil, err
	}
	return &value, nil
}

func checkPreferenceRange(key PreferenceKey, value float64) error {
	spec, ok := PreferenceSpecFor(key)
	if !ok {
		return fmt.Errorf("unknown preference key %q", key)
	}
	if math.IsNaN(value) || value < spec.Min || value > spec.Max {
		return fmt.Errorf("%s must be between %s and %s", key, formatPreferenceNumber(spec.Min), formatPreferenceNumber(spec.Max))
	}
	return nil
}

func formatPreferenceNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// PreferenceValue sets one key; a nil Value clears it (back to the default).
type PreferenceValue struct {
	Key   PreferenceKey `json:"key"`
	Value *float64      `json:"value"`
}

// PreferencesUpdate is a save from the app: one or more keys at once, so the
// three allocation targets change together.
type PreferencesUpdate struct {
	Values []PreferenceValue `json:"values"`
}

// Validate checks every key is known, appears once and is in range.
func (u PreferencesUpdate) Validate() error {
	if len(u.Values) == 0 {
		return fmt.Errorf("%w: at least one value is required", ErrValidation)
	}
	var problems []string
	seen := map[PreferenceKey]bool{}
	for _, item := range u.Values {
		spec, ok := PreferenceSpecFor(item.Key)
		if !ok {
			problems = append(problems, fmt.Sprintf("unknown preference key %q", item.Key))
			continue
		}
		if spec.Kind != PreferenceNumber {
			problems = append(problems, fmt.Sprintf("%s is set per account: use %s", item.Key, accountListRoute[item.Key]))
			continue
		}
		if seen[item.Key] {
			problems = append(problems, fmt.Sprintf("%s is given more than once", item.Key))
		}
		seen[item.Key] = true
		if item.Value != nil {
			if err := checkPreferenceRange(item.Key, *item.Value); err != nil {
				problems = append(problems, err.Error())
			}
		}
	}
	return validationError(problems)
}

// accountListRoute names where each account list key is changed.
var accountListRoute = map[PreferenceKey]string{
	PrefSafeToSpendNotReadyAccounts: "POST /api/bills/safe-to-spend/accounts (MCP SetAccountReadyToUse)",
	PrefIdleCashEmergencyAccounts:   "POST /api/bills/idle-cash/emergency-accounts (MCP SetEmergencyAccount)",
	PrefAccountsMinimumBalance:      "POST /api/bills/safe-to-spend/minimum-balance (MCP SetAccountMinimumBalance)",
	PrefIdleCashAccountProducts:     "POST /api/bills/idle-cash/account-products (MCP LinkAccountProduct)",
}

func (u PreferencesUpdate) touchesTargets() bool {
	for _, item := range u.Values {
		switch item.Key {
		case PrefAllocationTargetCash, PrefAllocationTargetFunds, PrefAllocationTargetCrypto:
			return true
		}
	}
	return false
}

// ApplyPreferencesUpdate returns the preferences after the update, failing
// when the result would leave the allocation targets incomplete or not
// adding up to 100. Rows the update adds have Row 0.
func ApplyPreferencesUpdate(current Preferences, update PreferencesUpdate) (Preferences, error) {
	if err := update.Validate(); err != nil {
		return Preferences{}, err
	}
	next := Preferences{TabExists: current.TabExists, Entries: append([]PreferenceEntry(nil), current.Entries...), Issues: []string{}}
	for _, item := range update.Values {
		index := -1
		for i, entry := range next.Entries {
			if entry.Key == item.Key {
				index = i
			}
		}
		switch {
		case index >= 0:
			next.Entries[index].Value = item.Value
		case item.Value != nil:
			spec, _ := PreferenceSpecFor(item.Key)
			next.Entries = append(next.Entries, PreferenceEntry{Key: item.Key, Value: item.Value, Note: spec.Note})
		}
	}
	// Targets are checked as a set only when the update touches one, so a
	// hand-edited target row does not block saving another key.
	cash, hasCash := next.Value(PrefAllocationTargetCash)
	funds, hasFunds := next.Value(PrefAllocationTargetFunds)
	crypto, hasCrypto := next.Value(PrefAllocationTargetCrypto)
	if update.touchesTargets() && (hasCash || hasFunds || hasCrypto) {
		if !hasCash || !hasFunds || !hasCrypto {
			return Preferences{}, fmt.Errorf("%w: set all three allocation targets (cash, funds, crypto) or clear all three", ErrValidation)
		}
		if err := (AllocationTargets{Cash: cash, Funds: funds, Crypto: crypto}).Validate(); err != nil {
			return Preferences{}, err
		}
	}
	next.resolve()
	return next, nil
}

// cell returns the cell at index, or "" past the row end.
func cell(row []string, index int) string {
	if index < len(row) {
		return row[index]
	}
	return ""
}
