CREATE TABLE IF NOT EXISTS instagram_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ai_agent_id UUID NOT NULL REFERENCES ai_agents(id) ON DELETE RESTRICT,
  instagram_user_id TEXT NOT NULL,
  username TEXT NOT NULL,
  profile_picture_url TEXT,
  access_token_ciphertext TEXT NOT NULL,
  token_expires_at TIMESTAMPTZ,
  webhook_subscribed_at TIMESTAMPTZ,
  connected_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_instagram_sessions_account_active
  ON instagram_sessions (instagram_user_id)
  WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_instagram_sessions_org_active
  ON instagram_sessions (organization_id, updated_at DESC)
  WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS instagram_oauth_states (
  state_hash TEXT PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  ai_agent_id UUID NOT NULL REFERENCES ai_agents(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_instagram_oauth_states_expiry ON instagram_oauth_states (expires_at);

CREATE TABLE IF NOT EXISTS instagram_webhook_events (
  event_key TEXT PRIMARY KEY,
  instagram_session_id UUID REFERENCES instagram_sessions(id) ON DELETE CASCADE,
  payload_hash TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS instagram_session_id UUID REFERENCES instagram_sessions(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_instagram_session
  ON conversations (organization_id, instagram_session_id, last_message_at DESC)
  WHERE instagram_session_id IS NOT NULL;
