-- A short title per page, written by the AI reading or, for a document
-- filed before titles, by a title job that sends only the masked text
-- (HW-READ-9). '' means the page has none yet.
ALTER TABLE pages ADD COLUMN title TEXT NOT NULL DEFAULT '';

-- The title job is a new job kind; SQLite can't change a CHECK, so the
-- table is rebuilt.
CREATE TABLE jobs_new (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  kind            TEXT NOT NULL CHECK (kind IN ('ocr', 'read', 'title')),
  subject_id      TEXT NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed')),
  attempts        INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT NOT NULL,
  last_error      TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  UNIQUE (kind, subject_id)
);
INSERT INTO jobs_new SELECT id, tenant_id, kind, subject_id, status, attempts, next_attempt_at, last_error, created_at, updated_at FROM jobs;
DROP TABLE jobs;
ALTER TABLE jobs_new RENAME TO jobs;
CREATE INDEX jobs_due ON jobs (status, next_attempt_at);
