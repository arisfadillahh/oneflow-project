ALTER TABLE job_positions
  ADD COLUMN IF NOT EXISTS ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_job_positions_org_ai_agent_active
  ON job_positions (organization_id, ai_agent_id, is_active, status);

UPDATE knowledge_records kr
SET ai_agent_id = jp.ai_agent_id
FROM job_positions jp
WHERE kr.id = jp.id
  AND kr.organization_id = jp.organization_id
  AND kr.ai_agent_id IS NULL
  AND jp.ai_agent_id IS NOT NULL
  AND kr.metadata_json->>'legacy_table' = 'job_positions';
