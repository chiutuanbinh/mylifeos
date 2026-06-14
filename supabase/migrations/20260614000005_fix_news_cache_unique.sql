-- Ensure url unique constraint exists (idempotent)
-- Needed because CREATE TABLE IF NOT EXISTS skips schema changes if table pre-existed
CREATE UNIQUE INDEX IF NOT EXISTS news_cache_url_key ON news_cache(url);
