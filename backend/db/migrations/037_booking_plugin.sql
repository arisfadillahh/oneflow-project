CREATE TABLE IF NOT EXISTS booking_services (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  duration_minutes INTEGER NOT NULL DEFAULT 30,
  buffer_minutes INTEGER NOT NULL DEFAULT 0,
  price NUMERIC(12,2) NOT NULL DEFAULT 0,
  timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
  availability JSONB NOT NULL DEFAULT '{"days":[1,2,3,4,5],"start":"09:00","end":"17:00"}'::jsonb,
  status TEXT NOT NULL DEFAULT 'active',
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (duration_minutes > 0 AND duration_minutes <= 1440),
  CHECK (buffer_minutes >= 0 AND buffer_minutes <= 1440),
  CHECK (price >= 0),
  CHECK (status IN ('active', 'inactive', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_booking_services_org_name
  ON booking_services (organization_id, lower(name))
  WHERE status <> 'archived';

CREATE INDEX IF NOT EXISTS idx_booking_services_org_status
  ON booking_services (organization_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS booking_appointments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  service_id UUID NOT NULL REFERENCES booking_services(id) ON DELETE RESTRICT,
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
  customer_name TEXT,
  customer_phone TEXT,
  scheduled_start TIMESTAMPTZ NOT NULL,
  scheduled_end TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL DEFAULT 'scheduled',
  source TEXT NOT NULL DEFAULT 'dashboard',
  notes TEXT,
  created_by UUID REFERENCES agents(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (scheduled_end > scheduled_start),
  CHECK (status IN ('draft', 'scheduled', 'confirmed', 'cancelled', 'completed'))
);

CREATE INDEX IF NOT EXISTS idx_booking_appointments_org_start
  ON booking_appointments (organization_id, scheduled_start DESC);

CREATE INDEX IF NOT EXISTS idx_booking_appointments_org_status
  ON booking_appointments (organization_id, status, scheduled_start DESC);

CREATE INDEX IF NOT EXISTS idx_booking_appointments_service_start
  ON booking_appointments (organization_id, service_id, scheduled_start, scheduled_end)
  WHERE status IN ('scheduled', 'confirmed');
