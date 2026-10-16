package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Quick capture and drafts (§6.3). The phone sends one screenshot, with an
// optional note, to the capture inbox; the engine runs the Smart Capture
// pipeline in the background and keeps what it finds as drafts the user
// approves or rejects later. Nothing reaches the sheet without approval.

type QuickCaptureStatus string

const (
	QuickCaptureQueued     QuickCaptureStatus = "queued"
	QuickCaptureProcessing QuickCaptureStatus = "processing"
	QuickCaptureDone       QuickCaptureStatus = "done"
	// QuickCaptureFailed is final: the retries ran out or the material is
	// invalid. The user can retry or discard it.
	QuickCaptureFailed QuickCaptureStatus = "failed"
)

// QuickCaptureMaxNoteBytes bounds the note sent with a screenshot.
const QuickCaptureMaxNoteBytes = 2000

// CaptureDraftRetention is how long decided drafts are kept to recognise
// the same row in later screenshots (WM-CAP-18).
const CaptureDraftRetention = 90 * 24 * time.Hour

// quickCaptureBackoff is the wait before each retry of a failed attempt.
var quickCaptureBackoff = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour}

// QuickCaptureRetryDelay is the wait after the given number of failed
// attempts; false once the retries are used up.
func QuickCaptureRetryDelay(attempts int) (time.Duration, bool) {
	if attempts < 1 || attempts > len(quickCaptureBackoff) {
		return 0, false
	}
	return quickCaptureBackoff[attempts-1], true
}

// QuickCaptureInput is what the phone sends (WM-CAP-15).
type QuickCaptureInput struct {
	Image []byte
	Note  string
}

// Hash identifies a submission, so a resent request returns the first
// capture instead of a second one.
func (input QuickCaptureInput) Hash() string {
	sum := sha256.New()
	sum.Write(input.Image)
	sum.Write([]byte{0})
	sum.Write([]byte(strings.TrimSpace(input.Note)))
	return hex.EncodeToString(sum.Sum(nil))
}

// QuickCapture is one submission and its processing state.
type QuickCapture struct {
	ID            string             `json:"id"`
	CreatedAt     time.Time          `json:"createdAt"`
	Status        QuickCaptureStatus `json:"status"`
	Note          string             `json:"note,omitempty"`
	HasImage      bool               `json:"hasImage"`
	Attempts      int                `json:"attempts"`
	NextAttemptAt *time.Time         `json:"nextAttemptAt,omitempty"`
	// Message says what was found, or why the capture failed.
	Message   string `json:"message,omitempty"`
	NewDrafts int    `json:"newDrafts"`
	// Balances are the balances the material showed (WM-CAP-7).
	Balances []CaptureBalance `json:"-"`
}

// QuickCaptureMaterials turns a submission into pipeline materials: the
// screenshot, then the note as text, so a note such as "cash 50k coffee"
// is read like any pasted text.
func QuickCaptureMaterials(captureID string, input QuickCaptureInput, limits CaptureLimits) ([]CaptureMaterial, error) {
	note := strings.TrimSpace(input.Note)
	var problems []string
	if len(input.Image) == 0 && note == "" {
		problems = append(problems, "an image or a note is required")
	}
	if len(note) > QuickCaptureMaxNoteBytes {
		problems = append(problems, fmt.Sprintf("note: %d bytes exceeds the %d byte limit", len(note), QuickCaptureMaxNoteBytes))
	}
	if err := validationError(problems); err != nil {
		return nil, err
	}
	var materials []CaptureMaterial
	if len(input.Image) > 0 {
		mediaType := http.DetectContentType(input.Image)
		materials = append(materials, CaptureMaterial{ID: QuickCaptureImageID(captureID), Kind: CaptureImage, Name: "screenshot", MediaType: mediaType, Data: input.Image})
	}
	if note != "" {
		materials = append(materials, CaptureMaterial{ID: QuickCaptureNoteID(captureID), Kind: CaptureText, Name: "note", Text: note})
	}
	if err := ValidateCaptureMaterials(materials, limits); err != nil {
		return nil, err
	}
	return materials, nil
}

func QuickCaptureImageID(captureID string) string { return captureID + "-image" }
func QuickCaptureNoteID(captureID string) string  { return captureID + "-note" }

type CaptureDraftStatus string

const (
	CaptureDraftPending  CaptureDraftStatus = "pending"
	CaptureDraftApproved CaptureDraftStatus = "approved"
	CaptureDraftRejected CaptureDraftStatus = "rejected"
)

// CaptureDraft is a suggested transaction waiting for a decision. The
// suggestion keeps what the engine read; an approved draft also records
// the row written to the sheet.
type CaptureDraft struct {
	ID         string             `json:"id"`
	Status     CaptureDraftStatus `json:"status"`
	CreatedAt  time.Time          `json:"createdAt"`
	DecidedAt  *time.Time         `json:"decidedAt,omitempty"`
	CaptureIDs []string           `json:"captureIds"`
	Suggestion CaptureSuggestion  `json:"suggestion"`
	// TransactionID is the sheet row of an approved draft.
	TransactionID string `json:"transactionId,omitempty"`
	// BatchID is the approval batch that wrote it, so a retry of that batch
	// replays instead of conflicting.
	BatchID string `json:"-"`
}

// NewCaptureDrafts makes pending drafts from one capture's suggestions:
// draft ids replace the per-run suggestion ids, transfer groups are scoped
// to the capture, and the note is appended to the memo.
func NewCaptureDrafts(suggestions []CaptureSuggestion, captureID string, note string, now time.Time, newID func() string) []CaptureDraft {
	drafts := make([]CaptureDraft, 0, len(suggestions))
	for _, suggestion := range suggestions {
		id := newID()
		suggestion.ID = id
		if suggestion.TransferGroup != "" {
			suggestion.TransferGroup = captureID + "-" + suggestion.TransferGroup
		}
		appendCaptureNote(&suggestion, note)
		drafts = append(drafts, CaptureDraft{ID: id, Status: CaptureDraftPending, CreatedAt: now, CaptureIDs: []string{captureID}, Suggestion: suggestion})
	}
	return drafts
}

// appendCaptureNote adds the user's note to the memo once.
func appendCaptureNote(suggestion *CaptureSuggestion, note string) {
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	memo := note
	if suggestion.Transaction.Memo != nil && strings.TrimSpace(*suggestion.Transaction.Memo) != "" {
		if strings.Contains(*suggestion.Transaction.Memo, note) {
			return
		}
		memo = *suggestion.Transaction.Memo + " - " + note
	}
	suggestion.Transaction.Memo = &memo
}

// MergeCaptureDraft records that a later capture showed the draft's row
// again: its evidence, capture and note are added, the draft is otherwise
// kept.
func MergeCaptureDraft(draft *CaptureDraft, again CaptureSuggestion, captureID string, note string) {
	for _, evidence := range again.Evidence {
		known := false
		for _, existing := range draft.Suggestion.Evidence {
			known = known || existing.MaterialID == evidence.MaterialID
		}
		if !known {
			draft.Suggestion.Evidence = append(draft.Suggestion.Evidence, evidence)
		}
	}
	if !containsString(draft.CaptureIDs, captureID) {
		draft.CaptureIDs = append(draft.CaptureIDs, captureID)
	}
	appendCaptureNote(&draft.Suggestion, note)
}

// MatchCaptureDrafts finds, for each new suggestion, the earlier draft
// holding the same row, or -1 (WM-CAP-18). Overlapping screenshots of one
// history screen read the same rows, so the match is exact rather than the
// ±2 days of ledger dedup: same account, direction, amount and day, and
// the same time and reference when both sides show one. Each draft matches
// at most one suggestion, so identical rows on one day stay separate;
// matches on a reference or time are taken before day-only ones.
func MatchCaptureDrafts(suggestions []CaptureSuggestion, drafts []CaptureDraft) []int {
	matches := make([]int, len(suggestions))
	for index := range matches {
		matches[index] = -1
	}
	used := make([]bool, len(drafts))
	for _, minimum := range []int{draftMatchReference, draftMatchTime, draftMatchDay} {
		for index, suggestion := range suggestions {
			if matches[index] >= 0 {
				continue
			}
			for candidate, draft := range drafts {
				if !used[candidate] && draftMatchScore(suggestion, draft.Suggestion) >= minimum {
					matches[index] = candidate
					used[candidate] = true
					break
				}
			}
		}
	}
	return matches
}

const (
	draftMatchNone = iota
	draftMatchDay
	draftMatchTime
	draftMatchReference
)

func draftMatchScore(a, b CaptureSuggestion) int {
	if draftAccountKey(a) != draftAccountKey(b) || a.Direction != b.Direction || math.Abs(a.Amount-b.Amount) >= 0.005 {
		return draftMatchNone
	}
	dayA, timeA := splitOccurredAt(a.OccurredAt)
	dayB, timeB := splitOccurredAt(b.OccurredAt)
	if dayA == "" || dayA != dayB {
		return draftMatchNone
	}
	refA, refB := referenceText(a), referenceText(b)
	if refA != "" && refB != "" {
		if refA != refB {
			return draftMatchNone
		}
		return draftMatchReference
	}
	if timeA != "" && timeB != "" {
		if timeA != timeB {
			return draftMatchNone
		}
		return draftMatchTime
	}
	return draftMatchDay
}

// draftAccountKey is the matched account, or the account text read from
// the material when no account matched.
func draftAccountKey(suggestion CaptureSuggestion) string {
	if name := strings.TrimSpace(suggestion.Transaction.AccountName); name != "" {
		return "account:" + FoldText(name)
	}
	return "hint:" + FoldText(suggestion.AccountHint)
}

func splitOccurredAt(value string) (string, string) {
	value = strings.TrimSpace(value)
	if len(value) < 10 {
		return "", ""
	}
	return value[:10], strings.TrimSpace(value[10:])
}

func referenceText(suggestion CaptureSuggestion) string {
	if suggestion.Transaction.ReferenceNumber == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(*suggestion.Transaction.ReferenceNumber))
}

// LatestCaptureBalances keeps the newest balance each account showed
// across captures, against the current sheet balance (WM-CAP-7).
func LatestCaptureBalances(captures []QuickCapture, accounts []Account) []CaptureBalance {
	sheet := map[string]float64{}
	for _, account := range accounts {
		sheet[account.Name] = account.Balance
	}
	latest := map[string]CaptureBalance{}
	for _, capture := range captures {
		for _, balance := range capture.Balances {
			if current, seen := latest[balance.Account]; !seen || balance.AsOf > current.AsOf {
				latest[balance.Account] = balance
			}
		}
	}
	balances := make([]CaptureBalance, 0, len(latest))
	for name, balance := range latest {
		balance.SheetBalance = sheet[name]
		balances = append(balances, balance)
	}
	sort.Slice(balances, func(a, b int) bool { return balances[a].Account < balances[b].Account })
	return balances
}

// QuickCaptureOverview is the capture list with the number of drafts
// waiting for review, which the nav shows.
type QuickCaptureOverview struct {
	Captures      []QuickCapture `json:"captures"`
	PendingDrafts int            `json:"pendingDrafts"`
}

// CaptureDrafts is the drafts page: pending drafts, the materials they
// came from, and the balance hints of their captures.
type CaptureDrafts struct {
	Drafts    []CaptureDraft         `json:"drafts"`
	Materials []CaptureMaterialEntry `json:"materials"`
	Balances  []CaptureBalance       `json:"balances"`
}

// CaptureMaterialEntry describes a stored material; images are fetched
// separately, notes are inline.
type CaptureMaterialEntry struct {
	ID        string              `json:"id"`
	CaptureID string              `json:"captureId"`
	Kind      CaptureMaterialKind `json:"kind"`
	Name      string              `json:"name"`
	MediaType string              `json:"mediaType,omitempty"`
	Text      string              `json:"text,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
}

// CaptureDraftApproval writes drafts to the sheet through the batch create
// path; retrying the same BatchID cannot write twice (WM-TX-10).
type CaptureDraftApproval struct {
	BatchID string                    `json:"batchId"`
	Drafts  []CaptureDraftApprovalRow `json:"drafts"`
}

// CaptureDraftApprovalRow is one draft with the row as the user edited it.
type CaptureDraftApprovalRow struct {
	ID          string                 `json:"id"`
	Transaction TransactionCreateInput `json:"transaction"`
}

func (approval CaptureDraftApproval) Validate() error {
	var problems []string
	if strings.TrimSpace(approval.BatchID) == "" {
		problems = append(problems, "batchId is required")
	}
	if len(approval.Drafts) == 0 {
		problems = append(problems, "drafts must not be empty")
	}
	if len(approval.Drafts) > MaxTransactionBatch {
		problems = append(problems, fmt.Sprintf("at most %d drafts per approval", MaxTransactionBatch))
	}
	seen := map[string]bool{}
	for index, row := range approval.Drafts {
		id := strings.TrimSpace(row.ID)
		switch {
		case id == "":
			problems = append(problems, fmt.Sprintf("draft %d: id is required", index+1))
		case seen[id]:
			problems = append(problems, fmt.Sprintf("draft %d: duplicate id %q", index+1, id))
		}
		seen[id] = true
	}
	return validationError(problems)
}

// CaptureDraftDecision moves a pending draft to approved or rejected.
type CaptureDraftDecision struct {
	ID            string
	Status        CaptureDraftStatus
	TransactionID string
	BatchID       string
	// Transaction is the row as written, for an approval.
	Transaction *TransactionCreateInput
}
