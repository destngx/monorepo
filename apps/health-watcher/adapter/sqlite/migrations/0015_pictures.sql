-- Pictures printed on a page, such as an X-ray film, an ultrasound frame,
-- an endoscopy photo or an ECG trace, cut out of the page image and kept
-- as their own images (HW-READ-11, FHIR Media). box_json is where the
-- picture sits on the page, 0-1 from the top-left corner.
CREATE TABLE pictures (
  id         TEXT PRIMARY KEY,
  tenant_id  TEXT NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  page_id    TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
  position   INTEGER NOT NULL,
  kind       TEXT NOT NULL CHECK (kind IN ('xray', 'ultrasound', 'endoscopy', 'ecg', 'ct', 'mri', 'photo', 'diagram', 'other')),
  caption    TEXT NOT NULL DEFAULT '',
  box_json   TEXT NOT NULL,
  width      INTEGER NOT NULL,
  height     INTEGER NOT NULL,
  image_sha  TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE (page_id, position)
);
CREATE INDEX pictures_page ON pictures (tenant_id, page_id);
CREATE INDEX pictures_image ON pictures (image_sha);
