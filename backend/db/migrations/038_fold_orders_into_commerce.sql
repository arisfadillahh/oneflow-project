-- Pesanan is now part of the commerce module, not a standalone installable tool.
INSERT INTO organization_modules (
  organization_id,
  module_key,
  is_enabled,
  enabled_at,
  enabled_by,
  metadata
)
SELECT
  organization_id,
  'commerce',
  TRUE,
  COALESCE(enabled_at, NOW()),
  enabled_by,
  COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('migratedFrom', 'orders')
FROM organization_modules
WHERE module_key = 'orders' AND is_enabled = TRUE
ON CONFLICT (organization_id, module_key) DO UPDATE
SET is_enabled = TRUE,
    enabled_at = COALESCE(organization_modules.enabled_at, EXCLUDED.enabled_at),
    enabled_by = COALESCE(organization_modules.enabled_by, EXCLUDED.enabled_by),
    disabled_at = NULL,
    disabled_by = NULL,
    metadata = COALESCE(organization_modules.metadata, '{}'::jsonb) || jsonb_build_object('ordersFoldedIntoCommerce', TRUE),
    updated_at = NOW();

UPDATE organization_modules
SET is_enabled = FALSE,
    disabled_at = COALESCE(disabled_at, NOW()),
    metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('foldedInto', 'commerce'),
    updated_at = NOW()
WHERE module_key = 'orders' AND is_enabled = TRUE;
