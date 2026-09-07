ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS active_from TIMESTAMPTZ;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS active_until TIMESTAMPTZ;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS payment_method_changed_at TIMESTAMPTZ;

UPDATE credit_purchases p
SET active_from = COALESCE(p.active_from, p.confirmed_at, p.paid_at, p.created_at),
    active_until = COALESCE(p.active_until, COALESCE(p.confirmed_at, p.paid_at, p.created_at) + INTERVAL '1 month')
FROM credit_packages cp
WHERE cp.id = p.credit_package_id
  AND cp.organization_id = p.organization_id
  AND p.payment_status = 'confirmed'
  AND COALESCE(cp.billing_period, 'monthly') = 'monthly';

UPDATE credit_purchases p
SET active_from = COALESCE(p.active_from, p.confirmed_at, p.paid_at, p.created_at),
    active_until = COALESCE(
      p.active_until,
      (
        SELECT MAX(monthly.active_until)
        FROM credit_purchases monthly
        JOIN credit_packages monthly_package
          ON monthly_package.id = monthly.credit_package_id
         AND monthly_package.organization_id = monthly.organization_id
        WHERE monthly.organization_id = p.organization_id
          AND monthly.payment_status = 'confirmed'
          AND COALESCE(monthly_package.billing_period, 'monthly') = 'monthly'
      )
    )
FROM credit_packages cp
WHERE cp.id = p.credit_package_id
  AND cp.organization_id = p.organization_id
  AND p.payment_status = 'confirmed'
  AND COALESCE(cp.billing_period, 'monthly') = 'one_time';

CREATE INDEX IF NOT EXISTS idx_credit_purchases_org_active_period
  ON credit_purchases (organization_id, payment_status, active_until DESC);

CREATE TABLE IF NOT EXISTS credit_purchase_payment_orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  credit_purchase_id UUID NOT NULL REFERENCES credit_purchases(id) ON DELETE CASCADE,
  midtrans_order_id TEXT NOT NULL UNIQUE,
  payment_method TEXT NOT NULL,
  snap_token TEXT,
  snap_redirect_url TEXT,
  transaction_status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO credit_purchase_payment_orders (
  credit_purchase_id, midtrans_order_id, payment_method, snap_token, snap_redirect_url, transaction_status, created_at
)
SELECT id, midtrans_order_id, payment_method, snap_token, snap_redirect_url, COALESCE(transaction_status, payment_status), created_at
FROM credit_purchases
WHERE midtrans_order_id IS NOT NULL
ON CONFLICT (midtrans_order_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_credit_purchase_payment_orders_purchase
  ON credit_purchase_payment_orders (credit_purchase_id, created_at DESC);
