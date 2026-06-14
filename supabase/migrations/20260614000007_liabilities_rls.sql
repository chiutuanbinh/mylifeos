-- Fix user_id TEXT→UUID first so auth.uid() (UUID) comparison works
ALTER TABLE liabilities ALTER COLUMN user_id TYPE UUID USING user_id::uuid;

ALTER TABLE liabilities ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS liabilities_user ON liabilities;
CREATE POLICY liabilities_user ON liabilities USING (user_id = auth.uid());
