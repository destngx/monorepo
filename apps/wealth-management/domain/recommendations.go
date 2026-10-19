package domain

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Recommendations (ROADMAP R7, WM-REC-1..): one persistent list of advice
// the user acts on. AI reviews and deterministic engine rules add to it; the
// user marks items done, dismissed or snoozed; the next review checks the
// outcome. The list lives in the app-managed Recommendations tab.

type RecommendationStatus string

const (
	RecommendationOpen      RecommendationStatus = "open"
	RecommendationDone      RecommendationStatus = "done"
	RecommendationDismissed RecommendationStatus = "dismissed"
	RecommendationSnoozed   RecommendationStatus = "snoozed"
)

// RecommendationStatuses in display order.
var RecommendationStatuses = []RecommendationStatus{RecommendationOpen, RecommendationSnoozed, RecommendationDone, RecommendationDismissed}

func (s RecommendationStatus) Valid() bool {
	return slices.Contains(RecommendationStatuses, s)
}

const (
	// RecommendationIDPrefix starts every id (rec-<8 hex>).
	RecommendationIDPrefix = "rec-"
	// RecommendationRuleSourcePrefix marks a recommendation made by an engine rule.
	RecommendationRuleSourcePrefix = "rule:"

	// RecommendationsPerReview caps what one review may add (0-3).
	RecommendationsPerReview = 3
	// RecommendationOutcomeWindowDays: done items this recent are re-checked by the next review.
	RecommendationOutcomeWindowDays = 60
	// RecommendationOutcomeLimit caps the previous items given to a review.
	RecommendationOutcomeLimit = 10
	// RecommendationDismissQuietDays: a dismissed rule item keeps the rule quiet this long.
	RecommendationDismissQuietDays = 90
	// RecommendationDoneQuietDays: a rule item the user marked done keeps the
	// rule quiet this long, so the sheet can catch up with what was done.
	RecommendationDoneQuietDays = 30
	// RecommendationMaxSnoozeDays bounds a snooze.
	RecommendationMaxSnoozeDays = 365

	recommendationTitleMax    = 120
	recommendationActionMax   = 400
	recommendationEvidenceMax = 300
	recommendationNoteMax     = 300
)

// Recommendation is one row of the Recommendations tab.
type Recommendation struct {
	ID string `json:"id"`
	// Row is the sheet row, for diagnostics; the id is the stable key.
	Row     int    `json:"row"`
	Created string `json:"created"`
	// Source is the review that made it (account-review, budget-review from
	// the budget advisor, transaction-review, credit-summary, year-review,
	// financial-health, intelligence-briefing, think-tank) or an engine rule
	// (rule:<id>).
	Source string `json:"source"`
	Title  string `json:"title"`
	Action string `json:"action"`
	// ImpactVND is the estimated VND per year; nil when not estimated.
	ImpactVND *float64 `json:"impactVnd"`
	// EvidencePath is the app route of the evidence; Evidence the short facts.
	EvidencePath string `json:"evidencePath"`
	Evidence     string `json:"evidence"`
	// Status as stored; in a list, a snooze that ended reads as open.
	Status        RecommendationStatus `json:"status"`
	SnoozedUntil  string               `json:"snoozedUntil,omitempty"`
	StatusChanged string               `json:"statusChanged,omitempty"`
	Outcome       string               `json:"outcome,omitempty"`
	// Page is the app route the recommendation belongs to.
	Page string `json:"page"`
	ETag string `json:"etag"`
}

// Pending reports an item still waiting for the user: open or snoozed.
func (r Recommendation) Pending() bool {
	return r.Status == RecommendationOpen || r.Status == RecommendationSnoozed
}

// IsRule reports an item made by an engine rule.
func (r Recommendation) IsRule() bool {
	return strings.HasPrefix(r.Source, RecommendationRuleSourcePrefix)
}

// RecommendationIssue is a row the reader left out, with the reason.
type RecommendationIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// RecommendationSheet is the parsed tab; Exists is false until the first item.
type RecommendationSheet struct {
	Exists bool                  `json:"exists"`
	Items  []Recommendation      `json:"items"`
	Issues []RecommendationIssue `json:"issues"`
}

// RecommendationRow is a row as written: the reader's input and the writer's output.
type RecommendationRow struct {
	ID, Created, Source, Title, Action string
	ImpactVND                          *float64
	Evidence                           string
	Status                             RecommendationStatus
	SnoozedUntil, StatusChanged        string
	Outcome, Page                      string
}

// ParseRecommendationRow validates a row read from the sheet. Dates are
// yyyy-mm-dd (date serials are accepted); a blank status is open.
func ParseRecommendationRow(row RecommendationRow, sheetRow int, etag string) (Recommendation, []string) {
	var problems []string
	id := strings.TrimSpace(row.ID)
	if !strings.HasPrefix(id, RecommendationIDPrefix) || len(id) <= len(RecommendationIDPrefix) {
		problems = append(problems, fmt.Sprintf("id %q must look like %s<code>", id, RecommendationIDPrefix))
	}
	title := strings.TrimSpace(row.Title)
	if title == "" {
		problems = append(problems, "title is required")
	}
	status := RecommendationStatus(strings.ToLower(strings.TrimSpace(string(row.Status))))
	if status == "" {
		status = RecommendationOpen
	}
	if !status.Valid() {
		problems = append(problems, fmt.Sprintf("status %q must be open, done, dismissed or snoozed", strings.TrimSpace(string(row.Status))))
	}
	created, problem := normalizeSheetDate(row.Created, "created")
	problems = appendProblem(problems, problem)
	snoozed, problem := normalizeSheetDate(row.SnoozedUntil, "snoozed until")
	problems = appendProblem(problems, problem)
	changed, problem := normalizeSheetDate(row.StatusChanged, "status changed")
	problems = appendProblem(problems, problem)
	if row.ImpactVND != nil && (*row.ImpactVND < 0 || math.IsNaN(*row.ImpactVND)) {
		problems = append(problems, "impact must be 0 or more")
	}
	if len(problems) > 0 {
		return Recommendation{}, problems
	}
	path, evidence := splitEvidence(row.Evidence)
	item := Recommendation{
		ID: id, Row: sheetRow, Created: created, Source: strings.TrimSpace(row.Source), Title: title,
		Action: strings.TrimSpace(row.Action), ImpactVND: row.ImpactVND, EvidencePath: path, Evidence: evidence,
		Status: status, SnoozedUntil: snoozed, StatusChanged: changed, Outcome: strings.TrimSpace(row.Outcome),
		Page: strings.TrimSpace(row.Page), ETag: etag,
	}
	return item, nil
}

func appendProblem(problems []string, problem string) []string {
	if problem != "" {
		return append(problems, problem)
	}
	return problems
}

// normalizeSheetDate reads yyyy-mm-dd or a date serial; blank stays blank.
func normalizeSheetDate(value string, field string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}
	if parsed, err := time.Parse(time.DateOnly, trimmed); err == nil {
		return parsed.Format(time.DateOnly), ""
	}
	if serial, err := strconv.ParseFloat(trimmed, 64); err == nil && serial > 1000 {
		return SerialToDate(serial).Format(time.DateOnly), ""
	}
	return "", fmt.Sprintf("%s %q must be a yyyy-mm-dd date", field, trimmed)
}

// Evidence is written as "<app path>: <facts>" so the sheet reads on its own.
func joinEvidence(path string, facts string) string {
	path, facts = strings.TrimSpace(path), strings.TrimSpace(facts)
	switch {
	case path == "":
		return facts
	case facts == "":
		return path
	}
	return path + ": " + facts
}

func splitEvidence(value string) (string, string) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") {
		return "", value
	}
	path, facts, found := strings.Cut(value, ": ")
	if !found {
		if strings.ContainsAny(value, " \t") {
			return "", value
		}
		return value, ""
	}
	if strings.ContainsAny(path, " \t") {
		return "", value
	}
	return path, strings.TrimSpace(facts)
}

// SheetRow renders the item for the writer.
func (r Recommendation) SheetRow() RecommendationRow {
	return RecommendationRow{
		ID: r.ID, Created: r.Created, Source: r.Source, Title: r.Title, Action: r.Action, ImpactVND: r.ImpactVND,
		Evidence: joinEvidence(r.EvidencePath, r.Evidence), Status: r.Status, SnoozedUntil: r.SnoozedUntil,
		StatusChanged: r.StatusChanged, Outcome: r.Outcome, Page: r.Page,
	}
}

// RecommendationUpdate rewrites the status columns (H-K) of one row.
type RecommendationUpdate struct {
	ID            string
	Row           int
	Status        RecommendationStatus
	SnoozedUntil  string
	StatusChanged string
	Outcome       string
}

// RecommendationChanges is what one write adds and updates. IDs of added
// items are unique in the tab.
type RecommendationChanges struct {
	Add    []Recommendation
	Update []RecommendationUpdate
}

func (c RecommendationChanges) Empty() bool {
	return len(c.Add) == 0 && len(c.Update) == 0
}

// RecommendationPlan computes the changes from the tab as it is now. The
// store runs it under its write lock on a fresh read, so concurrent writers
// cannot add the same item twice.
type RecommendationPlan func(sheet RecommendationSheet) (RecommendationChanges, error)

// RecommendationWrite is the tab after a write and what changed.
type RecommendationWrite struct {
	Sheet   RecommendationSheet
	Added   []string
	Updated []string
}

// Items returns the items with the given ids, in that order.
func (w RecommendationWrite) Items(ids []string) []Recommendation {
	byID := map[string]Recommendation{}
	for _, item := range w.Sheet.Items {
		byID[item.ID] = item
	}
	out := []Recommendation{}
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			out = append(out, item)
		}
	}
	return out
}

// newRecommendationID is rec-<8 hex> from the source, title and date,
// salted until it is unique.
func newRecommendationID(source string, title string, created string, taken map[string]bool) string {
	for salt := 0; ; salt++ {
		sum := sha256.Sum256([]byte(source + "\x00" + RecommendationTitleKey(title) + "\x00" + created + "\x00" + strconv.Itoa(salt)))
		id := RecommendationIDPrefix + hex.EncodeToString(sum[:4])
		if !taken[id] {
			taken[id] = true
			return id
		}
	}
}

func takenIDs(sheet RecommendationSheet) map[string]bool {
	taken := map[string]bool{}
	for _, item := range sheet.Items {
		taken[item.ID] = true
	}
	return taken
}

// RecommendationTitleKey normalises a title for de-duplication: case,
// accents, punctuation and spacing are ignored.
func RecommendationTitleKey(title string) string {
	return FoldText(title)
}

// List and filters.

type RecommendationFilter struct {
	Status RecommendationStatus `json:"status,omitempty"`
	Page   string               `json:"page,omitempty"`
	Source string               `json:"source,omitempty"`
}

func (f RecommendationFilter) Validate() error {
	if f.Status != "" && !f.Status.Valid() {
		return fmt.Errorf("%w: status must be open, done, dismissed or snoozed", ErrValidation)
	}
	return nil
}

func (f RecommendationFilter) matches(item Recommendation) bool {
	return (f.Status == "" || item.Status == f.Status) &&
		(strings.TrimSpace(f.Page) == "" || item.Page == strings.TrimSpace(f.Page)) &&
		(strings.TrimSpace(f.Source) == "" || item.Source == strings.TrimSpace(f.Source))
}

// RecommendationSummary is the headline: counts per status and what the
// open items are worth a year.
type RecommendationSummary struct {
	Open      int `json:"open"`
	Snoozed   int `json:"snoozed"`
	Done      int `json:"done"`
	Dismissed int `json:"dismissed"`
	// OpenImpactVND sums the open items' yearly impact; OpenWithoutImpact
	// counts open items with no estimate.
	OpenImpactVND     float64 `json:"openImpactVnd"`
	OpenWithoutImpact int     `json:"openWithoutImpact"`
}

type RecommendationList struct {
	TabExists bool                  `json:"tabExists"`
	Items     []Recommendation      `json:"items"`
	Summary   RecommendationSummary `json:"summary"`
	Issues    []RecommendationIssue `json:"issues"`
}

// BuildRecommendationList filters and orders the items: by status (open,
// snoozed, done, dismissed), open items by impact then newest, snoozed ones
// by the end of the snooze, the others by the date their status changed,
// newest first. A snooze that ended reads as open.
func BuildRecommendationList(sheet RecommendationSheet, filter RecommendationFilter, today time.Time) RecommendationList {
	list := RecommendationList{TabExists: sheet.Exists, Items: []Recommendation{}, Issues: sheet.Issues}
	if list.Issues == nil {
		list.Issues = []RecommendationIssue{}
	}
	for _, item := range sheet.Items {
		if item.Status == RecommendationSnoozed && item.SnoozedUntil != "" && item.SnoozedUntil <= today.Format(time.DateOnly) {
			item.Status = RecommendationOpen
		}
		if !filter.matches(item) {
			continue
		}
		list.Items = append(list.Items, item)
		switch item.Status {
		case RecommendationOpen:
			list.Summary.Open++
			if item.ImpactVND != nil {
				list.Summary.OpenImpactVND += *item.ImpactVND
			} else {
				list.Summary.OpenWithoutImpact++
			}
		case RecommendationSnoozed:
			list.Summary.Snoozed++
		case RecommendationDone:
			list.Summary.Done++
		case RecommendationDismissed:
			list.Summary.Dismissed++
		}
	}
	slices.SortStableFunc(list.Items, compareRecommendations)
	return list
}

func compareRecommendations(a, b Recommendation) int {
	if order := cmp.Compare(slices.Index(RecommendationStatuses, a.Status), slices.Index(RecommendationStatuses, b.Status)); order != 0 {
		return order
	}
	if a.Status == RecommendationOpen {
		if order := cmp.Compare(impactOf(b), impactOf(a)); order != 0 {
			return order
		}
		return cmp.Compare(b.Created, a.Created)
	}
	if a.Status == RecommendationSnoozed {
		return cmp.Compare(a.SnoozedUntil, b.SnoozedUntil)
	}
	return cmp.Compare(cmp.Or(b.StatusChanged, b.Created), cmp.Or(a.StatusChanged, a.Created))
}

func impactOf(item Recommendation) float64 {
	if item.ImpactVND == nil {
		return -1
	}
	return *item.ImpactVND
}

// Status changes by the user.

type RecommendationStatusInput struct {
	// Status is done, dismissed, snoozed, or open to reopen.
	Status RecommendationStatus `json:"status"`
	// SnoozedUntil (yyyy-mm-dd, after today) or SnoozeDays sets a snooze.
	SnoozedUntil string `json:"snoozedUntil,omitempty"`
	SnoozeDays   int    `json:"snoozeDays,omitempty"`
	ETag         string `json:"etag"`
}

func (i RecommendationStatusInput) Validate() error {
	if !i.Status.Valid() {
		return fmt.Errorf("%w: status must be open, done, dismissed or snoozed", ErrValidation)
	}
	if strings.TrimSpace(i.ETag) == "" {
		return fmt.Errorf("%w: etag is required", ErrValidation)
	}
	until := strings.TrimSpace(i.SnoozedUntil)
	if i.Status != RecommendationSnoozed {
		if until != "" || i.SnoozeDays != 0 {
			return fmt.Errorf("%w: snoozedUntil and snoozeDays only go with status snoozed", ErrValidation)
		}
		return nil
	}
	switch {
	case until != "" && i.SnoozeDays != 0:
		return fmt.Errorf("%w: give snoozedUntil or snoozeDays, not both", ErrValidation)
	case until == "" && (i.SnoozeDays < 1 || i.SnoozeDays > RecommendationMaxSnoozeDays):
		return fmt.Errorf("%w: a snooze needs snoozedUntil or snoozeDays between 1 and %d", ErrValidation, RecommendationMaxSnoozeDays)
	case until != "":
		if _, err := time.Parse(time.DateOnly, until); err != nil {
			return fmt.Errorf("%w: snoozedUntil must be yyyy-mm-dd", ErrValidation)
		}
	}
	return nil
}

// PlanStatusChange applies the user's change to the item with the id when
// its row still matches the ETag. Columns H-J change; the outcome stays.
func PlanStatusChange(sheet RecommendationSheet, id string, input RecommendationStatusInput, today time.Time) (RecommendationChanges, error) {
	if err := input.Validate(); err != nil {
		return RecommendationChanges{}, err
	}
	item, ok := findRecommendation(sheet, id)
	if !ok {
		return RecommendationChanges{}, fmt.Errorf("%w: recommendation %s", ErrNotFound, strings.TrimSpace(id))
	}
	if item.ETag != strings.TrimSpace(input.ETag) {
		return RecommendationChanges{}, fmt.Errorf("%w: recommendation %s changed since it was read; reload and retry", ErrConflict, item.ID)
	}
	until := ""
	if input.Status == RecommendationSnoozed {
		until = strings.TrimSpace(input.SnoozedUntil)
		if until == "" {
			until = today.AddDate(0, 0, input.SnoozeDays).Format(time.DateOnly)
		}
		if until <= today.Format(time.DateOnly) || until > today.AddDate(0, 0, RecommendationMaxSnoozeDays).Format(time.DateOnly) {
			return RecommendationChanges{}, fmt.Errorf("%w: snoozedUntil must be after today and within %d days", ErrValidation, RecommendationMaxSnoozeDays)
		}
	}
	return RecommendationChanges{Update: []RecommendationUpdate{{
		ID: item.ID, Row: item.Row, Status: input.Status, SnoozedUntil: until,
		StatusChanged: today.Format(time.DateOnly), Outcome: item.Outcome,
	}}}, nil
}

func findRecommendation(sheet RecommendationSheet, id string) (Recommendation, bool) {
	id = strings.TrimSpace(id)
	for _, item := range sheet.Items {
		if item.ID == id {
			return item, true
		}
	}
	return Recommendation{}, false
}

// AI reviews: proposals and outcome checks.

// RecommendationProposal is a recommendation as a review returns it.
type RecommendationProposal struct {
	Title     string   `json:"title"`
	Action    string   `json:"action"`
	ImpactVND *float64 `json:"impactVnd"`
	Evidence  string   `json:"evidence"`
	invalid   bool
}

// UnmarshalJSON accepts impactVnd as a number, a numeric string or null; any
// other value marks the proposal invalid instead of failing the review.
func (p *RecommendationProposal) UnmarshalJSON(data []byte) error {
	var raw struct {
		Title    any `json:"title"`
		Action   any `json:"action"`
		Impact   any `json:"impactVnd"`
		Evidence any `json:"evidence"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = RecommendationProposal{}
	var ok bool
	if p.Title, ok = optionalText(raw.Title); !ok {
		p.invalid = true
	}
	if p.Action, ok = optionalText(raw.Action); !ok {
		p.invalid = true
	}
	if p.Evidence, ok = optionalText(raw.Evidence); !ok {
		p.invalid = true
	}
	switch value := raw.Impact.(type) {
	case nil:
	case float64:
		p.ImpactVND = &value
	case string:
		cleaned := strings.NewReplacer(",", "", ".", "", " ", "", "₫", "", "VND", "").Replace(strings.TrimSpace(value))
		if cleaned == "" {
			break
		}
		parsed, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			p.invalid = true
			break
		}
		p.ImpactVND = &parsed
	default:
		p.invalid = true
	}
	return nil
}

func optionalText(value any) (string, bool) {
	switch text := value.(type) {
	case nil:
		return "", true
	case string:
		return text, true
	}
	return "", false
}

// NormalizeProposals trims the proposals and keeps at most three valid ones:
// a title and an action, an impact of 0 or more or none, not repeated.
func NormalizeProposals(proposals []RecommendationProposal) []RecommendationProposal {
	out := []RecommendationProposal{}
	seen := map[string]bool{}
	for _, proposal := range proposals {
		proposal.Title = clip(strings.TrimSpace(proposal.Title), recommendationTitleMax)
		proposal.Action = clip(strings.TrimSpace(proposal.Action), recommendationActionMax)
		proposal.Evidence = clip(strings.TrimSpace(proposal.Evidence), recommendationEvidenceMax)
		key := RecommendationTitleKey(proposal.Title)
		if proposal.invalid || key == "" || proposal.Action == "" || seen[key] {
			continue
		}
		if proposal.ImpactVND != nil {
			if *proposal.ImpactVND < 0 || math.IsNaN(*proposal.ImpactVND) || math.IsInf(*proposal.ImpactVND, 0) {
				continue
			}
			rounded := math.Round(*proposal.ImpactVND)
			proposal.ImpactVND = &rounded
		}
		seen[key] = true
		if out = append(out, proposal); len(out) == RecommendationsPerReview {
			break
		}
	}
	return out
}

func clip(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

type OutcomeVerdict string

const (
	OutcomeMet     OutcomeVerdict = "met"
	OutcomeNotMet  OutcomeVerdict = "not_met"
	OutcomeUnclear OutcomeVerdict = "unclear"
)

var outcomeLabels = map[OutcomeVerdict]string{OutcomeMet: "Met", OutcomeNotMet: "Not met", OutcomeUnclear: "Unclear"}

// RecommendationOutcome is a review's verdict on an earlier recommendation.
type RecommendationOutcome struct {
	ID      string         `json:"id"`
	Verdict OutcomeVerdict `json:"verdict"`
	Note    string         `json:"note"`
}

// NormalizeOutcomes keeps outcomes with an id and a known verdict, one per id.
func NormalizeOutcomes(outcomes []RecommendationOutcome) []RecommendationOutcome {
	out := []RecommendationOutcome{}
	seen := map[string]bool{}
	for _, outcome := range outcomes {
		outcome.ID = strings.TrimSpace(outcome.ID)
		outcome.Verdict = OutcomeVerdict(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(string(outcome.Verdict))), " ", "_"))
		outcome.Note = clip(strings.TrimSpace(outcome.Note), recommendationNoteMax)
		if outcome.ID == "" || seen[outcome.ID] || outcomeLabels[outcome.Verdict] == "" {
			continue
		}
		seen[outcome.ID] = true
		out = append(out, outcome)
	}
	return out
}

// RecentForReview are the items of a source a review checks: pending ones
// and those done in the last 60 days, newest first, at most ten.
func RecentForReview(sheet RecommendationSheet, source string, today time.Time) []Recommendation {
	since := today.AddDate(0, 0, -RecommendationOutcomeWindowDays).Format(time.DateOnly)
	recent := []Recommendation{}
	for _, item := range sheet.Items {
		if item.Source != source {
			continue
		}
		if item.Pending() || (item.Status == RecommendationDone && cmp.Or(item.StatusChanged, item.Created) >= since) {
			recent = append(recent, item)
		}
	}
	slices.SortStableFunc(recent, func(a, b Recommendation) int { return cmp.Compare(b.Created, a.Created) })
	if len(recent) > RecommendationOutcomeLimit {
		recent = recent[:RecommendationOutcomeLimit]
	}
	return recent
}

// ReviewRecommendations is what a review hands to the list.
type ReviewRecommendations struct {
	Source    string                   `json:"source"`
	Page      string                   `json:"page"`
	Proposals []RecommendationProposal `json:"proposals"`
	Outcomes  []RecommendationOutcome  `json:"outcomes"`
}

// RecommendationRecord is what a review changed in the list.
type RecommendationRecord struct {
	// Created are the new items; proposals matching a pending item of the
	// same source by title are counted in Duplicates.
	Created    []Recommendation `json:"created"`
	Duplicates int              `json:"duplicates"`
	// Checked are the earlier items the review gave an outcome for.
	Checked []Recommendation `json:"checked"`
	// IgnoredOutcomes name ids the review was not asked about.
	IgnoredOutcomes int `json:"ignoredOutcomes"`
	// Error is set when the list could not be written; the review still stands.
	Error string `json:"error,omitempty"`
}

// PlanReviewRecord adds the proposals that are not already pending for the
// source (same title, ignoring case, accents and punctuation) and writes the
// outcomes for items in RecentForReview: the note in K, and a met item still
// pending becomes done.
func PlanReviewRecord(sheet RecommendationSheet, review ReviewRecommendations, today time.Time) (RecommendationChanges, int, int) {
	date := today.Format(time.DateOnly)
	pending := map[string]bool{}
	for _, item := range sheet.Items {
		if item.Source == review.Source && item.Pending() {
			pending[RecommendationTitleKey(item.Title)] = true
		}
	}
	changes := RecommendationChanges{}
	taken := takenIDs(sheet)
	duplicates := 0
	for _, proposal := range NormalizeProposals(review.Proposals) {
		key := RecommendationTitleKey(proposal.Title)
		if pending[key] {
			duplicates++
			continue
		}
		pending[key] = true
		changes.Add = append(changes.Add, Recommendation{
			ID: newRecommendationID(review.Source, proposal.Title, date, taken), Created: date, Source: review.Source,
			Title: proposal.Title, Action: proposal.Action, ImpactVND: proposal.ImpactVND, EvidencePath: review.Page,
			Evidence: proposal.Evidence, Status: RecommendationOpen, Page: review.Page,
		})
	}
	allowed := map[string]Recommendation{}
	for _, item := range RecentForReview(sheet, review.Source, today) {
		allowed[item.ID] = item
	}
	ignored := 0
	for _, outcome := range NormalizeOutcomes(review.Outcomes) {
		item, ok := allowed[outcome.ID]
		if !ok {
			ignored++
			continue
		}
		update := RecommendationUpdate{
			ID: item.ID, Row: item.Row, Status: item.Status, SnoozedUntil: item.SnoozedUntil,
			StatusChanged: item.StatusChanged, Outcome: outcomeText(outcome),
		}
		if outcome.Verdict == OutcomeMet && item.Pending() {
			update.Status, update.SnoozedUntil, update.StatusChanged = RecommendationDone, "", date
		}
		changes.Update = append(changes.Update, update)
	}
	return changes, duplicates, ignored
}

func outcomeText(outcome RecommendationOutcome) string {
	label := outcomeLabels[outcome.Verdict]
	if outcome.Note == "" {
		return label
	}
	return label + ": " + outcome.Note
}

// RecommendationFormatInstruction extends a review's JSON contract with the
// recommendation list (engine code, not prompt content).
const RecommendationFormatInstruction = `The same JSON object may also carry these optional keys:
  "recommendations": [{"title": "Short imperative title", "action": "The concrete step to take", "impactVnd": 1200000, "evidence": "The facts behind it"}],
  "outcomes": [{"id": "rec-…", "verdict": "<one of: met, not_met, unclear>", "note": "One sentence on what the data shows"}]

Recommendation rules:
- 0-3 recommendations, only for actions worth tracking until done; none is fine.
- impactVnd is the estimated VND gained or saved per year if the action is taken, from the given figures; null when it cannot be estimated. Never negative.
- evidence quotes the figures that justify it, briefly.
- Do not repeat a recommendation listed under PREVIOUS RECOMMENDATIONS; give it an outcome instead.
- outcomes: only for ids listed under PREVIOUS RECOMMENDATIONS, judged from the current data (met = the action was taken or the problem is gone).`

// PreviousRecommendationsBlock lists earlier items for the outcome check.
func PreviousRecommendationsBlock(items []Recommendation, mask func(string) string) string {
	if len(items) == 0 {
		return "\n\nPREVIOUS RECOMMENDATIONS: none."
	}
	type previous struct {
		ID      string   `json:"id"`
		Created string   `json:"created"`
		Status  string   `json:"status"`
		Title   string   `json:"title"`
		Action  string   `json:"action"`
		Impact  *float64 `json:"impactVnd,omitempty"`
		Outcome string   `json:"outcome,omitempty"`
	}
	rows := make([]previous, 0, len(items))
	for _, item := range items {
		rows = append(rows, previous{
			ID: item.ID, Created: item.Created, Status: string(item.Status), Title: mask(item.Title),
			Action: mask(item.Action), Impact: item.ImpactVND, Outcome: mask(item.Outcome),
		})
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return "\n\nPREVIOUS RECOMMENDATIONS: none."
	}
	return "\n\nPREVIOUS RECOMMENDATIONS (made earlier from this review; status as the user set it):\n" + string(payload)
}

// FormatVND renders an amount like the UI: 1.250.000 ₫.
func FormatVND(value float64) string {
	rounded := int64(math.Round(value))
	sign := ""
	if rounded < 0 {
		sign, rounded = "-", -rounded
	}
	digits := strconv.FormatInt(rounded, 10)
	var grouped strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(digit)
	}
	return sign + grouped.String() + " ₫"
}
