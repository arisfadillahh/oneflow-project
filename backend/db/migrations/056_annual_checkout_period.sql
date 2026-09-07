ALTER TABLE credit_purchases
  ADD COLUMN IF NOT EXISTS billing_period TEXT;

UPDATE credit_purchases p
SET billing_period = COALESCE(cp.billing_period, 'monthly')
FROM credit_packages cp
WHERE cp.id = p.credit_package_id
  AND p.billing_period IS NULL;

UPDATE credit_purchases
SET billing_period = 'monthly'
WHERE billing_period IS NULL;

ALTER TABLE credit_purchases
  ALTER COLUMN billing_period SET DEFAULT 'monthly',
  ALTER COLUMN billing_period SET NOT NULL;

ALTER TABLE credit_purchases
  DROP CONSTRAINT IF EXISTS credit_purchases_billing_period_check;

ALTER TABLE credit_purchases
  ADD CONSTRAINT credit_purchases_billing_period_check
  CHECK (billing_period IN ('monthly', 'annual', 'one_time'));

CREATE INDEX IF NOT EXISTS idx_credit_purchases_org_period_status
  ON credit_purchases (organization_id, billing_period, payment_status, created_at DESC);
