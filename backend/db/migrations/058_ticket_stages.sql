CREATE TABLE IF NOT EXISTS ticket_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'new',
  position INTEGER NOT NULL DEFAULT 0,
  color TEXT NOT NULL DEFAULT '#2196F3',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (length(BTRIM(name)) > 0),
  CHECK (status IN ('new', 'triage', 'waiting_customer', 'waiting_internal', 'in_progress', 'resolved', 'closed', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_stages_org_name
  ON ticket_stages (organization_id, lower(name));

CREATE INDEX IF NOT EXISTS idx_ticket_stages_org_position
  ON ticket_stages (organization_id, position);

ALTER TABLE tickets
  ADD COLUMN IF NOT EXISTS stage_id UUID REFERENCES ticket_stages(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tickets_org_stage_updated
  ON tickets (organization_id, stage_id, updated_at DESC);

WITH default_stages(name, status, position, color) AS (
  VALUES
    ('Baru', 'new', 10, '#2563EB'),
    ('Triage', 'triage', 20, '#7C3AED'),
    ('Diproses', 'in_progress', 30, '#0F766E'),
    ('Menunggu Customer', 'waiting_customer', 40, '#F59E0B'),
    ('Menunggu Internal', 'waiting_internal', 50, '#F97316'),
    ('Resolved', 'resolved', 60, '#16A34A'),
    ('Closed', 'closed', 70, '#64748B'),
    ('Batal', 'cancelled', 80, '#991B1B')
)
INSERT INTO ticket_stages (organization_id, name, status, position, color)
SELECT o.id, ds.name, ds.status, ds.position, ds.color
FROM organizations o
CROSS JOIN default_stages ds
WHERE NOT EXISTS (
  SELECT 1
  FROM ticket_stages existing
  WHERE existing.organization_id = o.id
);

UPDATE tickets t
SET stage_id = (
  SELECT ts.id
  FROM ticket_stages ts
  WHERE ts.organization_id = t.organization_id
    AND ts.status = t.status
  ORDER BY ts.position ASC, ts.created_at ASC
  LIMIT 1
)
WHERE t.stage_id IS NULL;
