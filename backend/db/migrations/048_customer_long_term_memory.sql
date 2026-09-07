ALTER TABLE ai_settings
  ADD COLUMN IF NOT EXISTS customer_memory_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_auto_save_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_admin_notes_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_ai_extraction_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_verifier_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS customer_memory_max_items INTEGER NOT NULL DEFAULT 5,
  ADD COLUMN IF NOT EXISTS customer_memory_max_chars INTEGER NOT NULL DEFAULT 800,
  ADD COLUMN IF NOT EXISTS customer_memory_retention_days INTEGER NOT NULL DEFAULT 180;

ALTER TABLE ai_agents
  ADD COLUMN IF NOT EXISTS customer_memory_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_auto_save_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_admin_notes_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_ai_extraction_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS customer_memory_verifier_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS customer_memory_max_items INTEGER NOT NULL DEFAULT 5,
  ADD COLUMN IF NOT EXISTS customer_memory_max_chars INTEGER NOT NULL DEFAULT 800,
  ADD COLUMN IF NOT EXISTS customer_memory_retention_days INTEGER NOT NULL DEFAULT 180;

ALTER TABLE contacts
  ADD COLUMN IF NOT EXISTS customer_memory_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS customer_memory_disabled_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS contact_memory_candidates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  raw_text TEXT NOT NULL,
  extracted_fact JSONB NOT NULL DEFAULT '{}'::jsonb,
  memory_type TEXT NOT NULL,
  risk_category TEXT NOT NULL DEFAULT 'safe',
  source_type TEXT NOT NULL DEFAULT 'ai_extracted',
  gate1_result TEXT NOT NULL DEFAULT 'uncertain',
  gate1_matched_category TEXT,
  gate2_result TEXT,
  final_status TEXT NOT NULL DEFAULT 'pending',
  rejection_reason TEXT,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (memory_type IN ('preference', 'interest', 'open_issue', 'communication_preference', 'admin_note', 'context')),
  CHECK (risk_category IN ('safe', 'business_claim', 'personal_data', 'transaction_status_claim', 'system_action_claim', 'prompt_instruction', 'uncertain')),
  CHECK (source_type IN ('ai_extracted', 'admin', 'tool', 'system')),
  CHECK (gate1_result IN ('pass', 'reject', 'uncertain')),
  CHECK (gate2_result IS NULL OR gate2_result IN ('pass', 'reject', 'uncertain')),
  CHECK (final_status IN ('pending', 'approved', 'rejected', 'review'))
);

CREATE INDEX IF NOT EXISTS idx_contact_memory_candidates_org_contact_created
  ON contact_memory_candidates (organization_id, contact_id, created_at DESC);

CREATE TABLE IF NOT EXISTS contact_memories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  memory_type TEXT NOT NULL,
  value TEXT NOT NULL,
  normalized_value TEXT NOT NULL,
  source_type TEXT NOT NULL DEFAULT 'ai_extracted',
  source_candidate_id UUID REFERENCES contact_memory_candidates(id) ON DELETE SET NULL,
  source_conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  source_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  status TEXT NOT NULL DEFAULT 'active',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  CHECK (memory_type IN ('preference', 'interest', 'open_issue', 'communication_preference', 'admin_note', 'context')),
  CHECK (source_type IN ('ai_extracted', 'admin', 'tool', 'system')),
  CHECK (status IN ('active', 'disabled', 'expired', 'deleted')),
  CHECK (confidence >= 0 AND confidence <= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_memories_org_contact_type_normalized_active
  ON contact_memories (organization_id, contact_id, memory_type, normalized_value)
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_contact_memories_org_contact_status_seen
  ON contact_memories (organization_id, contact_id, status, last_seen_at DESC);
