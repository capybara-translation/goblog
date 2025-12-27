-- 記事テーブル
CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    tags TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME
);

-- slugでの検索を高速化
CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug);

-- ステータスでの検索を高速化（公開記事の一覧取得など）
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);

-- 公開日時でのソートを高速化
CREATE INDEX IF NOT EXISTS idx_posts_published_at ON posts(published_at DESC);

-- ユーザーテーブル（管理画面のログイン用）
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- セッションテーブル（Cookie認証用）
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- セッションの有効期限でのクエリを高速化
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
