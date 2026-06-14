ALTER TABLE net_worth_snapshots
  ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS benchmark_data (
  id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,
  date   DATE NOT NULL,
  value  NUMERIC(18,4) NOT NULL,
  UNIQUE(source, date)
);
ALTER TABLE benchmark_data ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS benchmark_data_read ON benchmark_data;
CREATE POLICY benchmark_data_read ON benchmark_data FOR SELECT USING (true);

CREATE TABLE IF NOT EXISTS news_cache (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source       TEXT NOT NULL,
  published_at TIMESTAMPTZ NOT NULL,
  title        TEXT NOT NULL,
  url          TEXT NOT NULL UNIQUE,
  fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE news_cache ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS news_cache_read ON news_cache;
CREATE POLICY news_cache_read ON news_cache FOR SELECT USING (true);
