-- Enable RLS on all tables
ALTER TABLE transactions    ENABLE ROW LEVEL SECURITY;
ALTER TABLE budgets         ENABLE ROW LEVEL SECURITY;
ALTER TABLE habits          ENABLE ROW LEVEL SECURITY;
ALTER TABLE habit_logs      ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals           ENABLE ROW LEVEL SECURITY;
ALTER TABLE key_results     ENABLE ROW LEVEL SECURITY;
ALTER TABLE notes           ENABLE ROW LEVEL SECURITY;
ALTER TABLE events          ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets          ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_settings   ENABLE ROW LEVEL SECURITY;

-- Policies: users only see their own rows
CREATE POLICY transactions_user    ON transactions    USING (user_id = auth.uid());
CREATE POLICY budgets_user         ON budgets         USING (user_id = auth.uid());
CREATE POLICY habits_user          ON habits          USING (user_id = auth.uid());
CREATE POLICY habit_logs_user      ON habit_logs      USING (user_id = auth.uid());
CREATE POLICY goals_user           ON goals           USING (user_id = auth.uid());
CREATE POLICY key_results_user     ON key_results     USING (user_id = auth.uid());
CREATE POLICY notes_user           ON notes           USING (user_id = auth.uid());
CREATE POLICY events_user          ON events          USING (user_id = auth.uid());
CREATE POLICY assets_user          ON assets          USING (user_id = auth.uid());
CREATE POLICY user_settings_user   ON user_settings   USING (user_id = auth.uid());
