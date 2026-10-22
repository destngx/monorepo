-- The masking rules version a page was masked with (HW-AI-7). Pages
-- masked with older rules are masked again when their item opens.
ALTER TABLE pages ADD COLUMN mask_version INTEGER NOT NULL DEFAULT 0;
