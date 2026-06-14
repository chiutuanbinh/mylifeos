-- Add recurring + reminder_time to key_results
ALTER TABLE key_results
  ADD COLUMN IF NOT EXISTS recurring      BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS reminder_time  TIME    DEFAULT NULL;

-- KR logs table (replaces habit_logs, keyed by kr_id)
CREATE TABLE IF NOT EXISTS kr_logs (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kr_id       UUID NOT NULL REFERENCES key_results(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL,
  logged_date DATE NOT NULL,
  done        BOOLEAN NOT NULL DEFAULT TRUE,
  UNIQUE(kr_id, logged_date)
);

ALTER TABLE kr_logs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS kr_logs_user ON kr_logs;
CREATE POLICY kr_logs_user ON kr_logs USING (user_id = auth.uid());

-- Migrate habits → recurring KRs under a "Daily Routines" goal per user
DO $$
DECLARE
  u        RECORD;
  goal_id  UUID;
  h        RECORD;
  new_kr   UUID;
BEGIN
  FOR u IN SELECT DISTINCT user_id FROM habits LOOP
    -- Create (or find existing) "Daily Routines" goal
    INSERT INTO goals (user_id, name, description, color, status)
    VALUES (u.user_id, 'Daily Routines', 'Auto-migrated from habits', '#52c41a', 'active')
    ON CONFLICT DO NOTHING
    RETURNING id INTO goal_id;

    IF goal_id IS NULL THEN
      SELECT id INTO goal_id
      FROM goals
      WHERE user_id = u.user_id AND name = 'Daily Routines'
      ORDER BY created_at
      LIMIT 1;
    END IF;

    -- Insert each habit as a recurring KR
    FOR h IN SELECT * FROM habits WHERE user_id = u.user_id LOOP
      INSERT INTO key_results (goal_id, user_id, description, done, recurring)
      VALUES (goal_id, u.user_id, h.icon || ' ' || h.name, FALSE, TRUE)
      RETURNING id INTO new_kr;

      -- Copy habit_logs → kr_logs
      INSERT INTO kr_logs (kr_id, user_id, logged_date, done)
      SELECT new_kr, hl.user_id, hl.logged_date, hl.done
      FROM habit_logs hl
      WHERE hl.habit_id = h.id
      ON CONFLICT (kr_id, logged_date) DO NOTHING;
    END LOOP;
  END LOOP;
END $$;

-- Drop old tables
DROP TABLE IF EXISTS habit_logs;
DROP TABLE IF EXISTS habits;
