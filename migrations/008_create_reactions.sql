-- Reaction master table: the set of emojis readers may react with.
CREATE TABLE IF NOT EXISTS reaction_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    emoji TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One row per (post, reaction type, anonymous visitor). The UNIQUE constraint
-- enforces "one emoji per visitor per post" at the DB level even under races.
CREATE TABLE IF NOT EXISTS post_reactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    reaction_type_id INTEGER NOT NULL,
    visitor_key TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (reaction_type_id) REFERENCES reaction_types(id) ON DELETE CASCADE,
    UNIQUE(post_id, reaction_type_id, visitor_key)
);

CREATE INDEX IF NOT EXISTS idx_post_reactions_post_type
    ON post_reactions(post_id, reaction_type_id);
