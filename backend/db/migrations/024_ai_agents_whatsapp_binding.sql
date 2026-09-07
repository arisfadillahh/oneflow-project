CREATE TABLE IF NOT EXISTS ai_agents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  model_name TEXT NOT NULL DEFAULT 'openai/gpt-4o-mini',
  system_prompt TEXT NOT NULL DEFAULT '',
  escalation_prompt TEXT NOT NULL DEFAULT '',
  fallback_waiting_message TEXT NOT NULL DEFAULT 'Pesan Anda sudah kami teruskan ke tim kami. Mohon tunggu sebentar ya.',
  allow_clarification BOOLEAN NOT NULL DEFAULT TRUE,
  max_clarification_count INTEGER NOT NULL DEFAULT 1,
  answer_only_from_knowledge BOOLEAN NOT NULL DEFAULT TRUE,
  dont_broaden_topic BOOLEAN NOT NULL DEFAULT TRUE,
  forbid_promises BOOLEAN NOT NULL DEFAULT TRUE,
  forbid_sensitive_answers BOOLEAN NOT NULL DEFAULT TRUE,
  require_action_confirmation BOOLEAN NOT NULL DEFAULT TRUE,
  escalate_low_confidence BOOLEAN NOT NULL DEFAULT TRUE,
  guide_next_step BOOLEAN NOT NULL DEFAULT TRUE,
  concise_response BOOLEAN NOT NULL DEFAULT TRUE,
  allow_auto_update_contact_name BOOLEAN NOT NULL DEFAULT FALSE,
  only_fill_name_if_empty BOOLEAN NOT NULL DEFAULT TRUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (model_name IN ('openai/gpt-4o-mini', 'openai/gpt-4.1-mini')),
  CHECK (max_clarification_count BETWEEN 0 AND 5)
);

CREATE INDEX IF NOT EXISTS idx_ai_agents_org_active
  ON ai_agents (organization_id, is_active, updated_at DESC);

ALTER TABLE whatsapp_sessions
  ADD COLUMN IF NOT EXISTS ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE RESTRICT;

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE SET NULL;

ALTER TABLE knowledge_faqs
  ADD COLUMN IF NOT EXISTS ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE CASCADE;

ALTER TABLE knowledge_documents
  ADD COLUMN IF NOT EXISTS ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE CASCADE;

ALTER TABLE knowledge_records
  ADD COLUMN IF NOT EXISTS ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE CASCADE;

ALTER TABLE wa_escalation_group_bindings
  ADD COLUMN IF NOT EXISTS whatsapp_session_id UUID REFERENCES whatsapp_sessions(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_org_ai_agent
  ON whatsapp_sessions (organization_id, ai_agent_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversations_org_ai_agent
  ON conversations (organization_id, ai_agent_id, last_message_at DESC);

CREATE INDEX IF NOT EXISTS idx_knowledge_faqs_org_ai_agent_status
  ON knowledge_faqs (organization_id, ai_agent_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_knowledge_documents_org_ai_agent_status
  ON knowledge_documents (organization_id, ai_agent_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_knowledge_records_org_ai_agent_status
  ON knowledge_records (organization_id, ai_agent_id, status, priority DESC);

CREATE INDEX IF NOT EXISTS idx_wa_escalation_bindings_org_session_status
  ON wa_escalation_group_bindings (organization_id, whatsapp_session_id, status, expires_at DESC);
