CREATE TABLE IF NOT EXISTS remember_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    selector TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_remember_tokens_selector
ON remember_tokens(selector);

CREATE INDEX IF NOT EXISTS idx_remember_tokens_user_id
ON remember_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_remember_tokens_expires_at
ON remember_tokens(expires_at);
