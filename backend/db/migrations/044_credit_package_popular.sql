ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS is_popular BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE credit_packages
SET is_popular = TRUE,
    updated_at = NOW()
WHERE COALESCE(plan_key, '') = 'growth'
  AND COALESCE(billing_period, 'monthly') = 'monthly'
  AND is_active = TRUE;

WITH ranked AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY organization_id
      ORDER BY
        CASE WHEN COALESCE(plan_key, '') = 'growth' THEN 0 ELSE 1 END,
        price ASC,
        updated_at DESC
    ) AS popular_rank
  FROM credit_packages
  WHERE is_popular = TRUE
    AND COALESCE(billing_period, 'monthly') = 'monthly'
)
UPDATE credit_packages cp
SET is_popular = FALSE,
    updated_at = NOW()
FROM ranked
WHERE cp.id = ranked.id
  AND ranked.popular_rank > 1;

CREATE INDEX IF NOT EXISTS idx_credit_packages_org_popular
  ON credit_packages (organization_id, is_popular)
  WHERE is_popular = TRUE;
