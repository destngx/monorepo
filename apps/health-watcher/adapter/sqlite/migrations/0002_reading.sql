-- Phases 2 and 3: pages, background jobs, AI readings, observations,
-- findings, their audit log, analyte links and the search index.

-- Backlog mode: the person picked once for a stack of uploads (HW-IMP-6).
ALTER TABLE inbox_items ADD COLUMN person_id TEXT REFERENCES people (id) ON DELETE SET NULL;

-- One row per page of a stored file (HW-READ-1, HW-READ-2) with its local
-- mask (HW-AI-7). Images are in the file store by hash, never here.
CREATE TABLE pages (
  id                TEXT PRIMARY KEY,
  tenant_id         TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  file_id           TEXT NOT NULL REFERENCES files (id) ON DELETE CASCADE,
  number            INTEGER NOT NULL,
  method            TEXT NOT NULL CHECK (method IN ('text', 'ocr')),
  text              TEXT NOT NULL DEFAULT '',
  confidence        REAL NOT NULL DEFAULT 0,
  width             INTEGER NOT NULL DEFAULT 0,
  height            INTEGER NOT NULL DEFAULT 0,
  image_sha         TEXT NOT NULL DEFAULT '',
  lines_json        TEXT NOT NULL DEFAULT '[]',
  -- NULL when the OCR tool could not look for barcodes.
  barcodes_json     TEXT,
  mask_status       TEXT NOT NULL DEFAULT '' CHECK (mask_status IN ('', 'ready', 'needs_check', 'checked')),
  mask_reasons_json TEXT NOT NULL DEFAULT '[]',
  mask_boxes_json   TEXT NOT NULL DEFAULT '[]',
  mask_manual_json  TEXT NOT NULL DEFAULT '[]',
  mask_lines        INTEGER NOT NULL DEFAULT 0,
  mask_text         TEXT NOT NULL DEFAULT '',
  mask_image_sha    TEXT NOT NULL DEFAULT '',
  created_at        TEXT NOT NULL,
  UNIQUE (file_id, number)
);
CREATE INDEX pages_tenant ON pages (tenant_id);
CREATE INDEX pages_image ON pages (image_sha);
CREATE INDEX pages_mask_image ON pages (mask_image_sha);

-- Background work that survives restarts (§11 Workers, HW-READ-6).
CREATE TABLE jobs (
  id              TEXT PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  kind            TEXT NOT NULL CHECK (kind IN ('ocr', 'read')),
  subject_id      TEXT NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed')),
  attempts        INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT NOT NULL,
  last_error      TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  UNIQUE (kind, subject_id)
);
CREATE INDEX jobs_due ON jobs (status, next_attempt_at);

-- The AI reading of an inbox item (HW-READ-3).
CREATE TABLE readings (
  item_id       TEXT PRIMARY KEY REFERENCES inbox_items (id) ON DELETE CASCADE,
  tenant_id     TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  status        TEXT NOT NULL CHECK (status IN ('waiting', 'off', 'needs_check', 'queued', 'done', 'failed', 'skipped')),
  reason        TEXT NOT NULL DEFAULT '',
  model         TEXT NOT NULL DEFAULT '',
  proposal_json TEXT NOT NULL DEFAULT '',
  pages_sent    INTEGER NOT NULL DEFAULT 0,
  error         TEXT NOT NULL DEFAULT '',
  updated_at    TEXT NOT NULL
);

-- Measured values (HW-REC-3). Drafts belong to an inbox item; filing
-- approves them under a document (HW-READ-4).
CREATE TABLE observations (
  id          TEXT PRIMARY KEY,
  tenant_id   TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  status      TEXT NOT NULL CHECK (status IN ('draft', 'approved', 'rejected')),
  item_id     TEXT REFERENCES inbox_items (id) ON DELETE SET NULL,
  document_id TEXT REFERENCES documents (id) ON DELETE CASCADE,
  person_id   TEXT REFERENCES people (id),
  page_id     TEXT REFERENCES pages (id) ON DELETE SET NULL,
  position    INTEGER NOT NULL DEFAULT 0,
  analyte     TEXT NOT NULL,
  metric_code TEXT NOT NULL DEFAULT '',
  value       TEXT NOT NULL,
  value_num   REAL,
  unit        TEXT NOT NULL DEFAULT '',
  ref_text    TEXT NOT NULL DEFAULT '',
  ref_low     REAL,
  ref_high    REAL,
  flag        TEXT NOT NULL DEFAULT '' CHECK (flag IN ('', 'H', 'L', '*')),
  sample_date TEXT NOT NULL DEFAULT '',
  confidence  INTEGER NOT NULL DEFAULT 100,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
CREATE INDEX observations_item ON observations (tenant_id, item_id);
CREATE INDEX observations_document ON observations (tenant_id, document_id);
CREATE INDEX observations_person ON observations (tenant_id, person_id, metric_code, sample_date);

-- Diagnoses, prescriptions, procedures and other findings (HW-REC-4).
CREATE TABLE findings (
  id          TEXT PRIMARY KEY,
  tenant_id   TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  status      TEXT NOT NULL CHECK (status IN ('draft', 'approved', 'rejected')),
  item_id     TEXT REFERENCES inbox_items (id) ON DELETE SET NULL,
  document_id TEXT REFERENCES documents (id) ON DELETE CASCADE,
  person_id   TEXT REFERENCES people (id),
  page_id     TEXT REFERENCES pages (id) ON DELETE SET NULL,
  position    INTEGER NOT NULL DEFAULT 0,
  kind        TEXT NOT NULL,
  text        TEXT NOT NULL DEFAULT '',
  icd10       TEXT NOT NULL DEFAULT '',
  drug        TEXT NOT NULL DEFAULT '',
  dose        TEXT NOT NULL DEFAULT '',
  frequency   TEXT NOT NULL DEFAULT '',
  duration    TEXT NOT NULL DEFAULT '',
  confidence  INTEGER NOT NULL DEFAULT 100,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
CREATE INDEX findings_item ON findings (tenant_id, item_id);
CREATE INDEX findings_document ON findings (tenant_id, document_id);

-- Who changed or deleted an approved observation or finding, and its old
-- value (HW-REC-6).
CREATE TABLE audit_log (
  id        TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  entity    TEXT NOT NULL CHECK (entity IN ('observation', 'finding')),
  entity_id TEXT NOT NULL,
  action    TEXT NOT NULL CHECK (action IN ('update', 'delete')),
  user_id   TEXT REFERENCES users (id) ON DELETE SET NULL,
  at        TEXT NOT NULL,
  old_json  TEXT NOT NULL
);
CREATE INDEX audit_log_entity ON audit_log (tenant_id, entity, entity_id, at);

-- Analyte names linked to a catalog entry by hand, remembered per provider
-- (HW-READ-5). provider_id is '' for papers without a provider.
CREATE TABLE analyte_links (
  tenant_id   TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  provider_id TEXT NOT NULL DEFAULT '',
  analyte_key TEXT NOT NULL,
  metric_code TEXT NOT NULL,
  PRIMARY KEY (tenant_id, provider_id, analyte_key)
);

-- Full-text search (HW-FIND-2). folded is the accent-free text that is
-- matched; body keeps the original for snippets. The engine fills the
-- index; meta.search_version tells it when to rebuild.
CREATE VIRTUAL TABLE search_index USING fts5 (
  tenant_id UNINDEXED,
  kind UNINDEXED,
  ref_id UNINDEXED,
  body UNINDEXED,
  folded,
  tokenize = 'unicode61 remove_diacritics 0'
);

CREATE TABLE meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
