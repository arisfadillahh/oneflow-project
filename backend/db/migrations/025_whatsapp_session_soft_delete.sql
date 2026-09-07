ALTER TABLE whatsapp_sessions
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_org_active
  ON whatsapp_sessions (organization_id, updated_at DESC)
  WHERE deleted_at IS NULL;
