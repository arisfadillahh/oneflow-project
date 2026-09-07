CREATE TABLE IF NOT EXISTS organization_modules (
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  module_key TEXT NOT NULL,
  is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  enabled_at TIMESTAMPTZ,
  enabled_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  disabled_at TIMESTAMPTZ,
  disabled_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (organization_id, module_key),
  CHECK (module_key IN ('prospects', 'commerce', 'orders', 'booking', 'payments'))
);

CREATE INDEX IF NOT EXISTS idx_organization_modules_enabled
  ON organization_modules (organization_id, is_enabled);

CREATE TABLE IF NOT EXISTS ai_run_cost_steps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ai_run_id UUID NOT NULL REFERENCES ai_runs(id) ON DELETE CASCADE,
  step_index INTEGER NOT NULL,
  step_type TEXT NOT NULL,
  step_name TEXT NOT NULL,
  provider TEXT,
  model_name TEXT,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  embedding_tokens INTEGER NOT NULL DEFAULT 0,
  cost_usd NUMERIC(12,9),
  cost_idr NUMERIC(12,4),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (ai_run_id, step_index)
);

CREATE INDEX IF NOT EXISTS idx_ai_run_cost_steps_org_created
  ON ai_run_cost_steps (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_run_cost_steps_run
  ON ai_run_cost_steps (ai_run_id, step_index);

CREATE TABLE IF NOT EXISTS commerce_products (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  sku TEXT,
  name TEXT NOT NULL,
  description TEXT,
  unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
  stock_quantity INTEGER NOT NULL DEFAULT 0,
  reserved_quantity INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('active', 'inactive', 'archived')),
  CHECK (stock_quantity >= 0),
  CHECK (reserved_quantity >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_products_org_sku
  ON commerce_products (organization_id, lower(sku))
  WHERE NULLIF(sku, '') IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_commerce_products_org_status
  ON commerce_products (organization_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS commerce_order_drafts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  customer_name TEXT,
  customer_phone TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  source TEXT NOT NULL DEFAULT 'dashboard',
  total_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  notes TEXT,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  confirmed_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('draft', 'awaiting_confirmation', 'confirmed', 'cancelled')),
  CHECK (total_amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_commerce_order_drafts_org_status
  ON commerce_order_drafts (organization_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_commerce_order_drafts_contact
  ON commerce_order_drafts (organization_id, contact_id, created_at DESC);

CREATE TABLE IF NOT EXISTS commerce_order_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_draft_id UUID NOT NULL REFERENCES commerce_order_drafts(id) ON DELETE CASCADE,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  product_id UUID REFERENCES commerce_products(id) ON DELETE SET NULL,
  product_name TEXT NOT NULL,
  sku TEXT,
  quantity INTEGER NOT NULL,
  unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
  line_total NUMERIC(12,2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (quantity > 0),
  CHECK (unit_price >= 0),
  CHECK (line_total >= 0)
);

CREATE INDEX IF NOT EXISTS idx_commerce_order_items_draft
  ON commerce_order_items (organization_id, order_draft_id);
