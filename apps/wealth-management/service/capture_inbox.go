package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// CaptureReader reads materials into suggestions: the Smart Capture
// pipeline of the AI service (§6.2).
type CaptureReader interface {
	Capture(ctx context.Context, request port.CaptureRequest, progress func(port.CaptureProgress)) (domain.CaptureResult, error)
}

// CaptureLedger is the part of the sheet quick capture reads and writes.
type CaptureLedger interface {
	ListAccounts(forceFresh bool) ([]domain.Account, error)
	ListTransactions(forceFresh bool) ([]domain.Transaction, error)
	CreateTransactions(input domain.TransactionBatchInput) (domain.TransactionBatchResult, error)
}

type CaptureInboxOptions struct {
	Limits domain.CaptureLimits
	// Poll is how often the worker looks for retries that came due.
	Poll time.Duration
	// Now and NewID default to the clock and random ids; tests set them.
	Now   func() time.Time
	NewID func() string
	// Learner learns from approved drafts and rejected duplicates
	// (WM-CAP-23); nil learns nothing.
	Learner CaptureLearner
}

const (
	captureListLimit  = 50
	capturePurgeEvery = 6 * time.Hour
)

// CaptureInbox is quick capture (§6.3): Submit stores a capture at once, Run
// reads queued captures into drafts in the background, and the user approves
// or rejects drafts later. Material content is never logged.
type CaptureInbox struct {
	store   port.CaptureDraftStore
	reader  CaptureReader
	ledger  CaptureLedger
	learner CaptureLearner
	log     *logger.Logger
	limits  domain.CaptureLimits
	poll    time.Duration
	now     func() time.Time
	newID   func() string
	wake    chan struct{}
}

func NewCaptureInbox(store port.CaptureDraftStore, reader CaptureReader, ledger CaptureLedger, log *logger.Logger, options CaptureInboxOptions) *CaptureInbox {
	inbox := &CaptureInbox{
		store: store, reader: reader, ledger: ledger, learner: options.Learner, log: log,
		limits: options.Limits, poll: options.Poll, now: options.Now, newID: options.NewID,
		wake: make(chan struct{}, 1),
	}
	if inbox.limits.MaxItems == 0 {
		inbox.limits = domain.DefaultCaptureLimits()
	}
	if inbox.poll <= 0 {
		inbox.poll = 30 * time.Second
	}
	if inbox.now == nil {
		inbox.now = time.Now
	}
	if inbox.newID == nil {
		inbox.newID = rand.Text
	}
	return inbox
}

var _ port.CaptureInboxService = (*CaptureInbox)(nil)

func (s *CaptureInbox) Submit(ctx context.Context, input domain.QuickCaptureInput) (domain.QuickCapture, bool, error) {
	if _, err := domain.QuickCaptureMaterials("capture", input, s.limits); err != nil {
		return domain.QuickCapture{}, false, err
	}
	capture := domain.QuickCapture{
		ID:        "cap_" + strings.ToLower(s.newID()),
		CreatedAt: s.now().UTC(),
		Status:    domain.QuickCaptureQueued,
		Note:      strings.TrimSpace(input.Note),
	}
	stored, existed, err := s.store.AddCapture(ctx, capture, input.Hash(), input.Image)
	if err != nil {
		return domain.QuickCapture{}, false, err
	}
	if !existed {
		s.signal()
	}
	return stored, existed, nil
}

func (s *CaptureInbox) Overview(ctx context.Context) (domain.QuickCaptureOverview, error) {
	captures, err := s.store.ListCaptures(ctx, captureListLimit)
	if err != nil {
		return domain.QuickCaptureOverview{}, err
	}
	pending, err := s.store.ListDrafts(ctx, domain.CaptureDraftPending, time.Time{})
	if err != nil {
		return domain.QuickCaptureOverview{}, err
	}
	return domain.QuickCaptureOverview{Captures: captures, PendingDrafts: len(pending)}, nil
}

// PendingDraftCount counts the drafts waiting for approval, for the monthly
// close checklist (WM-CLOSE-2).
func (s *CaptureInbox) PendingDraftCount(ctx context.Context) (int, error) {
	pending, err := s.store.ListDrafts(ctx, domain.CaptureDraftPending, time.Time{})
	if err != nil {
		return 0, fmt.Errorf("count pending drafts: %w", err)
	}
	return len(pending), nil
}

// Retry queues a failed capture again with a fresh set of attempts.
func (s *CaptureInbox) Retry(ctx context.Context, captureID string) error {
	capture, err := s.store.GetCapture(ctx, captureID)
	if err != nil {
		return err
	}
	if capture.Status != domain.QuickCaptureFailed {
		return fmt.Errorf("%w: only a failed capture can be retried; this one is %s", domain.ErrConflict, capture.Status)
	}
	capture.Status, capture.Attempts, capture.NextAttemptAt, capture.Message = domain.QuickCaptureQueued, 0, nil, ""
	if err := s.store.UpdateCapture(ctx, capture); err != nil {
		return err
	}
	s.signal()
	return nil
}

func (s *CaptureInbox) Discard(ctx context.Context, captureID string) error {
	capture, err := s.store.GetCapture(ctx, captureID)
	if err != nil {
		return err
	}
	if capture.Status == domain.QuickCaptureProcessing {
		return fmt.Errorf("%w: the capture is being read; discard it when it finishes", domain.ErrConflict)
	}
	return s.store.DeleteCapture(ctx, captureID)
}

// Drafts returns the pending drafts with ledger dedup run again, so rows
// entered by hand since the capture show as likely recorded (WM-CAP-18),
// and rows the sheet has since been filled in past as possibly recorded
// (WM-CAP-21).
func (s *CaptureInbox) Drafts(ctx context.Context) (domain.CaptureDrafts, error) {
	drafts, err := s.store.ListDrafts(ctx, domain.CaptureDraftPending, time.Time{})
	if err != nil {
		return domain.CaptureDrafts{}, err
	}
	result := domain.CaptureDrafts{Drafts: drafts, Materials: []domain.CaptureMaterialEntry{}, Balances: []domain.CaptureBalance{}}
	if len(drafts) == 0 {
		return result, nil
	}
	transactions, err := s.ledger.ListTransactions(false)
	if err != nil {
		return domain.CaptureDrafts{}, fmt.Errorf("load transactions: %w", err)
	}
	suggestions := make([]domain.CaptureSuggestion, len(drafts))
	for index, draft := range drafts {
		suggestions[index] = draft.Suggestion
		suggestions[index].Selected, suggestions[index].Duplicate, suggestions[index].RecordedThrough = true, nil, ""
	}
	domain.ApplyCaptureDuplicates(suggestions, transactions, domain.Today(s.now()))
	var captureIDs []string
	for index := range drafts {
		drafts[index].Suggestion = suggestions[index]
		for _, id := range drafts[index].CaptureIDs {
			if !containsID(captureIDs, id) {
				captureIDs = append(captureIDs, id)
			}
		}
	}
	if result.Materials, err = s.store.Materials(ctx, captureIDs); err != nil {
		return domain.CaptureDrafts{}, err
	}
	captures := make([]domain.QuickCapture, 0, len(captureIDs))
	for _, id := range captureIDs {
		capture, err := s.store.GetCapture(ctx, id)
		if err != nil {
			return domain.CaptureDrafts{}, err
		}
		captures = append(captures, capture)
	}
	accounts, err := s.ledger.ListAccounts(false)
	if err != nil {
		return domain.CaptureDrafts{}, fmt.Errorf("load accounts: %w", err)
	}
	result.Balances = domain.LatestCaptureBalances(captures, accounts)
	return result, nil
}

func (s *CaptureInbox) Material(ctx context.Context, materialID string) (domain.CaptureMaterial, error) {
	return s.store.Material(ctx, materialID)
}

// Edit saves a pending draft's row as the user left it (WM-CAP-19);
// nothing reaches the sheet until the draft is approved.
func (s *CaptureInbox) Edit(ctx context.Context, draftID string, edit domain.CaptureDraftEdit) error {
	if strings.TrimSpace(draftID) == "" {
		return fmt.Errorf("%w: draft id is required", domain.ErrValidation)
	}
	if err := edit.Validate(); err != nil {
		return err
	}
	return s.store.EditDraft(ctx, draftID, edit)
}

// Approve writes the drafts through the batch create path (WM-TX-10) and
// marks the written ones approved. Drafts already decided are refused, so
// a draft is never written twice.
func (s *CaptureInbox) Approve(ctx context.Context, approval domain.CaptureDraftApproval) (domain.TransactionBatchResult, error) {
	if err := approval.Validate(); err != nil {
		return domain.TransactionBatchResult{}, err
	}
	ids := make([]string, len(approval.Drafts))
	for index, row := range approval.Drafts {
		ids[index] = row.ID
	}
	drafts, err := s.store.GetDrafts(ctx, ids)
	if err != nil {
		return domain.TransactionBatchResult{}, err
	}
	byID := make(map[string]domain.CaptureDraft, len(drafts))
	for _, draft := range drafts {
		byID[draft.ID] = draft
	}
	var problems []string
	for _, id := range ids {
		draft, ok := byID[id]
		switch {
		case !ok:
			return domain.TransactionBatchResult{}, fmt.Errorf("%w: draft %s", domain.ErrNotFound, id)
		case draft.Status == domain.CaptureDraftRejected:
			problems = append(problems, fmt.Sprintf("draft %s was rejected", id))
		case draft.Status == domain.CaptureDraftApproved && draft.BatchID != approval.BatchID:
			problems = append(problems, fmt.Sprintf("draft %s was already approved as %s", id, draft.TransactionID))
		}
	}
	if len(problems) > 0 {
		return domain.TransactionBatchResult{}, fmt.Errorf("%w: %s", domain.ErrConflict, strings.Join(problems, "; "))
	}

	input := domain.TransactionBatchInput{BatchID: approval.BatchID, Transactions: make([]domain.TransactionCreateInput, len(approval.Drafts))}
	for index, row := range approval.Drafts {
		input.Transactions[index] = row.Transaction
	}
	result, err := s.ledger.CreateTransactions(input)
	if err != nil {
		return domain.TransactionBatchResult{}, err
	}
	var decisions []domain.CaptureDraftDecision
	var memories []domain.CaptureMemory
	now := s.now().UTC()
	for _, row := range result.Rows {
		if row.Error != "" || row.ID == "" || row.Index < 0 || row.Index >= len(approval.Drafts) {
			continue
		}
		id := approval.Drafts[row.Index].ID
		written := approval.Drafts[row.Index].Transaction
		decisions = append(decisions, domain.CaptureDraftDecision{ID: id, Status: domain.CaptureDraftApproved, TransactionID: row.ID, BatchID: approval.BatchID, Transaction: &written})
		if memory, ok := domain.NewCaptureMemory(byID[id].Suggestion, written, row.ID, domain.CaptureMemoryApproved, now, memoryID(id)); ok {
			memories = append(memories, memory)
		}
	}
	if err := s.store.DecideDrafts(ctx, decisions, now); err != nil {
		// The rows are in the sheet; approving again with the same batch id
		// replays the batch and retries this step without writing twice.
		return domain.TransactionBatchResult{}, fmt.Errorf("transactions written but drafts not updated, approve again to finish: %w", err)
	}
	s.learn(ctx, memories)
	return result, nil
}

// memoryID ties a draft's memory to the draft, so approving it again
// (a replayed batch) stores nothing twice.
func memoryID(draftID string) string { return "mem_" + draftID }

// learn records memories after drafts are decided. The decision already
// stands, so a failure is logged rather than undoing it.
func (s *CaptureInbox) learn(ctx context.Context, memories []domain.CaptureMemory) {
	if s.learner == nil || len(memories) == 0 {
		return
	}
	if _, err := s.learner.Learn(ctx, memories); err != nil {
		s.log.LogError(ctx, "capture inbox: learning from decided drafts failed", err)
	}
}

// duplicateMemories are what rejected drafts teach: a draft rejected while
// it is a likely duplicate of a sheet row (WM-CAP-6) names its counterparty
// with that row's payee. Dedup runs again on the current sheet, as the
// drafts page shows it.
func (s *CaptureInbox) duplicateMemories(ctx context.Context, drafts []domain.CaptureDraft) []domain.CaptureMemory {
	if s.learner == nil || len(drafts) == 0 {
		return nil
	}
	transactions, err := s.ledger.ListTransactions(false)
	if err != nil {
		s.log.LogError(ctx, "capture inbox: learning from rejected duplicates skipped", fmt.Errorf("load transactions: %w", err))
		return nil
	}
	rows := make(map[string]domain.Transaction, len(transactions))
	for _, row := range transactions {
		rows[row.ID] = row
	}
	suggestions := make([]domain.CaptureSuggestion, len(drafts))
	for index, draft := range drafts {
		suggestions[index] = draft.Suggestion
		suggestions[index].Duplicate, suggestions[index].RecordedThrough = nil, ""
	}
	domain.ApplyCaptureDuplicates(suggestions, transactions, domain.Today(s.now()))
	now := s.now().UTC()
	var memories []domain.CaptureMemory
	for index, suggestion := range suggestions {
		if suggestion.Duplicate == nil || !suggestion.Duplicate.Likely {
			continue
		}
		row, found := rows[suggestion.Duplicate.TransactionID]
		if !found {
			continue
		}
		recorded := domain.TransactionCreateInput{AccountName: row.AccountName, Date: row.Date.Format(domain.SheetDateLayout), Payee: row.Payee, Category: row.Category, Tags: row.Tags}
		if memory, ok := domain.NewCaptureMemory(drafts[index].Suggestion, recorded, row.ID, domain.CaptureMemoryDuplicate, now, memoryID(drafts[index].ID)); ok {
			memories = append(memories, memory)
		}
	}
	return memories
}

func (s *CaptureInbox) Reject(ctx context.Context, draftIDs []string) error {
	if len(draftIDs) == 0 {
		return fmt.Errorf("%w: no drafts to reject", domain.ErrValidation)
	}
	drafts, err := s.store.GetDrafts(ctx, draftIDs)
	if err != nil {
		return err
	}
	decisions := make([]domain.CaptureDraftDecision, 0, len(drafts))
	for _, draft := range drafts {
		if draft.Status == domain.CaptureDraftApproved {
			return fmt.Errorf("%w: draft %s was already approved", domain.ErrConflict, draft.ID)
		}
		decisions = append(decisions, domain.CaptureDraftDecision{ID: draft.ID, Status: domain.CaptureDraftRejected})
	}
	if len(drafts) != len(uniqueIDs(draftIDs)) {
		return fmt.Errorf("%w: some drafts do not exist", domain.ErrNotFound)
	}
	memories := s.duplicateMemories(ctx, pendingOnly(drafts))
	if err := s.store.DecideDrafts(ctx, decisions, s.now().UTC()); err != nil {
		return err
	}
	s.learn(ctx, memories)
	return nil
}

func pendingOnly(drafts []domain.CaptureDraft) []domain.CaptureDraft {
	var pending []domain.CaptureDraft
	for _, draft := range drafts {
		if draft.Status == domain.CaptureDraftPending {
			pending = append(pending, draft)
		}
	}
	return pending
}

// Run reads queued captures until ctx ends. Captures left processing by a
// stopped engine are queued again first.
func (s *CaptureInbox) Run(ctx context.Context) {
	if err := s.store.RequeueProcessing(ctx); err != nil {
		s.log.LogError(ctx, "capture inbox: requeue failed", err)
	}
	s.purge(ctx)
	poll := time.NewTicker(s.poll)
	defer poll.Stop()
	purge := time.NewTicker(capturePurgeEvery)
	defer purge.Stop()
	for {
		s.drain(ctx)
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-poll.C:
		case <-purge.C:
			s.purge(ctx)
		}
	}
}

// drain processes every capture due now, one at a time.
func (s *CaptureInbox) drain(ctx context.Context) {
	for ctx.Err() == nil {
		processed, err := s.ProcessNext(ctx)
		if err != nil {
			if ctx.Err() == nil {
				s.log.LogError(ctx, "capture inbox: processing failed", err)
			}
			return
		}
		if !processed {
			return
		}
	}
}

func (s *CaptureInbox) purge(ctx context.Context) {
	if err := s.store.Purge(ctx, s.now().UTC().Add(-domain.CaptureDraftRetention)); err != nil && ctx.Err() == nil {
		s.log.LogError(ctx, "capture inbox: purge failed", err)
	}
}

func (s *CaptureInbox) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// ProcessNext reads the oldest due capture into drafts and reports whether
// there was one. A failure of the reader is recorded on the capture; only
// store failures are returned.
func (s *CaptureInbox) ProcessNext(ctx context.Context) (bool, error) {
	capture, image, found, err := s.store.ClaimCapture(ctx, s.now().UTC())
	if err != nil || !found {
		return false, err
	}
	materials, err := domain.QuickCaptureMaterials(capture.ID, domain.QuickCaptureInput{Image: image, Note: capture.Note}, s.limits)
	if err != nil {
		return true, s.fail(ctx, capture, err)
	}

	var extractFailure string
	result, err := s.reader.Capture(ctx, port.CaptureRequest{Materials: materials}, func(event port.CaptureProgress) {
		// The pipeline serialises progress calls.
		if event.Stage == "extract" && event.Status == "error" && extractFailure == "" {
			extractFailure = event.Message
		}
	})
	if ctx.Err() != nil {
		// Shutting down: the capture is queued again on the next start.
		return true, nil
	}
	if err == nil && extractFailure != "" {
		// A provider failure on one material would lose its rows; retry the
		// whole capture instead of keeping part of it.
		err = errors.New(extractFailure)
	}
	if err != nil {
		return true, s.fail(ctx, capture, err)
	}
	return true, s.complete(ctx, capture, materials, result)
}

func (s *CaptureInbox) complete(ctx context.Context, capture domain.QuickCapture, materials []domain.CaptureMaterial, result domain.CaptureResult) error {
	now := s.now().UTC()
	existing, err := s.store.ListDrafts(ctx, "", now.Add(-domain.CaptureDraftRetention))
	if err != nil {
		return err
	}
	matches := domain.MatchCaptureDrafts(result.Suggestions, existing)
	var fresh []domain.CaptureSuggestion
	merged := map[int]bool{}
	for index, match := range matches {
		switch {
		case match < 0:
			fresh = append(fresh, result.Suggestions[index])
		case existing[match].Status == domain.CaptureDraftPending:
			domain.MergeCaptureDraft(&existing[match], result.Suggestions[index], capture.ID, capture.Note)
			merged[match] = true
		}
		// A row already approved or rejected is not offered again.
	}
	created := domain.NewCaptureDrafts(fresh, capture.ID, capture.Note, now, func() string { return "drf_" + strings.ToLower(s.newID()) })
	var updated []domain.CaptureDraft
	for index := range existing {
		if merged[index] {
			updated = append(updated, existing[index])
		}
	}

	capture.Status, capture.NextAttemptAt = domain.QuickCaptureDone, nil
	capture.NewDrafts, capture.Balances = len(created), result.Balances
	capture.Message = captureOutcome(len(result.Suggestions), len(created), len(updated), unreadableImage(result, materials))
	return s.store.CompleteCapture(ctx, capture, created, updated)
}

// captureOutcome summarises a finished capture for the capture list.
func captureOutcome(found, created, merged int, unreadable string) string {
	switch {
	case found == 0 && unreadable != "":
		return "Nothing found: " + unreadable
	case found == 0:
		return "Nothing found."
	case created == 0 && merged > 0:
		return "Already captured: every row matched an earlier draft."
	case created == 0:
		return "Already decided: every row matched an approved or rejected draft."
	}
	return ""
}

// unreadableImage is the reader's reason when the screenshot could not be
// read. A note is often context rather than a transaction, so a note read
// as nothing is not reported.
func unreadableImage(result domain.CaptureResult, materials []domain.CaptureMaterial) string {
	for _, unreadable := range result.Unreadable {
		for _, material := range materials {
			if material.ID == unreadable.MaterialID && material.Kind == domain.CaptureImage {
				return unreadable.Reason
			}
		}
	}
	return ""
}

// fail records a reader failure: retried with backoff, final once the
// retries run out or when the material itself is invalid (WM-CAP-16).
func (s *CaptureInbox) fail(ctx context.Context, capture domain.QuickCapture, cause error) error {
	capture.Attempts++
	capture.Message = cause.Error()
	delay, retry := domain.QuickCaptureRetryDelay(capture.Attempts)
	if retry && !errors.Is(cause, domain.ErrValidation) {
		next := s.now().UTC().Add(delay)
		capture.Status, capture.NextAttemptAt = domain.QuickCaptureQueued, &next
	} else {
		capture.Status, capture.NextAttemptAt = domain.QuickCaptureFailed, nil
	}
	s.log.LogError(ctx, "capture inbox: capture failed", cause,
		slog.String("capture_id", capture.ID), slog.Int("attempts", capture.Attempts), slog.String("status", string(capture.Status)))
	return s.store.UpdateCapture(ctx, capture)
}

func containsID(ids []string, id string) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func uniqueIDs(ids []string) []string {
	var unique []string
	for _, id := range ids {
		if !containsID(unique, id) {
			unique = append(unique, id)
		}
	}
	return unique
}
