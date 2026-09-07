CREATE TABLE IF NOT EXISTS wa_escalation_group_bindings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  binding_code TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  group_id TEXT,
  group_name TEXT,
  sender_jid TEXT,
  external_message_id TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  bound_at TIMESTAMPTZ,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_escalation_group_bindings_status_expires
  ON wa_escalation_group_bindings (status, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_wa_escalation_group_bindings_group_id
  ON wa_escalation_group_bindings (group_id)
  WHERE group_id IS NOT NULL;
