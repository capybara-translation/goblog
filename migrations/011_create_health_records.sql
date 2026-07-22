CREATE TABLE IF NOT EXISTS health_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    measured_at DATETIME NOT NULL,          -- Health Planet の測定時刻（分単位、ローカル時刻）
    metric TEXT NOT NULL,                   -- weight / body_fat / systolic / diastolic / pulse
    value REAL NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (measured_at, metric)
);

-- Single-row table holding the Health Planet OAuth token state.
-- Tokens are stored in plaintext: they are needed in plaintext to call the
-- API, and the DB only leaves the host via the private S3 backup.
CREATE TABLE IF NOT EXISTS healthplanet_tokens (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
