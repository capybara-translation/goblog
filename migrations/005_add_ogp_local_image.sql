-- Add local_image_path column to ogp_cache table for caching OGP images locally
ALTER TABLE ogp_cache ADD COLUMN local_image_path TEXT;
