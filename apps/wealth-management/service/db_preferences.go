package service

import (
	"apps/wealth-management/domain"
	"fmt"
)

// Preferences store (WM-SET-6..9): the app-managed Preferences tab, A Key,
// B Value, C Note. It is read like the other tabs and written one row per
// saved key; the tab is created with its header on the first save. No other
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
	if !current.TabExists {
		if err := s.client.CreateSheet(preferencesSheet, preferencesHeader); err != nil {
			return domain.Preferences{}, fmt.Errorf("create %s tab: %w", preferencesSheet, err)
		}
	}

	var appends [][]any
	for _, entry := range next.Entries {
		if !updated(update, entry.Key) {
			continue
		}
		if entry.Row == 0 {
			appends = append(appends, []any{string(entry.Key), preferenceCell(entry.Value), entry.Note})
			continue
		}
		cellRange := fmt.Sprintf("%s!A%d:B%d", preferencesSheet, entry.Row, entry.Row)
		if err := s.client.UpdateRow(cellRange, []any{string(entry.Key), preferenceCell(entry.Value)}); err != nil {
			s.invalidate(cacheKeyPreferences)
			return domain.Preferences{}, fmt.Errorf("update preference %s: %w", entry.Key, err)
		}
	}
	if len(appends) > 0 {
		if _, err := s.client.WriteRowsAtFirstEmptyRow(preferencesSheet, preferencesAnchorRange, appends); err != nil {
			s.invalidate(cacheKeyPreferences)
			return domain.Preferences{}, fmt.Errorf("write preferences: %w", err)
		}
	}
	s.invalidate(cacheKeyPreferences)
	return s.readPreferences()
}

func updated(update domain.PreferencesUpdate, key domain.PreferenceKey) bool {
	for _, item := range update.Values {
		if item.Key == key {
			return true
		}
	}
	return false
}

// preferenceCell writes a number, or "" to clear the value.
func preferenceCell(value *float64) any {
	if value == nil {
		return ""
	}
	return *value
}
