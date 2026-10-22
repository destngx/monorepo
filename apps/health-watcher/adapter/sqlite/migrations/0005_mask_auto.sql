-- The boxes the masking rules found, numbered on the review page, and
-- which of them a person turned off (HW-AI-7). NULL marks a mask made
-- before boxes were numbered; the engine masks that page again.
ALTER TABLE pages ADD COLUMN mask_auto_json TEXT;
