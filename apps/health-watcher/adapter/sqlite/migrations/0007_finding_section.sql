-- The heading a finding was written under: the specialty or body system of
-- an exam, the study of an imaging report (HW-REC-4).
ALTER TABLE findings ADD COLUMN section TEXT NOT NULL DEFAULT '';
