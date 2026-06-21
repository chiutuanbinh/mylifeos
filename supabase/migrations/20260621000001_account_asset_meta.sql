ALTER TABLE accounts
  ADD COLUMN IF NOT EXISTS purchase_value    NUMERIC,
  ADD COLUMN IF NOT EXISTS purchased_at      DATE,
  ADD COLUMN IF NOT EXISTS depreciation_rate NUMERIC,
  ADD COLUMN IF NOT EXISTS asset_notes       TEXT;
