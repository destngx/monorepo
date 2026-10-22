package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"strings"
)

// Preferences store (WM-SET-6..9): the app-managed Preferences tab, A Key,
// B Value (a number, a JSON list of account names, or a JSON object of
// per-account minimum balances), C Note. It is read
// like the other tabs and written one row per saved key; the tab is created with its header on the first save. No other
// tab is ever written from here.

const (
	preferencesSheet       = "Preferences"
	preferencesRange       = "Preferences!A2:C"
	preferencesAnchorRange = "Preferences!A2:A"
	preferencesFirstRow    = 2
	cacheKeyPreferences    = "preferences"
)

var preferencesHeader = []any{"Key", "Value", "Note"}

func (s *dbService) ListPreferences(forceFresh bool) (domain.Preferences, error) {
	return cachedRead(s, cacheKeyPreferences, sheetDataTTL, forceFresh, s.readPreferences)
}

// readPreferences reads the tab; a missing tab means every default.
func (s *dbService) readPreferences() (domain.Preferences, error) {
	exists, err := s.hasSheet(preferencesSheet)
	if err != nil {
		return domain.Preferences{}, err
	}
	if !exists {
		prefs := domain.ParsePreferences(nil, preferencesFirstRow)
		prefs.TabExists = false
		return prefs, nil
	}
	rows, err := s.client.ReadSheet(preferencesRange)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("read preferences: %w", err)
	}
	return domain.ParsePreferences(rows, preferencesFirstRow), nil
}

// SavePreferences writes the given keys: the row of an existing key gets its
// new value (its note is kept), a new key is appended with a note saying
// what it means. The tab is re-read under the write lock, so hand edits made
// since the last read are respected.
func (s *dbService) SavePreferences(update domain.PreferencesUpdate) (domain.Preferences, error) {
	if err := update.Validate(); err != nil {
		return domain.Preferences{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	current, err := s.readPreferences()
	if err != nil {
		return domain.Preferences{}, err
	}
	next, err := domain.ApplyPreferencesUpdate(current, update)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := s.writePreferences(current, next, func(key domain.PreferenceKey) bool { return updated(update, key) }); err != nil {
		return domain.Preferences{}, err
	}
	return s.readPreferences()
}

// SetAccountReadiness marks one account ready to use or not (D33) in the
// safeToSpend.notReadyAccounts row: re-read under the write lock, refused
// with ErrConflict when the setting changed since its ETag was read. An
// account marked not ready must exist; one marked ready need not, so a
// renamed or deleted account can be cleared from the list.
func (s *dbService) SetAccountReadiness(input domain.AccountReadinessInput) (domain.Preferences, error) {
	if err := input.Validate(); err != nil {
		return domain.Preferences{}, err
	}
	return s.markAccount(domain.PrefSafeToSpendNotReadyAccounts, input.Account, !input.Ready, func(current domain.Preferences) (domain.Preferences, error) {
		return domain.ApplyAccountReadiness(current, input)
	})
}

// SetEmergencyAccount marks one account as an emergency fund account or
// not (D34) in the idleCash.emergencyAccounts row, the same way: re-read
// under the write lock, ErrConflict on a stale ETag, the account must exist
// to be marked (not to be unmarked).
func (s *dbService) SetEmergencyAccount(input domain.EmergencyAccountInput) (domain.Preferences, error) {
	if err := input.Validate(); err != nil {
		return domain.Preferences{}, err
	}
	return s.markAccount(domain.PrefIdleCashEmergencyAccounts, input.Account, input.Emergency, func(current domain.Preferences) (domain.Preferences, error) {
		return domain.ApplyEmergencyAccount(current, input)
	})
}

// SetAccountMinimumBalance sets or removes one account's minimum balance
// to keep (D35) in the accounts.minimumBalance row, the same way: re-read
// under the write lock, ErrConflict on a stale ETag, the account must exist
// to get a minimum (not to have it removed).
func (s *dbService) SetAccountMinimumBalance(input domain.AccountMinimumInput) (domain.Preferences, error) {
	if err := input.Validate(); err != nil {
		return domain.Preferences{}, err
	}
	return s.markAccount(domain.PrefAccountsMinimumBalance, input.Account, !input.Removes(), func(current domain.Preferences) (domain.Preferences, error) {
		return domain.ApplyAccountMinimum(current, input)
	})
}

// SetAccountProduct links one account to a flexible savings product, or
// unlinks it (D41), in the idleCash.accountProducts row, the same way:
// re-read under the write lock, ErrConflict on a stale ETag, the account
// must exist to be linked (not to be unlinked).
func (s *dbService) SetAccountProduct(input domain.AccountProductInput) (domain.Preferences, error) {
	if err := input.Validate(); err != nil {
		return domain.Preferences{}, err
	}
	return s.markAccount(domain.PrefIdleCashAccountProducts, input.Account, !input.Removes(), func(current domain.Preferences) (domain.Preferences, error) {
		return domain.ApplyAccountProduct(current, input)
	})
}

// markAccount writes one per-account key: an account being added to the
// list must exist; apply computes the next preferences from the tab
// re-read under the write lock.
func (s *dbService) markAccount(key domain.PreferenceKey, account string, adding bool, apply func(domain.Preferences) (domain.Preferences, error)) (domain.Preferences, error) {
	if adding {
		accounts, err := s.ListAccounts(false)
		if err != nil {
			return domain.Preferences{}, err
		}
		if !slices.ContainsFunc(accounts, func(a domain.Account) bool {
			return strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(account))
		}) {
			return domain.Preferences{}, fmt.Errorf("%w: account %q", domain.ErrNotFound, strings.TrimSpace(account))
		}
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	current, err := s.readPreferences()
	if err != nil {
		return domain.Preferences{}, err
	}
	next, err := apply(current)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := s.writePreferences(current, next, func(changed domain.PreferenceKey) bool { return changed == key }); err != nil {
		return domain.Preferences{}, err
	}
	return s.readPreferences()
}

// writePreferences writes the rows of the changed keys: an existing row
// gets A-B rewritten (its note kept), a new key is appended with its note,
// and the tab is created with its header first when missing. The cache is
// invalidated whatever happens. Callers hold writeMu.
func (s *dbService) writePreferences(current, next domain.Preferences, changed func(domain.PreferenceKey) bool) error {
	defer s.invalidate(cacheKeyPreferences)
	if !current.TabExists {
		if err := s.client.CreateSheet(preferencesSheet, preferencesHeader); err != nil {
			return fmt.Errorf("create %s tab: %w", preferencesSheet, err)
		}
	}
	var appends [][]any
	for _, entry := range next.Entries {
		if !changed(entry.Key) {
			continue
		}
		value, err := preferenceCell(entry)
		if err != nil {
			return err
		}
		if entry.Row == 0 {
			appends = append(appends, []any{string(entry.Key), value, entry.Note})
			continue
		}
		cellRange := fmt.Sprintf("%s!A%d:B%d", preferencesSheet, entry.Row, entry.Row)
		if err := s.client.UpdateRow(cellRange, []any{string(entry.Key), value}); err != nil {
			return fmt.Errorf("update preference %s: %w", entry.Key, err)
		}
	}
	if len(appends) > 0 {
		if _, err := s.client.WriteRowsAtFirstEmptyRow(preferencesSheet, preferencesAnchorRange, appends); err != nil {
			return fmt.Errorf("write preferences: %w", err)
		}
	}
	return nil
}

func updated(update domain.PreferencesUpdate, key domain.PreferenceKey) bool {
	for _, item := range update.Values {
		if item.Key == key {
			return true
		}
	}
	return false
}

// preferenceCell is column B: the number, the account list or minimums as
// JSON, or "" to clear the value.
func preferenceCell(entry domain.PreferenceEntry) (any, error) {
	switch spec, _ := domain.PreferenceSpecFor(entry.Key); spec.Kind {
	case domain.PreferenceAccountNames:
		return domain.EncodeAccountNames(entry.Names)
	case domain.PreferenceAccountMinimums:
		return domain.EncodeAccountMinimums(entry.Minimums)
	case domain.PreferenceAccountProducts:
		return domain.EncodeAccountProducts(entry.Products)
	}
	if entry.Value == nil {
		return "", nil
	}
	return *entry.Value, nil
}
