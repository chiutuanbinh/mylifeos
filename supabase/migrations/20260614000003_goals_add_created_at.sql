-- Add missing created_at column to goals table
-- The original CREATE TABLE IF NOT EXISTS was skipped on instances
-- where the table already existed without this column.
ALTER TABLE goals
  ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();
