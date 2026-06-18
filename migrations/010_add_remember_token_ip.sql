-- Device metadata (2/2): IP captured at remember-token issue time. Separate
-- file from 009 so each ADD COLUMN is independently idempotent/recoverable.
ALTER TABLE remember_tokens ADD COLUMN ip_address TEXT NOT NULL DEFAULT '';
