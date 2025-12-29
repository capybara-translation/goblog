-- ユーザーテーブルの作成
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- usernameにインデックスを作成（ログイン時の検索を高速化）
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
