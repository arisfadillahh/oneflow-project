ALTER TABLE conversations DROP CONSTRAINT IF EXISTS conversations_contact_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_org_contact_channel_session_unique
  ON conversations (
    organization_id,
    contact_id,
    channel,
    COALESCE(whatsapp_session_id, '00000000-0000-0000-0000-000000000000'::uuid)
  );

CREATE TABLE IF NOT EXISTS subscription_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  monthly_credit_limit INTEGER NOT NULL DEFAULT 10000,
  price_idr NUMERIC(12,2) NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS organization_subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  plan_id UUID REFERENCES subscription_plans(id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'active',
  current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  current_period_end TIMESTAMPTZ,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('trialing', 'active', 'past_due', 'paused', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_subscriptions_org_active
  ON organization_subscriptions (organization_id)
  WHERE status IN ('trialing', 'active', 'past_due', 'paused');

INSERT INTO subscription_plans (code, name, monthly_credit_limit, price_idr)
VALUES ('starter', 'Starter', 10000, 0)
ON CONFLICT (code) DO NOTHING;

INSERT INTO organization_subscriptions (organization_id, plan_id, status)
SELECT o.id, p.id, 'active'
FROM organizations o
CROSS JOIN subscription_plans p
WHERE p.code = 'starter'
  AND NOT EXISTS (
    SELECT 1
    FROM organization_subscriptions s
    WHERE s.organization_id = o.id
      AND s.status IN ('trialing', 'active', 'past_due', 'paused')
  );

CREATE INDEX IF NOT EXISTS idx_org_subscriptions_status ON organization_subscriptions (organization_id, status);
