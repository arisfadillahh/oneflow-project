CREATE TABLE IF NOT EXISTS commerce_order_settings (
  organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
  enabled_fulfillment_types TEXT[] NOT NULL DEFAULT ARRAY['pickup', 'delivery', 'shipping', 'digital', 'onsite_service'],
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (
    cardinality(enabled_fulfillment_types) > 0
    AND enabled_fulfillment_types <@ ARRAY['pickup', 'delivery', 'shipping', 'digital', 'onsite_service']::TEXT[]
  )
);
