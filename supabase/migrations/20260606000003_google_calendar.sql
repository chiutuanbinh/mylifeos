ALTER TABLE events ADD COLUMN IF NOT EXISTS google_event_id text;
CREATE UNIQUE INDEX IF NOT EXISTS events_google_event_id_idx ON events (google_event_id) WHERE google_event_id IS NOT NULL;
