-- Photos the phone sent ahead, before the person pressed Send (HW-IMP-11).
-- Send puts them in an inbox item and removes the row; the engine deletes
-- the ones never sent after a day.
CREATE TABLE staged_files (
  file_id    TEXT PRIMARY KEY REFERENCES files (id) ON DELETE CASCADE,
  tenant_id  TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at TEXT NOT NULL
);
CREATE INDEX staged_files_age ON staged_files (created_at);
