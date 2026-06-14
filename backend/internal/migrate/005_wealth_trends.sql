ALTER TABLE net_worth_snapshots
  ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS benchmark_data (
  id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,
  date   DATE NOT NULL,
  value  NUMERIC(18,4) NOT NULL,
  UNIQUE(source, date)
);

CREATE TABLE IF NOT EXISTS news_cache (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source       TEXT NOT NULL,
  published_at TIMESTAMPTZ NOT NULL,
  title        TEXT NOT NULL,
  url          TEXT NOT NULL UNIQUE,
  fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
