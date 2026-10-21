-- Phase 1: accounts, tenants, people, visits, documents, files, inbox.
-- Times are RFC 3339 UTC text; calendar dates are YYYY-MM-DD text.

CREATE TABLE users (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  display_name  TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at    TEXT NOT NULL
);

CREATE TABLE sessions (
  id_hash      TEXT PRIMARY KEY,
  user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  csrf_token   TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  expires_at   TEXT NOT NULL
);
CREATE INDEX sessions_user ON sessions (user_id);

CREATE TABLE login_failures (
  email TEXT NOT NULL,
  at    TEXT NOT NULL
);
CREATE INDEX login_failures_email ON login_failures (email, at);

CREATE TABLE tenants (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE memberships (
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  user_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role      TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
  joined_at TEXT NOT NULL,
  PRIMARY KEY (tenant_id, user_id)
);
CREATE INDEX memberships_user ON memberships (user_id);

CREATE TABLE invites (
  id         TEXT PRIMARY KEY,
  tenant_id  TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  role       TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
  created_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  used_by    TEXT REFERENCES users (id) ON DELETE SET NULL,
  used_at    TEXT
);
CREATE INDEX invites_tenant ON invites (tenant_id);

CREATE TABLE upload_tokens (
  id           TEXT PRIMARY KEY,
  tenant_id    TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  token_hash   TEXT NOT NULL UNIQUE,
  created_at   TEXT NOT NULL,
  last_used_at TEXT
);
CREATE INDEX upload_tokens_owner ON upload_tokens (tenant_id, user_id);

CREATE TABLE people (
  id           TEXT PRIMARY KEY,
  tenant_id    TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  birth_date   TEXT NOT NULL DEFAULT '',
  sex          TEXT NOT NULL DEFAULT '' CHECK (sex IN ('', 'female', 'male')),
  user_id      TEXT REFERENCES users (id) ON DELETE SET NULL,
  created_at   TEXT NOT NULL
);
CREATE INDEX people_tenant ON people (tenant_id);

CREATE TABLE providers (
  id        TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  name      TEXT NOT NULL,
  name_key  TEXT NOT NULL,
  kind      TEXT NOT NULL,
  UNIQUE (tenant_id, name_key)
);

CREATE TABLE visits (
  id          TEXT PRIMARY KEY,
  tenant_id   TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  person_id   TEXT NOT NULL REFERENCES people (id),
  provider_id TEXT REFERENCES providers (id),
  date        TEXT NOT NULL,
  type        TEXT NOT NULL,
  title       TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL
);
CREATE INDEX visits_person ON visits (tenant_id, person_id, date);

CREATE TABLE files (
  id            TEXT PRIMARY KEY,
  tenant_id     TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  sha256        TEXT NOT NULL,
  mime          TEXT NOT NULL,
  size          INTEGER NOT NULL,
  original_name TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL,
  UNIQUE (tenant_id, sha256)
);
CREATE INDEX files_sha ON files (sha256);

CREATE TABLE documents (
  id         TEXT PRIMARY KEY,
  tenant_id  TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  visit_id   TEXT NOT NULL REFERENCES visits (id),
  person_id  TEXT NOT NULL REFERENCES people (id),
  kind       TEXT NOT NULL,
  title      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX documents_visit ON documents (tenant_id, visit_id);

CREATE TABLE document_files (
  document_id TEXT NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
  position    INTEGER NOT NULL,
  file_id     TEXT NOT NULL REFERENCES files (id),
  PRIMARY KEY (document_id, position)
);
CREATE INDEX document_files_file ON document_files (file_id);

CREATE TABLE inbox_items (
  id          TEXT PRIMARY KEY,
  tenant_id   TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  status      TEXT NOT NULL CHECK (status IN ('new', 'filed', 'discarded')),
  source      TEXT NOT NULL,
  note        TEXT NOT NULL DEFAULT '',
  uploaded_by TEXT NOT NULL REFERENCES users (id),
  document_id TEXT REFERENCES documents (id) ON DELETE SET NULL,
  created_at  TEXT NOT NULL
);
CREATE INDEX inbox_items_status ON inbox_items (tenant_id, status, created_at);

CREATE TABLE inbox_files (
  item_id  TEXT NOT NULL REFERENCES inbox_items (id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  file_id  TEXT NOT NULL REFERENCES files (id),
  PRIMARY KEY (item_id, position)
);
CREATE INDEX inbox_files_file ON inbox_files (file_id);
