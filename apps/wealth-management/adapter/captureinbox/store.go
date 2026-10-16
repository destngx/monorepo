// Package captureinbox stores quick captures, their materials and the
// drafts read from them (§6.3) in a SQLite file private to the user.
// Materials are kept only while a pending draft or an unfinished capture
// needs them (WM-CAP-17).
package captureinbox

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/sqlitefile"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const schema = `
CREATE TABLE IF NOT EXISTS captures (
	id              TEXT PRIMARY KEY,
	hash            TEXT NOT NULL UNIQUE,
	created_at      INTEGER NOT NULL,
	status          TEXT NOT NULL,
	note            TEXT NOT NULL DEFAULT '',
	has_image       INTEGER NOT NULL DEFAULT 0,
	attempts        INTEGER NOT NULL DEFAULT 0,
	next_attempt_at INTEGER NOT NULL DEFAULT 0,
	message         TEXT NOT NULL DEFAULT '',
	new_drafts      INTEGER NOT NULL DEFAULT 0,
	balances        TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS captures_queue ON captures (status, next_attempt_at, created_at);
CREATE TABLE IF NOT EXISTS capture_images (
	capture_id TEXT PRIMARY KEY,
	media_type TEXT NOT NULL,
	data       BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS capture_drafts (
	id             TEXT PRIMARY KEY,
	status         TEXT NOT NULL,
	created_at     INTEGER NOT NULL,
	decided_at     INTEGER NOT NULL DEFAULT 0,
	suggestion     TEXT NOT NULL,
	transaction_id TEXT NOT NULL DEFAULT '',
	batch_id       TEXT NOT NULL DEFAULT '',
	approved       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS capture_drafts_status ON capture_drafts (status, created_at);
CREATE TABLE IF NOT EXISTS capture_draft_sources (
	draft_id   TEXT NOT NULL,
	capture_id TEXT NOT NULL,
	position   INTEGER NOT NULL,
	PRIMARY KEY (draft_id, capture_id)
);
CREATE INDEX IF NOT EXISTS capture_draft_sources_capture ON capture_draft_sources (capture_id);`

// unneededCaptures are finished captures no pending draft comes from:
// their image and note are no longer needed (WM-CAP-17).
const unneededCaptures = `
SELECT c.id FROM captures c
WHERE c.status = 'done' AND NOT EXISTS (
	SELECT 1 FROM capture_draft_sources s JOIN capture_drafts d ON d.id = s.draft_id
	WHERE s.capture_id = c.id AND d.status = 'pending'
)`

type Store struct {
	db   *sql.DB
	path string
}

// DefaultPath is capture.db next to the settings store.
func DefaultPath(settingsPath string) string {
	return filepath.Join(filepath.Dir(settingsPath), "capture.db")
}

// ResolvePath is override (WM_CAPTURE_PATH) when set, DefaultPath otherwise.
func ResolvePath(override string, settingsPath string) string {
	if override = strings.TrimSpace(override); override != "" {
		return override
	}
	return DefaultPath(settingsPath)
}

func Open(path string) (*Store, error) {
	db, err := sqlitefile.Open(path)
	if err != nil {
		return nil, fmt.Errorf("capture store: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, errors.Join(fmt.Errorf("create capture schema: %w", err), db.Close())
	}
	return &Store{db: db, path: path}, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) AddCapture(ctx context.Context, capture domain.QuickCapture, hash string, image []byte) (domain.QuickCapture, bool, error) {
	var stored domain.QuickCapture
	var existed bool
	err := s.inTx(ctx, "add capture", func(tx *sql.Tx) error {
		var id string
		switch err := tx.QueryRowContext(ctx, `SELECT id FROM captures WHERE hash = ?`, hash).Scan(&id); {
		case err == nil:
			existing, err := getCapture(ctx, tx, id)
			stored, existed = existing, true
			return err
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		balances, err := json.Marshal(nonNilBalances(capture.Balances))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO captures (id, hash, created_at, status, note, has_image, attempts, next_attempt_at, message, new_drafts, balances)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			capture.ID, hash, millis(capture.CreatedAt), capture.Status, capture.Note, len(image) > 0, capture.Attempts,
			millisOrZero(capture.NextAttemptAt), capture.Message, capture.NewDrafts, string(balances)); err != nil {
			return err
		}
		if len(image) > 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO capture_images (capture_id, media_type, data) VALUES (?, ?, ?)`, capture.ID, http.DetectContentType(image), image); err != nil {
				return err
			}
		}
		capture.HasImage = len(image) > 0
		stored = capture
		return nil
	})
	return stored, existed, err
}

func (s *Store) ClaimCapture(ctx context.Context, now time.Time) (domain.QuickCapture, []byte, bool, error) {
	var capture domain.QuickCapture
	var image []byte
	found := false
	err := s.inTx(ctx, "claim capture", func(tx *sql.Tx) error {
		var id string
		err := tx.QueryRowContext(ctx, `SELECT id FROM captures WHERE status = ? AND next_attempt_at <= ? ORDER BY created_at, id LIMIT 1`,
			domain.QuickCaptureQueued, millis(now)).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE captures SET status = ? WHERE id = ?`, domain.QuickCaptureProcessing, id); err != nil {
			return err
		}
		if capture, err = getCapture(ctx, tx, id); err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, `SELECT data FROM capture_images WHERE capture_id = ?`, id).Scan(&image)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		found = true
		return nil
	})
	return capture, image, found, err
}

func (s *Store) RequeueProcessing(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE captures SET status = ? WHERE status = ?`, domain.QuickCaptureQueued, domain.QuickCaptureProcessing); err != nil {
		return fmt.Errorf("requeue captures: %w", err)
	}
	return nil
}

func (s *Store) CompleteCapture(ctx context.Context, capture domain.QuickCapture, created []domain.CaptureDraft, merged []domain.CaptureDraft) error {
	return s.inTx(ctx, "complete capture", func(tx *sql.Tx) error {
		if err := updateCapture(ctx, tx, capture); err != nil {
			return err
		}
		for _, draft := range created {
			suggestion, err := json.Marshal(draft.Suggestion)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO capture_drafts (id, status, created_at, suggestion) VALUES (?, ?, ?, ?)`,
				draft.ID, draft.Status, millis(draft.CreatedAt), string(suggestion)); err != nil {
				return err
			}
			if err := linkSources(ctx, tx, draft); err != nil {
				return err
			}
		}
		for _, draft := range merged {
			suggestion, err := json.Marshal(draft.Suggestion)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE capture_drafts SET suggestion = ? WHERE id = ?`, string(suggestion), draft.ID); err != nil {
				return err
			}
			if err := linkSources(ctx, tx, draft); err != nil {
				return err
			}
		}
		return dropUnneededMaterials(ctx, tx)
	})
}

func linkSources(ctx context.Context, tx *sql.Tx, draft domain.CaptureDraft) error {
	for position, captureID := range draft.CaptureIDs {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_draft_sources (draft_id, capture_id, position) VALUES (?, ?, ?)`,
			draft.ID, captureID, position); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateCapture(ctx context.Context, capture domain.QuickCapture) error {
	return s.inTx(ctx, "update capture", func(tx *sql.Tx) error {
		return updateCapture(ctx, tx, capture)
	})
}

func updateCapture(ctx context.Context, tx *sql.Tx, capture domain.QuickCapture) error {
	balances, err := json.Marshal(nonNilBalances(capture.Balances))
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE captures SET status = ?, attempts = ?, next_attempt_at = ?, message = ?, new_drafts = ?, balances = ? WHERE id = ?`,
		capture.Status, capture.Attempts, millisOrZero(capture.NextAttemptAt), capture.Message, capture.NewDrafts, string(balances), capture.ID)
	if err != nil {
		return err
	}
	return requireRow(result, "capture "+capture.ID)
}

func (s *Store) GetCapture(ctx context.Context, id string) (domain.QuickCapture, error) {
	var capture domain.QuickCapture
	err := s.inTx(ctx, "read capture", func(tx *sql.Tx) error {
		var err error
		capture, err = getCapture(ctx, tx, id)
		return err
	})
	return capture, err
}

const captureColumns = `id, created_at, status, note, has_image, attempts, next_attempt_at, message, new_drafts, balances`

func getCapture(ctx context.Context, tx *sql.Tx, id string) (domain.QuickCapture, error) {
	capture, err := scanCapture(tx.QueryRowContext(ctx, `SELECT `+captureColumns+` FROM captures WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return capture, fmt.Errorf("%w: capture %s", domain.ErrNotFound, id)
	}
	return capture, err
}

type scanner interface{ Scan(dest ...any) error }

func scanCapture(row scanner) (domain.QuickCapture, error) {
	var capture domain.QuickCapture
	var created, next int64
	var balances string
	if err := row.Scan(&capture.ID, &created, &capture.Status, &capture.Note, &capture.HasImage, &capture.Attempts, &next, &capture.Message, &capture.NewDrafts, &balances); err != nil {
		return capture, err
	}
	capture.CreatedAt = fromMillis(created)
	if next > 0 {
		at := fromMillis(next)
		capture.NextAttemptAt = &at
	}
	if err := json.Unmarshal([]byte(balances), &capture.Balances); err != nil {
		return capture, fmt.Errorf("read capture %s balances: %w", capture.ID, err)
	}
	return capture, nil
}

func (s *Store) ListCaptures(ctx context.Context, limit int) ([]domain.QuickCapture, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+captureColumns+` FROM captures ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list captures: %w", err)
	}
	defer rows.Close()
	captures := []domain.QuickCapture{}
	for rows.Next() {
		capture, err := scanCapture(rows)
		if err != nil {
			return nil, fmt.Errorf("list captures: %w", err)
		}
		captures = append(captures, capture)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list captures: %w", err)
	}
	return captures, nil
}

func (s *Store) DeleteCapture(ctx context.Context, id string) error {
	return s.inTx(ctx, "delete capture", func(tx *sql.Tx) error {
		if _, err := getCapture(ctx, tx, id); err != nil {
			return err
		}
		var drafts int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM capture_draft_sources WHERE capture_id = ?`, id).Scan(&drafts); err != nil {
			return err
		}
		if drafts > 0 {
			return fmt.Errorf("%w: capture %s has drafts; approve or reject them instead", domain.ErrConflict, id)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM capture_images WHERE capture_id = ?`, id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM captures WHERE id = ?`, id)
		return err
	})
}

const draftColumns = `id, status, created_at, decided_at, suggestion, transaction_id, batch_id`

func (s *Store) ListDrafts(ctx context.Context, status domain.CaptureDraftStatus, since time.Time) ([]domain.CaptureDraft, error) {
	query := `SELECT ` + draftColumns + ` FROM capture_drafts WHERE created_at >= ?`
	args := []any{millis(since)}
	if since.IsZero() {
		args[0] = int64(0)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	var drafts []domain.CaptureDraft
	err := s.inTx(ctx, "list drafts", func(tx *sql.Tx) error {
		var err error
		drafts, err = queryDrafts(ctx, tx, query+` ORDER BY created_at, id`, args...)
		return err
	})
	return drafts, err
}

func (s *Store) GetDrafts(ctx context.Context, ids []string) ([]domain.CaptureDraft, error) {
	if len(ids) == 0 {
		return []domain.CaptureDraft{}, nil
	}
	var drafts []domain.CaptureDraft
	err := s.inTx(ctx, "read drafts", func(tx *sql.Tx) error {
		var err error
		drafts, err = queryDrafts(ctx, tx, `SELECT `+draftColumns+` FROM capture_drafts WHERE id IN (`+placeholders(len(ids))+`) ORDER BY created_at, id`, anySlice(ids)...)
		return err
	})
	return drafts, err
}

func queryDrafts(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]domain.CaptureDraft, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	drafts := []domain.CaptureDraft{}
	for rows.Next() {
		var draft domain.CaptureDraft
		var created, decided int64
		var suggestion string
		if err := rows.Scan(&draft.ID, &draft.Status, &created, &decided, &suggestion, &draft.TransactionID, &draft.BatchID); err != nil {
			_ = rows.Close()
			return nil, err
		}
		draft.CreatedAt = fromMillis(created)
		if decided > 0 {
			at := fromMillis(decided)
			draft.DecidedAt = &at
		}
		if err := json.Unmarshal([]byte(suggestion), &draft.Suggestion); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read draft %s: %w", draft.ID, err)
		}
		drafts = append(drafts, draft)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, err
	}
	// Sources are read after the cursor closes: the pool has one connection.
	for index := range drafts {
		sources, err := tx.QueryContext(ctx, `SELECT capture_id FROM capture_draft_sources WHERE draft_id = ? ORDER BY position, capture_id`, drafts[index].ID)
		if err != nil {
			return nil, err
		}
		drafts[index].CaptureIDs = []string{}
		for sources.Next() {
			var id string
			if err := sources.Scan(&id); err != nil {
				_ = sources.Close()
				return nil, err
			}
			drafts[index].CaptureIDs = append(drafts[index].CaptureIDs, id)
		}
		if err := errors.Join(sources.Err(), sources.Close()); err != nil {
			return nil, err
		}
	}
	return drafts, nil
}

func (s *Store) DecideDrafts(ctx context.Context, decisions []domain.CaptureDraftDecision, now time.Time) error {
	return s.inTx(ctx, "decide drafts", func(tx *sql.Tx) error {
		for _, decision := range decisions {
			approved := ""
			if decision.Transaction != nil {
				encoded, err := json.Marshal(decision.Transaction)
				if err != nil {
					return err
				}
				approved = string(encoded)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE capture_drafts SET status = ?, decided_at = ?, transaction_id = ?, batch_id = ?, approved = ? WHERE id = ? AND status = ?`,
				decision.Status, millis(now), decision.TransactionID, decision.BatchID, approved, decision.ID, domain.CaptureDraftPending); err != nil {
				return err
			}
		}
		return dropUnneededMaterials(ctx, tx)
	})
}

// dropUnneededMaterials deletes the image and note of finished captures
// that no pending draft comes from.
func dropUnneededMaterials(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM capture_images WHERE capture_id IN (`+unneededCaptures+`)`); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE captures SET note = '' WHERE note != '' AND id IN (`+unneededCaptures+`)`)
	return err
}

func (s *Store) Material(ctx context.Context, materialID string) (domain.CaptureMaterial, error) {
	notFound := fmt.Errorf("%w: material %s", domain.ErrNotFound, materialID)
	if captureID, ok := strings.CutSuffix(materialID, "-image"); ok {
		material := domain.CaptureMaterial{ID: materialID, Kind: domain.CaptureImage, Name: "screenshot"}
		err := s.db.QueryRowContext(ctx, `SELECT media_type, data FROM capture_images WHERE capture_id = ?`, captureID).Scan(&material.MediaType, &material.Data)
		if errors.Is(err, sql.ErrNoRows) {
			return material, notFound
		}
		if err != nil {
			return material, fmt.Errorf("read material: %w", err)
		}
		return material, nil
	}
	if captureID, ok := strings.CutSuffix(materialID, "-note"); ok {
		material := domain.CaptureMaterial{ID: materialID, Kind: domain.CaptureText, Name: "note"}
		err := s.db.QueryRowContext(ctx, `SELECT note FROM captures WHERE id = ?`, captureID).Scan(&material.Text)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && material.Text == "") {
			return material, notFound
		}
		if err != nil {
			return material, fmt.Errorf("read material: %w", err)
		}
		return material, nil
	}
	return domain.CaptureMaterial{}, notFound
}

func (s *Store) Materials(ctx context.Context, captureIDs []string) ([]domain.CaptureMaterialEntry, error) {
	entries := []domain.CaptureMaterialEntry{}
	if len(captureIDs) == 0 {
		return entries, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, c.created_at, c.note, COALESCE(i.media_type, ''), i.capture_id IS NOT NULL
		FROM captures c LEFT JOIN capture_images i ON i.capture_id = c.id
		WHERE c.id IN (`+placeholders(len(captureIDs))+`) ORDER BY c.created_at, c.id`, anySlice(captureIDs)...)
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, note, mediaType string
		var created int64
		var hasImage bool
		if err := rows.Scan(&id, &created, &note, &mediaType, &hasImage); err != nil {
			return nil, fmt.Errorf("list materials: %w", err)
		}
		at := fromMillis(created)
		if hasImage {
			entries = append(entries, domain.CaptureMaterialEntry{ID: domain.QuickCaptureImageID(id), CaptureID: id, Kind: domain.CaptureImage, Name: "screenshot", MediaType: mediaType, CreatedAt: at})
		}
		if note != "" {
			entries = append(entries, domain.CaptureMaterialEntry{ID: domain.QuickCaptureNoteID(id), CaptureID: id, Kind: domain.CaptureText, Name: "note", Text: note, CreatedAt: at})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	return entries, nil
}

func (s *Store) Purge(ctx context.Context, before time.Time) error {
	return s.inTx(ctx, "purge captures", func(tx *sql.Tx) error {
		cutoff := millis(before)
		if _, err := tx.ExecContext(ctx, `DELETE FROM capture_draft_sources WHERE draft_id IN (SELECT id FROM capture_drafts WHERE status != ? AND decided_at < ?)`, domain.CaptureDraftPending, cutoff); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM capture_drafts WHERE status != ? AND decided_at < ?`, domain.CaptureDraftPending, cutoff); err != nil {
			return err
		}
		finished := `SELECT id FROM captures WHERE status = 'done' AND created_at < ? AND NOT EXISTS (SELECT 1 FROM capture_draft_sources s WHERE s.capture_id = captures.id)`
		if _, err := tx.ExecContext(ctx, `DELETE FROM capture_images WHERE capture_id IN (`+finished+`)`, cutoff); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM captures WHERE id IN (`+finished+`)`, cutoff)
		return err
	})
}

func (s *Store) inTx(ctx context.Context, action string, run func(tx *sql.Tx) error) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	if err := run(tx); err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) {
			return errors.Join(err, tx.Rollback())
		}
		return errors.Join(fmt.Errorf("%s: %w", action, err), tx.Rollback())
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	return nil
}

func requireRow(result sql.Result, what string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, what)
	}
	return nil
}

func nonNilBalances(balances []domain.CaptureBalance) []domain.CaptureBalance {
	if balances == nil {
		return []domain.CaptureBalance{}
	}
	return balances
}

func millis(at time.Time) int64 { return at.UnixMilli() }

func millisOrZero(at *time.Time) int64 {
	if at == nil {
		return 0
	}
	return at.UnixMilli()
}

func fromMillis(value int64) time.Time { return time.UnixMilli(value).UTC() }

func placeholders(count int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", count), ", ")
}

func anySlice(values []string) []any {
	out := make([]any, len(values))
	for index, value := range values {
		out[index] = value
	}
	return out
}
