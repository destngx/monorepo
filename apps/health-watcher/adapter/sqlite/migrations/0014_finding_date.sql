-- The day written beside a finding, such as the day of one vaccine dose
-- (FHIR Immunization.occurrenceDateTime) or the day a next dose is due;
-- '' means the visit date. A vaccination also keeps its vaccine in drug and
-- the amount given in dose (HW-REC-4).
ALTER TABLE findings ADD COLUMN date TEXT NOT NULL DEFAULT '';
