-- supabase/migrations/20260613000001_improvements.sql

ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS purchase_value    NUMERIC(12,2),
  ADD COLUMN IF NOT EXISTS depreciation_rate NUMERIC(5,4) NOT NULL DEFAULT 0;

ALTER TABLE assets
  ADD CONSTRAINT assets_depreciation_rate_range
    CHECK (depreciation_rate BETWEEN 0 AND 1);

ALTER TABLE goals
  ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

ALTER TABLE goals
  ADD CONSTRAINT goals_status_valid
    CHECK (status IN ('active', 'completed', 'archived'));

CREATE TABLE IF NOT EXISTS net_worth_snapshots (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL,
  snapshot_date DATE NOT NULL,
  assets_value  NUMERIC(12,2) NOT NULL,
  cash_position NUMERIC(12,2) NOT NULL,
  net_worth     NUMERIC(12,2) NOT NULL,
  UNIQUE(user_id, snapshot_date)
);

ALTER TABLE net_worth_snapshots ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS net_worth_snapshots_user ON net_worth_snapshots;
CREATE POLICY net_worth_snapshots_user ON net_worth_snapshots
  USING (user_id = auth.uid());
