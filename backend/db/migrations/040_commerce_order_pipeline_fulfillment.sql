ALTER TABLE commerce_products
  ADD COLUMN IF NOT EXISTS low_stock_threshold INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
  ALTER TABLE commerce_products
    ADD CONSTRAINT commerce_products_low_stock_threshold_check CHECK (low_stock_threshold >= 0);
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS commerce_order_pipelines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_order_pipelines_org_name
  ON commerce_order_pipelines (organization_id, lower(name));

CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_order_pipelines_org_default
  ON commerce_order_pipelines (organization_id)
  WHERE is_default;

CREATE TABLE IF NOT EXISTS commerce_order_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  pipeline_id UUID NOT NULL REFERENCES commerce_order_pipelines(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  stage_type TEXT NOT NULL DEFAULT 'open',
  color TEXT NOT NULL DEFAULT '#2196F3',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (stage_type IN ('open', 'completed', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_order_stages_pipeline_position
  ON commerce_order_stages (pipeline_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS idx_commerce_order_stages_pipeline_name
  ON commerce_order_stages (pipeline_id, lower(name));

CREATE INDEX IF NOT EXISTS idx_commerce_order_stages_org_pipeline_position
  ON commerce_order_stages (organization_id, pipeline_id, position);

ALTER TABLE commerce_order_drafts
  ADD COLUMN IF NOT EXISTS stage_id UUID REFERENCES commerce_order_stages(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS fulfillment_type TEXT NOT NULL DEFAULT 'pickup',
  ADD COLUMN IF NOT EXISTS recipient_name TEXT,
  ADD COLUMN IF NOT EXISTS recipient_phone TEXT,
  ADD COLUMN IF NOT EXISTS address_line TEXT,
  ADD COLUMN IF NOT EXISTS address_area TEXT,
  ADD COLUMN IF NOT EXISTS address_notes TEXT;

DO $$
BEGIN
  ALTER TABLE commerce_order_drafts
    ADD CONSTRAINT commerce_order_drafts_fulfillment_type_check
    CHECK (fulfillment_type IN ('pickup', 'delivery', 'shipping', 'digital', 'onsite_service'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_commerce_order_drafts_org_stage
  ON commerce_order_drafts (organization_id, stage_id, created_at DESC);

CREATE TABLE IF NOT EXISTS commerce_stock_movements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES commerce_products(id) ON DELETE CASCADE,
  order_draft_id UUID REFERENCES commerce_order_drafts(id) ON DELETE SET NULL,
  movement_type TEXT NOT NULL,
  stock_delta INTEGER NOT NULL DEFAULT 0,
  reserved_delta INTEGER NOT NULL DEFAULT 0,
  stock_quantity_before INTEGER NOT NULL DEFAULT 0,
  stock_quantity_after INTEGER NOT NULL DEFAULT 0,
  reserved_quantity_before INTEGER NOT NULL DEFAULT 0,
  reserved_quantity_after INTEGER NOT NULL DEFAULT 0,
  notes TEXT,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (movement_type IN ('manual_adjustment', 'reserve_order', 'release_order'))
);

CREATE INDEX IF NOT EXISTS idx_commerce_stock_movements_org_product_created
  ON commerce_stock_movements (organization_id, product_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_commerce_stock_movements_org_order
  ON commerce_stock_movements (organization_id, order_draft_id);

ALTER TABLE booking_appointments
  ADD COLUMN IF NOT EXISTS location_type TEXT NOT NULL DEFAULT 'business_location',
  ADD COLUMN IF NOT EXISTS location_address TEXT,
  ADD COLUMN IF NOT EXISTS location_notes TEXT;

DO $$
BEGIN
  ALTER TABLE booking_appointments
    ADD CONSTRAINT booking_appointments_location_type_check
    CHECK (location_type IN ('business_location', 'online', 'customer_address'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;
