-- A page a person removed from a document (HW-FIND-3). The row stays, so
-- the other pages of its file keep their numbers and a new reading of the
-- file keeps it removed; NULL means the page is shown.
ALTER TABLE pages ADD COLUMN removed_at TEXT;
