-- Metrics a family approved from the app's suggestions (HW-MET-8). They
-- join the shipped catalog for that tenant. The about_* columns hold the
-- word-for-word description fetched by LOINC code (HW-MET-6), '' when the
-- source has none.
CREATE TABLE custom_metrics (
  tenant_id     TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  code          TEXT NOT NULL,
  loinc         TEXT NOT NULL DEFAULT '',
  name          TEXT NOT NULL,
  name_vi       TEXT NOT NULL DEFAULT '',
  category      TEXT NOT NULL,
  unit          TEXT NOT NULL DEFAULT '',
  aliases_json  TEXT NOT NULL DEFAULT '[]',
  about_quote   TEXT NOT NULL DEFAULT '',
  about_title   TEXT NOT NULL DEFAULT '',
  about_url     TEXT NOT NULL DEFAULT '',
  about_source  TEXT NOT NULL DEFAULT '',
  about_fetched TEXT NOT NULL DEFAULT '',
  created_by    TEXT REFERENCES users (id) ON DELETE SET NULL,
  created_at    TEXT NOT NULL,
  PRIMARY KEY (tenant_id, code)
);

-- What the AI suggests for a printed name no metric knows (HW-MET-8): the
-- same test as a known metric ('link'), a new metric ('new'), or not a
-- test ('none', stored as rejected so it is not asked again). One row per
-- folded name, so a name is suggested once.
CREATE TABLE metric_suggestions (
  id            TEXT PRIMARY KEY,
  tenant_id     TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  analyte_key   TEXT NOT NULL,
  analyte       TEXT NOT NULL,
  unit          TEXT NOT NULL DEFAULT '',
  kind          TEXT NOT NULL CHECK (kind IN ('link', 'new', 'none')),
  metric_code   TEXT NOT NULL DEFAULT '',
  proposal_json TEXT NOT NULL DEFAULT '',
  reason        TEXT NOT NULL DEFAULT '',
  model         TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
  created_at    TEXT NOT NULL,
  decided_by    TEXT REFERENCES users (id) ON DELETE SET NULL,
  decided_at    TEXT NOT NULL DEFAULT '',
  UNIQUE (tenant_id, analyte_key)
);
CREATE INDEX metric_suggestions_status ON metric_suggestions (tenant_id, status);

-- The suggestion job is a new job kind; SQLite can't change a CHECK, so the
-- table is rebuilt.
CREATE TABLE jobs_new (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  kind            TEXT NOT NULL CHECK (kind IN ('ocr', 'read', 'title', 'suggest')),
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
