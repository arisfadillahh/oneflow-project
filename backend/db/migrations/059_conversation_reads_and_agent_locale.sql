ALTER TABLE agents
  ADD COLUMN IF NOT EXISTS preferred_locale TEXT NOT NULL DEFAULT 'id';

UPDATE agents
SET preferred_locale = 'id'
WHERE preferred_locale NOT IN ('id', 'en');

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'agents_preferred_locale_check'
  ) THEN
    ALTER TABLE agents
      ADD CONSTRAINT agents_preferred_locale_check
      CHECK (preferred_locale IN ('id', 'en'));
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS conversation_reads (
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  last_read_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (conversation_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_reads_org_agent
  ON conversation_reads (organization_id, agent_id, last_read_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_unread_inbound
  ON messages (organization_id, conversation_id, direction, sent_at DESC, created_at DESC);
