ALTER TABLE ai_agents
  DROP CONSTRAINT IF EXISTS ai_agents_model_name_check;

ALTER TABLE ai_agents
  DROP CONSTRAINT IF EXISTS ai_agents_model_name_openrouter_gpt_claude_check;

ALTER TABLE ai_agents
  ADD CONSTRAINT ai_agents_model_name_openrouter_gpt_claude_check
  CHECK (
    char_length(trim(model_name)) BETWEEN 1 AND 160
    AND (
      model_name ~* '^~?openai/.*gpt'
      OR model_name ~* '^~?anthropic/.*claude'
    )
  );

INSERT INTO ai_model_aliases (model_id, display_name)
VALUES
  ('~openai/gpt-mini-latest', 'Oneflow.id GPT Mini Latest'),
  ('~openai/gpt-latest', 'Oneflow.id GPT Latest'),
  ('~anthropic/claude-haiku-latest', 'Oneflow.id Claude Haiku'),
  ('~anthropic/claude-sonnet-latest', 'Oneflow.id Claude Sonnet'),
  ('~anthropic/claude-opus-latest', 'Oneflow.id Claude Opus')
ON CONFLICT (model_id) DO NOTHING;
