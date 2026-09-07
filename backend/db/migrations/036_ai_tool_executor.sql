CREATE TABLE IF NOT EXISTS ai_tool_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ai_run_id UUID REFERENCES ai_runs(id) ON DELETE SET NULL,
  tool_name TEXT NOT NULL,
  idempotency_key TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'running',
  request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  response_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  error_message TEXT,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMPTZ,
  CHECK (tool_name IN ('check_stock', 'create_order_draft')),
  CHECK (status IN ('running', 'succeeded', 'failed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_tool_executions_idempotency
  ON ai_tool_executions (organization_id, tool_name, idempotency_key)
  WHERE idempotency_key <> '';

CREATE INDEX IF NOT EXISTS idx_ai_tool_executions_org_created
  ON ai_tool_executions (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_tool_executions_run
  ON ai_tool_executions (ai_run_id, created_at DESC)
  WHERE ai_run_id IS NOT NULL;
