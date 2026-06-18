-- Device metadata for the "logged-in devices" admin feature.
-- Stored at remember-token issue time. Existing rows (issued before this
-- migration) default to '' and are shown as "不明な端末" in the UI.
-- SQLite requires a DEFAULT when adding a NOT NULL column to an existing table.
ALTER TABLE remember_tokens ADD COLUMN user_agent TEXT NOT NULL DEFAULT '';
ALTER TABLE remember_tokens ADD COLUMN ip_address TEXT NOT NULL DEFAULT '';
