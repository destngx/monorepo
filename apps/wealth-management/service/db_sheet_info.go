package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"time"
)

// SheetInfo reads the spreadsheet title live, so it doubles as a connection
// test, and reports the last successful sync of this process (WM-SET-2).
func (s *dbService) SheetInfo() (domain.SheetInfo, error) {
	title, err := s.client.SpreadsheetTitle()
	if err != nil {
		return domain.SheetInfo{}, fmt.Errorf("read spreadsheet title: %w", err)
	}
	return domain.SheetInfo{SpreadsheetID: s.spreadsheetID, Title: title, LastSyncAt: s.lastSync()}, nil
}

func (s *dbService) markSynced() {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	s.lastSyncAt = s.now()
}

// lastSync is the RFC3339 time of the last successful sync, nil before one.
func (s *dbService) lastSync() *string {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.lastSyncAt.IsZero() {
		return nil
	}
	value := s.lastSyncAt.UTC().Format(time.RFC3339)
	return &value
}
