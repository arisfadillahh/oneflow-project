ALTER TABLE credit_pricing_settings
  ADD COLUMN IF NOT EXISTS annual_discount_percent NUMERIC(5,2) NOT NULL DEFAULT 10;

ALTER TABLE credit_pricing_settings
  DROP CONSTRAINT IF EXISTS credit_pricing_settings_annual_discount_percent_check;

ALTER TABLE credit_pricing_settings
  ADD CONSTRAINT credit_pricing_settings_annual_discount_percent_check
  CHECK (annual_discount_percent >= 0 AND annual_discount_percent < 100);

UPDATE credit_pricing_settings
SET annual_discount_percent = 10
WHERE annual_discount_percent IS NULL;
