-- Finance
CREATE TABLE IF NOT EXISTS transactions (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL,
  date        date NOT NULL,
  description text NOT NULL,
  category    text NOT NULL,
  amount      numeric(12,2) NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS budgets (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       uuid NOT NULL,
  category      text NOT NULL,
  monthly_limit numeric(12,2) NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, category)
);

-- Health
CREATE TABLE IF NOT EXISTS habits (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL,
  name       text NOT NULL,
  icon       text NOT NULL DEFAULT '✓',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS habit_logs (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  habit_id    uuid NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL,
  logged_date date NOT NULL,
  done        boolean NOT NULL DEFAULT true,
  UNIQUE (habit_id, logged_date)
);

-- Goals
CREATE TABLE IF NOT EXISTS goals (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL,
  name        text NOT NULL,
  description text NOT NULL DEFAULT '',
  target_date date,
  progress    int NOT NULL DEFAULT 0,
  color       text NOT NULL DEFAULT '#1677ff',
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS key_results (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id     uuid NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL,
  description text NOT NULL,
  done        boolean NOT NULL DEFAULT false
);

-- Notes
CREATE TABLE IF NOT EXISTS notes (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL,
  title      text NOT NULL,
  content    text NOT NULL DEFAULT '',
  tags       text[] NOT NULL DEFAULT '{}',
  pinned     boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Calendar
CREATE TABLE IF NOT EXISTS events (
  id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id  uuid NOT NULL,
  title    text NOT NULL,
  start_at timestamptz NOT NULL,
  end_at   timestamptz NOT NULL,
  color    text NOT NULL DEFAULT '#1677ff',
  all_day  boolean NOT NULL DEFAULT false
);

-- Inventory
CREATE TABLE IF NOT EXISTS assets (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      uuid NOT NULL,
  name         text NOT NULL,
  category     text NOT NULL,
  value        numeric(12,2) NOT NULL DEFAULT 0,
  purchased_at date,
  notes        text NOT NULL DEFAULT ''
);

-- Settings
CREATE TABLE IF NOT EXISTS user_settings (
  user_id         uuid PRIMARY KEY,
  notifications   jsonb NOT NULL DEFAULT '{"email": true, "push": false}'::jsonb,
  modules_enabled jsonb NOT NULL DEFAULT '{"finance": true, "health": true, "goals": true, "notes": true, "calendar": true, "inventory": true}'::jsonb
);
