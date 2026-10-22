-- A phone capture link made on an inbox item adds files only to that item
-- (HW-IMP-10).
ALTER TABLE upload_tokens ADD COLUMN item_id TEXT REFERENCES inbox_items (id) ON DELETE CASCADE;
