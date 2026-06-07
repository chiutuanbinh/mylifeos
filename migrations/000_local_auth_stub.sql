-- Local dev stub for Supabase auth schema.
-- Supabase provides auth.uid() natively; local Postgres does not.
-- This stub makes 002_rls.sql run cleanly in docker-compose.
-- DO NOT run this on Supabase — it will be ignored if auth schema already exists.
CREATE SCHEMA IF NOT EXISTS auth;

CREATE OR REPLACE FUNCTION auth.uid() RETURNS uuid AS $$
  SELECT '00000000-0000-0000-0000-000000000001'::uuid;
$$ LANGUAGE sql;
