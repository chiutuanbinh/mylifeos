CREATE TABLE IF NOT EXISTS public.liabilities (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id            UUID NOT NULL,
  name               TEXT NOT NULL,
  category           TEXT NOT NULL,
  balance            FLOAT8 NOT NULL DEFAULT 0,
  original_principal FLOAT8,
  interest_rate      FLOAT8,
  started_at         DATE,
  due_at             DATE,
  notes              TEXT NOT NULL DEFAULT '',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS liabilities_user_id_idx ON public.liabilities(user_id);
