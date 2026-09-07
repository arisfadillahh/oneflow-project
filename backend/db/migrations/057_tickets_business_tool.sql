ALTER TABLE organization_modules
  DROP CONSTRAINT IF EXISTS organization_modules_module_key_check;

ALTER TABLE organization_modules
  ADD CONSTRAINT organization_modules_module_key_check
  CHECK (module_key IN ('prospects', 'commerce', 'orders', 'booking', 'payments', 'tickets'));

CREATE TABLE IF NOT EXISTS tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ticket_number TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  issue_type TEXT NOT NULL DEFAULT 'other',
  status TEXT NOT NULL DEFAULT 'new',
  priority TEXT NOT NULL DEFAULT 'normal',
  severity TEXT NOT NULL DEFAULT 'minor',
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  whatsapp_session_id UUID REFERENCES whatsapp_sessions(id) ON DELETE SET NULL,
  ai_agent_id UUID REFERENCES ai_agents(id) ON DELETE SET NULL,
  assigned_to UUID REFERENCES agents(id) ON DELETE SET NULL,
  source TEXT NOT NULL DEFAULT 'dashboard',
  sla_due_at TIMESTAMPTZ,
  resolved_at TIMESTAMPTZ,
  closed_at TIMESTAMPTZ,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (organization_id, ticket_number),
  CHECK (status IN ('new', 'triage', 'waiting_customer', 'waiting_internal', 'in_progress', 'resolved', 'closed', 'cancelled')),
  CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
  CHECK (severity IN ('minor', 'moderate', 'major', 'critical')),
  CHECK (issue_type IN ('complaint', 'product_question', 'payment', 'delivery', 'booking', 'technical', 'refund', 'other'))
);

CREATE INDEX IF NOT EXISTS idx_tickets_org_status_updated
  ON tickets (organization_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_tickets_org_priority_sla
  ON tickets (organization_id, priority, sla_due_at)
  WHERE status NOT IN ('resolved', 'closed', 'cancelled');

CREATE INDEX IF NOT EXISTS idx_tickets_org_contact
  ON tickets (organization_id, contact_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tickets_org_conversation
  ON tickets (organization_id, conversation_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ticket_comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  is_internal BOOLEAN NOT NULL DEFAULT TRUE,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_comments_ticket_created
  ON ticket_comments (organization_id, ticket_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ticket_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_events_ticket_created
  ON ticket_events (organization_id, ticket_id, created_at DESC);
