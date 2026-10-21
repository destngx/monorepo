-- Engine settings move from environment variables into the database, and
-- engine admins edit them (HW-OPS-9).
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- The account that ran first-time setup is the first one; it becomes admin.
UPDATE users SET is_admin = 1 WHERE id = (SELECT id FROM users ORDER BY created_at, id LIMIT 1);

CREATE TABLE settings (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  updated_by TEXT REFERENCES users (id) ON DELETE SET NULL
);
