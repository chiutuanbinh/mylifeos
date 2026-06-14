-- Fix user_id column type: TEXT → UUID (matches Supabase auth.users.id)
ALTER TABLE IF EXISTS liabilities ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
