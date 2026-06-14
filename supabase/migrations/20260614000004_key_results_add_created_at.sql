-- Add missing created_at column to key_results table
-- The key_results query in goals.List orders by created_at but the
-- original schema omitted the column definition.
ALTER TABLE key_results
  ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
