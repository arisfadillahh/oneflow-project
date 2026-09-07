ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS midtrans_order_id TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS midtrans_transaction_id TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS snap_token TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS snap_redirect_url TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS gross_amount NUMERIC(12, 2);
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS transaction_status TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS fraud_status TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS payment_type TEXT;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS expired_at TIMESTAMPTZ;
ALTER TABLE credit_purchases ADD COLUMN IF NOT EXISTS raw_notification JSONB;

CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_purchases_midtrans_order
  ON credit_purchases (midtrans_order_id)
  WHERE midtrans_order_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_credit_purchases_payment_status
  ON credit_purchases (payment_status, transaction_status);
