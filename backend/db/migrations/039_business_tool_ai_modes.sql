ALTER TABLE organization_modules
  ADD COLUMN IF NOT EXISTS ai_mode TEXT NOT NULL DEFAULT 'off',
  ADD COLUMN IF NOT EXISTS ai_updated_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS ai_updated_by UUID REFERENCES agents(id) ON DELETE SET NULL;

UPDATE organization_modules
SET ai_mode = 'off'
WHERE ai_mode IS NULL OR ai_mode NOT IN ('off', 'read', 'draft', 'action');

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'organization_modules_ai_mode_check'
  ) THEN
    ALTER TABLE organization_modules
      ADD CONSTRAINT organization_modules_ai_mode_check
      CHECK (ai_mode IN ('off', 'read', 'draft', 'action'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_organization_modules_ai_mode
  ON organization_modules (organization_id, ai_mode)
  WHERE is_enabled = TRUE;
