ALTER TABLE ai_model_aliases
  ADD COLUMN IF NOT EXISTS is_available BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE ai_agents
  ALTER COLUMN model_name SET DEFAULT 'openai/gpt-4.1-mini';

INSERT INTO ai_model_aliases (model_id, display_name, is_available)
VALUES
  ('openai/gpt-4.1-mini', 'Oneflow.id Basic Model', TRUE),
  ('anthropic/claude-haiku-4.5', 'Oneflow.id Advance Model', TRUE)
ON CONFLICT (model_id) DO UPDATE
SET is_available = TRUE,
    display_name = EXCLUDED.display_name;

UPDATE ai_model_aliases
SET is_available = FALSE
WHERE model_id = 'openai/gpt-4o-mini';

UPDATE ai_model_aliases
SET is_available = FALSE
WHERE model_id = 'anthropic/claude-sonnet-4.5';
