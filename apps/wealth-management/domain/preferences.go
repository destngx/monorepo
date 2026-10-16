package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Preferences store (WM-SET-6..9, ROADMAP C6): an app-managed `Preferences`
// tab with A Key, B Value (a number), C Note. The engine reads it like any
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
)

const (
	// DefaultEmergencyMonths is the emergency fund target in months of needs (R3).
	DefaultEmergencyMonths = 6.0
	// DefaultSpendingBufferMonths is the spending money kept out of the goal pool (D32).
	DefaultSpendingBufferMonths = 1.0

	maxEmergencyMonths      = 24.0
	maxSpendingBufferMonths = 12.0
	// targetSumTolerance absorbs decimal targets such as 33.33 + 33.33 + 33.34.
	targetSumTolerance = 0.01
)

// PreferenceSpec describes a known key: its valid range and the note written
// next to it when the engine adds its row.
type PreferenceSpec struct {
	Key  PreferenceKey `json:"key"`
	Min  float64       `json:"min"`
	Max  float64       `json:"max"`
	Note string        `json:"note"`
	// Default is nil when an unset key means "not set" (allocation targets).
	Default *float64 `json:"default"`
}

func preferenceDefault(value float64) *float64 { return &value }

// PreferenceSpecs lists every key the engine knows, in tab order.
var PreferenceSpecs = []PreferenceSpec{
	{Key: PrefAllocationTargetCash, Min: 0, Max: 100, Note: "Target share of cash and deposits, percent; the three targets add up to 100"},
	{Key: PrefAllocationTargetFunds, Min: 0, Max: 100, Note: "Target share of fund certificates, percent"},
	{Key: PrefAllocationTargetCrypto, Min: 0, Max: 100, Note: "Target share of crypto, percent"},
	{Key: PrefGoalEmergencyMonths, Min: 1, Max: maxEmergencyMonths, Note: "Emergency fund target in months of needs", Default: preferenceDefault(DefaultEmergencyMonths)},
	{Key: PrefGoalSpendingBufferMonths, Min: 0, Max: maxSpendingBufferMonths, Note: "Months of spending kept out of the goal pool", Default: preferenceDefault(DefaultSpendingBufferMonths)},
}

// PreferenceKeyNames lists the known keys, for API schemas.
func PreferenceKeyNames() []string {
	keys := make([]string, 0, len(PreferenceSpecs))
	for _, spec := range PreferenceSpecs {
		keys = append(keys, string(spec.Key))
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

// PreferenceEntry is one row of the tab. Row is the sheet row (1-based).
type PreferenceEntry struct {
	Key   PreferenceKey `json:"key"`
	Value *float64      `json:"value"`
	Note  string        `json:"note"`
	Row   int           `json:"row"`
}

// Preferences is the parsed tab with typed, validated values. Values the tab
// does not set (or sets badly) fall back to their defaults.
type Preferences struct {
	// TabExists is false until the first save creates the tab.
	TabExists bool `json:"tabExists"`
	// AllocationTargets is nil until all three targets are set and add up to 100.
	AllocationTargets *AllocationTargets `json:"allocationTargets"`
	GoalPool          GoalPoolSettings   `json:"goalPool"`
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
		entry := PreferenceEntry{Key: key, Note: strings.TrimSpace(cell(row, 2)), Row: sheetRow}
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
		if _, ok := PreferenceSpecFor(item.Key); !ok {
			problems = append(problems, fmt.Sprintf("unknown preference key %q", item.Key))
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
