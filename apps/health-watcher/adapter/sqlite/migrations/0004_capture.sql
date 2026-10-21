-- Phone capture links (HW-IMP-9) are upload tokens that expire.
ALTER TABLE upload_tokens ADD COLUMN expires_at TEXT;
