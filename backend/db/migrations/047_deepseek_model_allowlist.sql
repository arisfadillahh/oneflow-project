ALTER TABLE ai_agents
  DROP CONSTRAINT IF EXISTS ai_agents_model_name_openrouter_gpt_claude_check;

ALTER TABLE ai_agents
  DROP CONSTRAINT IF EXISTS ai_agents_model_name_openrouter_supported_check;

ALTER TABLE ai_agents
  ADD CONSTRAINT ai_agents_model_name_openrouter_supported_check
  CHECK (
    char_length(trim(model_name)) BETWEEN 1 AND 160
    AND (
      model_name ~* '^openai/.*gpt'
      OR model_name ~* '^anthropic/.*claude'
      OR model_name ~* '^deepseek/.*deepseek'
    )
  );

INSERT INTO ai_model_aliases (model_id, display_name, is_available, updated_at)
VALUES
  ('deepseek/deepseek-v4-flash', 'DeepSeek V4 Flash', FALSE, NOW()),
  ('deepseek/deepseek-v4-pro', 'DeepSeek V4 Pro', FALSE, NOW()),
  ('deepseek/deepseek-v3.2', 'DeepSeek V3.2', FALSE, NOW()),
  ('deepseek/deepseek-chat-v3.1', 'DeepSeek V3.1', FALSE, NOW()),
  ('deepseek/deepseek-r1', 'DeepSeek R1', FALSE, NOW())
ON CONFLICT (model_id) DO NOTHING;
