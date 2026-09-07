ALTER TABLE ai_agents
  ADD COLUMN IF NOT EXISTS customer_memory_llm_validator_enabled BOOLEAN NOT NULL DEFAULT FALSE;
