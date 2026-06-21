CREATE TABLE journal_entry_goals (
  entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
  goal_id  UUID NOT NULL REFERENCES goals(id)           ON DELETE CASCADE,
  user_id  TEXT NOT NULL,
  PRIMARY KEY (entry_id, goal_id)
);

CREATE INDEX idx_jeg_entry_id ON journal_entry_goals(entry_id);
CREATE INDEX idx_jeg_goal_id  ON journal_entry_goals(goal_id);
CREATE INDEX idx_jeg_user_id  ON journal_entry_goals(user_id);

ALTER TABLE journal_entry_goals ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users own their journal_entry_goals"
  ON journal_entry_goals FOR ALL
  USING (user_id = auth.uid()::text);
