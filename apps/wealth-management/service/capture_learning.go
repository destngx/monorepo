package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// CaptureRuleBook is the CaptureRules tab (WM-CAP-11).
type CaptureRuleBook interface {
	ListCaptureRules(forceFresh bool) ([]domain.CaptureRule, error)
	SaveCaptureRules(rules []domain.CaptureRule) (domain.CaptureRulesSaveResult, error)
}

// CaptureLearner learns from written rows (WM-CAP-23).
type CaptureLearner interface {
	Learn(ctx context.Context, memories []domain.CaptureMemory) (domain.CaptureLearnResult, error)
}

type CaptureLearningOptions struct {
	// Now and NewID default to the clock and random ids; tests set them.
	Now   func() time.Time
	NewID func() string
}

// CaptureLearning keeps capture memories in the private store and saves
// the counterparty rules they teach to CaptureRules, where the user can see
// and edit them (WM-CAP-23).
type CaptureLearning struct {
	memory port.CaptureMemoryStore
	rules  CaptureRuleBook
	now    func() time.Time
	newID  func() string
}

func NewCaptureLearning(memory port.CaptureMemoryStore, rules CaptureRuleBook, options CaptureLearningOptions) *CaptureLearning {
	learning := &CaptureLearning{memory: memory, rules: rules, now: options.Now, newID: options.NewID}
	if learning.now == nil {
		learning.now = time.Now
	}
	if learning.newID == nil {
		learning.newID = rand.Text
	}
	return learning
}

var (
	_ CaptureLearner              = (*CaptureLearning)(nil)
	_ port.CaptureLearningService = (*CaptureLearning)(nil)
)

// Learn stores the memories, then saves the counterparty rules they change.
// Rules are read fresh so an edit made in the sheet is not overwritten by a
// cached copy.
func (l *CaptureLearning) Learn(ctx context.Context, memories []domain.CaptureMemory) (domain.CaptureLearnResult, error) {
	if len(memories) == 0 {
		return domain.CaptureLearnResult{}, nil
	}
	if err := l.memory.AddMemories(ctx, memories); err != nil {
		return domain.CaptureLearnResult{}, fmt.Errorf("store capture memories: %w", err)
	}
	result := domain.CaptureLearnResult{Remembered: len(memories)}
	existing, err := l.rules.ListCaptureRules(true)
	if err != nil {
		return result, fmt.Errorf("load capture rules: %w", err)
	}
	updates := domain.CounterpartyRuleUpdates(memories, existing)
	if len(updates) == 0 {
		return result, nil
	}
	if result.Rules, err = l.rules.SaveCaptureRules(updates); err != nil {
		return result, fmt.Errorf("save counterparty rules: %w", err)
	}
	return result, nil
}

// LearnWritten learns from rows the Smart Capture page wrote. Rows with no
// counterparty, like transfers and fee rows, are skipped.
func (l *CaptureLearning) LearnWritten(ctx context.Context, rows []domain.CaptureWrittenRow) (domain.CaptureLearnResult, error) {
	if err := domain.ValidateCaptureWrittenRows(rows); err != nil {
		return domain.CaptureLearnResult{}, err
	}
	now := l.now().UTC()
	var memories []domain.CaptureMemory
	for _, row := range rows {
		id := "mem_" + strings.ToLower(l.newID())
		if memory, ok := domain.NewCaptureMemory(row.Suggestion, row.Transaction, row.TransactionID, domain.CaptureMemoryApproved, now, id); ok {
			memories = append(memories, memory)
		}
	}
	return l.Learn(ctx, memories)
}
