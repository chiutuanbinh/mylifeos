-- Fix user_id column type: TEXT → UUID (to match auth.users.id and enable RLS)
ALTER TABLE liabilities ALTER COLUMN user_id TYPE UUID USING user_id::uuid;

-- Re-apply RLS now that types match
ALTER TABLE liabilities ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS liabilities_user ON liabilities;
CREATE POLICY liabilities_user ON liabilities USING (user_id = auth.uid());
