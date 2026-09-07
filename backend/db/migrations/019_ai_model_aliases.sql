CREATE TABLE IF NOT EXISTS ai_model_aliases (
  model_id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO ai_model_aliases (model_id, display_name)
VALUES
  ('openai/gpt-4o-mini', 'Oneflow.id Basic Model'),
  ('openai/gpt-4.1-mini', 'Oneflow.id Advance Model')
ON CONFLICT (model_id) DO NOTHING;
