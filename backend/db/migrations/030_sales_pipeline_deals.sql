CREATE TABLE IF NOT EXISTS deal_pipelines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deal_pipelines_org_name_unique
  ON deal_pipelines (organization_id, lower(name));

CREATE UNIQUE INDEX IF NOT EXISTS idx_deal_pipelines_org_default_unique
  ON deal_pipelines (organization_id)
  WHERE is_default;

CREATE TABLE IF NOT EXISTS deal_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  pipeline_id UUID NOT NULL REFERENCES deal_pipelines(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  probability INTEGER NOT NULL DEFAULT 0,
  position INTEGER NOT NULL DEFAULT 0,
  stage_type TEXT NOT NULL DEFAULT 'open',
  color TEXT NOT NULL DEFAULT '#2196F3',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (probability >= 0 AND probability <= 100),
  CHECK (stage_type IN ('open', 'won', 'lost'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deal_stages_pipeline_position_unique
  ON deal_stages (pipeline_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS idx_deal_stages_pipeline_name_unique
  ON deal_stages (pipeline_id, lower(name));

CREATE INDEX IF NOT EXISTS idx_deal_stages_org_pipeline_position
  ON deal_stages (organization_id, pipeline_id, position);

CREATE TABLE IF NOT EXISTS deals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  pipeline_id UUID NOT NULL REFERENCES deal_pipelines(id) ON DELETE CASCADE,
  stage_id UUID NOT NULL REFERENCES deal_stages(id) ON DELETE RESTRICT,
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  value_amount BIGINT NOT NULL DEFAULT 0,
  currency TEXT NOT NULL DEFAULT 'IDR',
  status TEXT NOT NULL DEFAULT 'open',
  priority TEXT NOT NULL DEFAULT 'normal',
  owner_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
  expected_close_date DATE,
  source TEXT,
  notes TEXT,
  loss_reason TEXT,
  won_at TIMESTAMPTZ,
  lost_at TIMESTAMPTZ,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('open', 'won', 'lost', 'archived')),
  CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
  CHECK (currency = upper(currency))
);

CREATE INDEX IF NOT EXISTS idx_deals_org_status_updated
  ON deals (organization_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_deals_org_stage_updated
  ON deals (organization_id, stage_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_deals_org_contact_created
  ON deals (organization_id, contact_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_deals_org_owner_status
  ON deals (organization_id, owner_agent_id, status);

CREATE INDEX IF NOT EXISTS idx_deals_org_expected_close
  ON deals (organization_id, expected_close_date)
  WHERE status = 'open';

CREATE TABLE IF NOT EXISTS deal_activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
  activity_type TEXT NOT NULL DEFAULT 'note',
  title TEXT NOT NULL,
  body TEXT,
  due_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (activity_type IN ('note', 'call', 'meeting', 'task', 'stage_change'))
);

CREATE INDEX IF NOT EXISTS idx_deal_activities_org_deal_created
  ON deal_activities (organization_id, deal_id, created_at DESC);
