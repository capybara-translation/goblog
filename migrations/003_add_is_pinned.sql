-- Add is_pinned column to posts table
-- is_pinned: true (1) の記事は公開ページのヘッダーに表示される
ALTER TABLE posts ADD COLUMN is_pinned INTEGER NOT NULL DEFAULT 0;

-- Index for filtering pinned posts
CREATE INDEX IF NOT EXISTS idx_posts_is_pinned ON posts(is_pinned);
