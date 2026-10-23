-- Fields the FHIR R4 mapping needs (D9, docs/health-watcher/DATA-MODEL.md).

-- Encounter.period.end: the last day of an inpatient stay; '' for a visit
-- of one day.
ALTER TABLE visits ADD COLUMN end_date TEXT NOT NULL DEFAULT '';

-- DocumentReference.date and DiagnosticReport.issued: the date printed on
-- the paper; '' means the visit date.
ALTER TABLE documents ADD COLUMN date TEXT NOT NULL DEFAULT '';

-- meta.versionId and Provenance. version starts at 1 and each edit of an
-- approved record raises it. source says how the record was made:
-- 'reading' (AI reading of a document), 'manual' (typed by a person) or
-- 'import' (another system); source_detail is the reading's model or the
-- import id. approved_by and approved_at are empty for records approved
-- before this migration.
ALTER TABLE observations ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE observations ADD COLUMN source TEXT NOT NULL DEFAULT 'reading' CHECK (source IN ('reading', 'manual', 'import'));
ALTER TABLE observations ADD COLUMN source_detail TEXT NOT NULL DEFAULT '';
ALTER TABLE observations ADD COLUMN approved_by TEXT REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE observations ADD COLUMN approved_at TEXT NOT NULL DEFAULT '';

ALTER TABLE findings ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE findings ADD COLUMN source TEXT NOT NULL DEFAULT 'reading' CHECK (source IN ('reading', 'manual', 'import'));
ALTER TABLE findings ADD COLUMN source_detail TEXT NOT NULL DEFAULT '';
ALTER TABLE findings ADD COLUMN approved_by TEXT REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE findings ADD COLUMN approved_at TEXT NOT NULL DEFAULT '';

-- Records typed by hand were never part of an inbox item and were saved at
-- confidence 100. Records from a reading keep the reading's model.
UPDATE observations SET source = 'manual' WHERE item_id IS NULL AND confidence = 100;
UPDATE findings SET source = 'manual' WHERE item_id IS NULL AND confidence = 100;
UPDATE observations SET source_detail = COALESCE((SELECT model FROM readings r WHERE r.item_id = observations.item_id), '')
  WHERE source = 'reading';
UPDATE findings SET source_detail = COALESCE((SELECT model FROM readings r WHERE r.item_id = findings.item_id), '')
  WHERE source = 'reading';

-- The version an audit entry's old value had. Earlier entries are numbered
-- in order per record, and the record's current version follows the last.
ALTER TABLE audit_log ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
UPDATE audit_log SET version = (
  SELECT COUNT(*) FROM audit_log a
  WHERE a.tenant_id = audit_log.tenant_id AND a.entity = audit_log.entity AND a.entity_id = audit_log.entity_id
    AND (a.at < audit_log.at OR (a.at = audit_log.at AND a.id <= audit_log.id))
);
UPDATE observations SET version = 1 + (
  SELECT COUNT(*) FROM audit_log a WHERE a.tenant_id = observations.tenant_id AND a.entity = 'observation' AND a.entity_id = observations.id
);
UPDATE findings SET version = 1 + (
  SELECT COUNT(*) FROM audit_log a WHERE a.tenant_id = findings.tenant_id AND a.entity = 'finding' AND a.entity_id = findings.id
);
