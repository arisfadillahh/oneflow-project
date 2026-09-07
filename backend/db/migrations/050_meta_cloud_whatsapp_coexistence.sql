ALTER TABLE whatsapp_sessions
  ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'whatsmeow',
  ADD COLUMN IF NOT EXISTS onboarding_mode TEXT,
  ADD COLUMN IF NOT EXISTS meta_waba_id TEXT,
  ADD COLUMN IF NOT EXISTS meta_phone_number_id TEXT,
  ADD COLUMN IF NOT EXISTS meta_business_token_ciphertext TEXT,
  ADD COLUMN IF NOT EXISTS meta_webhook_subscribed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS meta_onboarded_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS meta_history_sync_status TEXT NOT NULL DEFAULT 'not_requested',
  ADD COLUMN IF NOT EXISTS meta_history_sync_deadline TIMESTAMPTZ;

ALTER TABLE whatsapp_sessions
  DROP CONSTRAINT IF EXISTS whatsapp_sessions_provider_check;
ALTER TABLE whatsapp_sessions
  ADD CONSTRAINT whatsapp_sessions_provider_check
  CHECK (provider IN ('whatsmeow', 'meta_cloud'));

ALTER TABLE whatsapp_sessions
  DROP CONSTRAINT IF EXISTS whatsapp_sessions_onboarding_mode_check;
ALTER TABLE whatsapp_sessions
  ADD CONSTRAINT whatsapp_sessions_onboarding_mode_check
  CHECK (onboarding_mode IS NULL OR onboarding_mode IN ('coexistence', 'cloud_api'));

ALTER TABLE whatsapp_sessions
  DROP CONSTRAINT IF EXISTS whatsapp_sessions_meta_history_sync_status_check;
ALTER TABLE whatsapp_sessions
  ADD CONSTRAINT whatsapp_sessions_meta_history_sync_status_check
  CHECK (meta_history_sync_status IN ('not_requested', 'pending', 'received', 'declined', 'expired', 'failed'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_sessions_meta_phone_active
  ON whatsapp_sessions (meta_phone_number_id)
  WHERE provider = 'meta_cloud'
    AND deleted_at IS NULL
    AND meta_phone_number_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS whatsapp_meta_webhook_events (
  event_key TEXT PRIMARY KEY,
  whatsapp_session_id UUID REFERENCES whatsapp_sessions(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_meta_events_session_time
  ON whatsapp_meta_webhook_events (whatsapp_session_id, received_at DESC);
