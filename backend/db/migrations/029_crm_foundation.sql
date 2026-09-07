ALTER TABLE contacts
  ADD COLUMN IF NOT EXISTS lifecycle_status TEXT NOT NULL DEFAULT 'lead',
  ADD COLUMN IF NOT EXISTS owner_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS last_summary TEXT,
  ADD COLUMN IF NOT EXISTS summary_updated_at TIMESTAMPTZ;

ALTER TABLE contacts
  DROP CONSTRAINT IF EXISTS contacts_lifecycle_status_check;

ALTER TABLE contacts
  ADD CONSTRAINT contacts_lifecycle_status_check
  CHECK (lifecycle_status IN ('lead', 'prospect', 'customer', 'inactive'));

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal',
  ADD COLUMN IF NOT EXISTS sla_due_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS internal_note TEXT,
  ADD COLUMN IF NOT EXISTS workflow_updated_at TIMESTAMPTZ;

ALTER TABLE conversations
  DROP CONSTRAINT IF EXISTS conversations_priority_check;

ALTER TABLE conversations
  ADD CONSTRAINT conversations_priority_check
  CHECK (priority IN ('low', 'normal', 'high', 'urgent'));

DO $$
BEGIN
  ALTER TYPE conversation_event_type ADD VALUE IF NOT EXISTS 'workflow_updated';
EXCEPTION WHEN duplicate_object THEN
  NULL;
END $$;

CREATE TABLE IF NOT EXISTS contact_tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#2563eb',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_tags_org_name_unique
  ON contact_tags (organization_id, lower(name));

CREATE TABLE IF NOT EXISTS contact_tag_links (
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  tag_id UUID NOT NULL REFERENCES contact_tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (contact_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_contact_tag_links_org_contact
  ON contact_tag_links (organization_id, contact_id);

CREATE TABLE IF NOT EXISTS contact_notes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contact_notes_org_contact_created
  ON contact_notes (organization_id, contact_id, created_at DESC);

CREATE TABLE IF NOT EXISTS follow_up_tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  notes TEXT,
  due_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'open',
  priority TEXT NOT NULL DEFAULT 'normal',
  assigned_to UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('open', 'done', 'cancelled')),
  CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
  CHECK (contact_id IS NOT NULL OR conversation_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_follow_up_tasks_org_status_due
  ON follow_up_tasks (organization_id, status, due_at);

CREATE INDEX IF NOT EXISTS idx_follow_up_tasks_org_contact
  ON follow_up_tasks (organization_id, contact_id, created_at DESC);

CREATE TABLE IF NOT EXISTS whatsapp_message_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'follow_up',
  body TEXT NOT NULL,
  variables JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'draft',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('draft', 'active', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_templates_org_name_unique
  ON whatsapp_message_templates (organization_id, lower(name))
  WHERE status <> 'archived';

CREATE TABLE IF NOT EXISTS whatsapp_broadcasts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  template_id UUID REFERENCES whatsapp_message_templates(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  recipient_count INTEGER NOT NULL DEFAULT 0,
  sent_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  sent_at TIMESTAMPTZ,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('draft', 'sending', 'sent', 'partial', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_broadcasts_org_created
  ON whatsapp_broadcasts (organization_id, created_at DESC);

CREATE TABLE IF NOT EXISTS whatsapp_broadcast_recipients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  broadcast_id UUID NOT NULL REFERENCES whatsapp_broadcasts(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  phone TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  error TEXT,
  sent_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (broadcast_id, contact_id),
  CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_broadcast_recipients_org_broadcast
  ON whatsapp_broadcast_recipients (organization_id, broadcast_id);

CREATE INDEX IF NOT EXISTS idx_contacts_org_lifecycle
  ON contacts (organization_id, lifecycle_status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_contacts_org_owner
  ON contacts (organization_id, owner_agent_id);

CREATE INDEX IF NOT EXISTS idx_conversations_org_priority_sla
  ON conversations (organization_id, priority, sla_due_at);
