-- Device metadata (1/2) for the "logged-in devices" admin feature, stored at
-- remember-token issue time. One ALTER per file so each is independently
-- idempotent under the runner's duplicate-column skip (see internal/db/db.go);
-- a two-statement file could leave the second column permanently unapplied.
-- SQLite requires a DEFAULT when adding a NOT NULL column to an existing table.
ALTER TABLE remember_tokens ADD COLUMN user_agent TEXT NOT NULL DEFAULT '';
