CREATE TABLE IF NOT EXISTS group_notification_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  trigger_key TEXT NOT NULL,
  is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  template_text TEXT NOT NULL,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (trigger_key IN (
    'human_handoff',
    'ai_answer_curation',
    'commerce_order_created',
    'commerce_order_confirmed',
    'booking_created',
    'booking_updated'
  )),
  CHECK (char_length(template_text) BETWEEN 1 AND 2000)
);

ALTER TABLE group_notification_rules
  DROP CONSTRAINT IF EXISTS group_notification_rules_trigger_key_check;

ALTER TABLE group_notification_rules
  ADD CONSTRAINT group_notification_rules_trigger_key_check
  CHECK (trigger_key IN (
    'human_handoff',
    'ai_answer_curation',
    'commerce_order_created',
    'commerce_order_confirmed',
    'booking_created',
    'booking_updated'
  ));

CREATE UNIQUE INDEX IF NOT EXISTS idx_group_notification_rules_org_trigger
  ON group_notification_rules (organization_id, trigger_key);
