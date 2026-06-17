CREATE TYPE account_type AS ENUM ('asset', 'liability', 'equity', 'income', 'expense');
CREATE TYPE journal_side AS ENUM ('debit', 'credit');

CREATE TABLE IF NOT EXISTS accounts (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL,
  parent_id   uuid        REFERENCES accounts(id) ON DELETE RESTRICT,
  name        text        NOT NULL,
  type        account_type NOT NULL,
  currency    text        NOT NULL DEFAULT 'VND',
  is_group    boolean     NOT NULL DEFAULT false,
  archived    boolean     NOT NULL DEFAULT false,
  sort_order  int         NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS journal_entries (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL,
  date        date        NOT NULL,
  description text        NOT NULL,
  memo        text        NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS journal_lines (
  id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
  entry_id    uuid         NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
  account_id  uuid         NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
  amount      numeric(15,2) NOT NULL CHECK (amount > 0),
  currency    text         NOT NULL DEFAULT 'VND',
  side        journal_side NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_accounts_user ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_journal_entries_user_date ON journal_entries(user_id, date);
CREATE INDEX IF NOT EXISTS idx_journal_lines_entry ON journal_lines(entry_id);
CREATE INDEX IF NOT EXISTS idx_journal_lines_account ON journal_lines(account_id);
