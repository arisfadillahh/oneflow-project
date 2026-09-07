CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'active',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS organization_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  invited_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (organization_id, agent_id),
  CHECK (role IN ('org_owner', 'org_admin', 'hr_manager', 'hr_agent')),
  CHECK (status IN ('active', 'invited', 'disabled'))
);

CREATE TABLE IF NOT EXISTS organization_invites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email TEXT,
  phone TEXT,
  role TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'pending',
  invited_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  accepted_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (role IN ('org_owner', 'org_admin', 'hr_manager', 'hr_agent')),
  CHECK (status IN ('pending', 'accepted', 'revoked', 'expired'))
);

ALTER TABLE agents ADD COLUMN IF NOT EXISTS current_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS platform_role TEXT NOT NULL DEFAULT 'none';
ALTER TABLE agents ADD CONSTRAINT agents_platform_role_check CHECK (platform_role IN ('none', 'platform_owner', 'platform_admin'));

INSERT INTO organizations (name, slug)
SELECT 'Default Company', 'default-company'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'default-company');

WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE agents
SET current_organization_id = (SELECT id FROM default_org)
WHERE current_organization_id IS NULL;

WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
INSERT INTO organization_members (organization_id, agent_id, role, status)
SELECT
  default_org.id,
  agents.id,
  CASE agents.role::text
    WHEN 'owner' THEN 'org_owner'
    WHEN 'super_admin' THEN 'org_admin'
    WHEN 'admin' THEN 'org_admin'
    ELSE 'hr_agent'
  END,
  CASE WHEN agents.is_active THEN 'active' ELSE 'disabled' END
FROM agents
CROSS JOIN default_org
ON CONFLICT (organization_id, agent_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS whatsapp_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  label TEXT NOT NULL,
  phone_number TEXT,
  mode TEXT NOT NULL DEFAULT 'mock',
  status TEXT NOT NULL DEFAULT 'disconnected',
  session_key TEXT NOT NULL UNIQUE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  last_connected_at TIMESTAMPTZ,
  last_disconnected_at TIMESTAMPTZ,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (mode IN ('mock', 'real')),
  CHECK (status IN ('disconnected', 'connecting', 'qr_pending', 'connected', 'error'))
);

WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
INSERT INTO whatsapp_sessions (organization_id, label, mode, status, session_key, is_default)
SELECT default_org.id, 'Default WhatsApp', 'real', 'disconnected', default_org.id::text || ':default', TRUE
FROM default_org
WHERE NOT EXISTS (
  SELECT 1 FROM whatsapp_sessions WHERE organization_id = default_org.id AND is_default = TRUE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_sessions_one_default
  ON whatsapp_sessions (organization_id)
  WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_org_status
  ON whatsapp_sessions (organization_id, status, updated_at DESC);

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS whatsapp_session_id UUID REFERENCES whatsapp_sessions(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE conversation_events ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE ai_settings ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE ai_runs ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE knowledge_faqs ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE knowledge_documents ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE job_positions ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE knowledge_records ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE credit_wallet ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE credit_usage_logs ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE credit_adjustments ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE credit_pricing_settings ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE system_alerts ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE wa_escalation_group_bindings ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE playground_shortcuts ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE contacts SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE conversations SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_session AS (
  SELECT id, organization_id FROM whatsapp_sessions WHERE is_default = TRUE
)
UPDATE conversations c
SET whatsapp_session_id = ds.id
FROM default_session ds
WHERE c.organization_id = ds.organization_id AND c.whatsapp_session_id IS NULL;
UPDATE messages m
SET organization_id = c.organization_id
FROM conversations c
WHERE m.conversation_id = c.id AND m.organization_id IS NULL;
UPDATE conversation_events e
SET organization_id = c.organization_id
FROM conversations c
WHERE e.conversation_id = c.id AND e.organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE ai_settings SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
UPDATE ai_runs r
SET organization_id = c.organization_id
FROM conversations c
WHERE r.conversation_id = c.id AND r.organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE knowledge_faqs SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE knowledge_documents SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE job_positions SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE knowledge_records SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE credit_wallet SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE credit_usage_logs SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE credit_packages SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE credit_purchases SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE credit_adjustments SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE credit_pricing_settings SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE audit_logs SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE system_alerts SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE wa_escalation_group_bindings SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;
WITH default_org AS (
  SELECT id FROM organizations WHERE slug = 'default-company'
)
UPDATE playground_shortcuts SET organization_id = (SELECT id FROM default_org) WHERE organization_id IS NULL;

ALTER TABLE contacts ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE conversations ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE messages ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE conversation_events ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE ai_settings ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE ai_runs ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE knowledge_faqs ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE knowledge_documents ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE job_positions ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE knowledge_records ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE credit_wallet ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE credit_usage_logs ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE credit_packages ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE credit_purchases ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE credit_adjustments ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE credit_pricing_settings ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE wa_escalation_group_bindings ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE playground_shortcuts ALTER COLUMN organization_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_agents_current_organization_id ON agents (current_organization_id);
CREATE INDEX IF NOT EXISTS idx_org_members_agent_status ON organization_members (agent_id, status);
CREATE INDEX IF NOT EXISTS idx_org_members_org_role ON organization_members (organization_id, role, status);
CREATE INDEX IF NOT EXISTS idx_org_invites_org_status ON organization_invites (organization_id, status, expires_at);

CREATE INDEX IF NOT EXISTS idx_contacts_org_phone ON contacts (organization_id, phone);
ALTER TABLE contacts DROP CONSTRAINT IF EXISTS contacts_phone_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_org_phone_unique ON contacts (organization_id, phone);
CREATE INDEX IF NOT EXISTS idx_conversations_org_status ON conversations (organization_id, status, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_org_session ON conversations (organization_id, whatsapp_session_id, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_org_conversation ON messages (organization_id, conversation_id, created_at);
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_external_message_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_org_external_id_unique
  ON messages (organization_id, external_message_id)
  WHERE external_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversation_events_org_conversation ON conversation_events (organization_id, conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ai_settings_org_active ON ai_settings (organization_id, is_active, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_runs_org_created ON ai_runs (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_knowledge_faqs_org_status ON knowledge_faqs (organization_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_org_status ON knowledge_documents (organization_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_positions_org_active ON job_positions (organization_id, is_active, status);
CREATE INDEX IF NOT EXISTS idx_knowledge_records_org_status ON knowledge_records (organization_id, status, priority DESC);
CREATE INDEX IF NOT EXISTS idx_credit_wallet_org_active ON credit_wallet (organization_id, is_active);
CREATE INDEX IF NOT EXISTS idx_credit_usage_logs_org_created ON credit_usage_logs (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_credit_packages_org_active ON credit_packages (organization_id, is_active);
CREATE INDEX IF NOT EXISTS idx_credit_purchases_org_status ON credit_purchases (organization_id, payment_status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_alerts_org_status ON system_alerts (organization_id, status, severity);
CREATE INDEX IF NOT EXISTS idx_wa_escalation_bindings_org_status ON wa_escalation_group_bindings (organization_id, status, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_playground_shortcuts_org_active ON playground_shortcuts (organization_id, is_active, sort_order);
