-- OGP cache table for link card feature
CREATE TABLE IF NOT EXISTS ogp_cache (
    url TEXT PRIMARY KEY,
    title TEXT,
    description TEXT,
    image_url TEXT,
    site_name TEXT,
    fetched_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    error_msg TEXT
);

CREATE INDEX IF NOT EXISTS idx_ogp_cache_expires_at ON ogp_cache(expires_at);
