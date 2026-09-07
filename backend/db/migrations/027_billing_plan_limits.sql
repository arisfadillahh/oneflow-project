ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS plan_key TEXT NOT NULL DEFAULT '';
ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS billing_period TEXT NOT NULL DEFAULT 'monthly';
ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS max_whatsapp_sessions INTEGER NOT NULL DEFAULT 1;
ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS max_ai_agents INTEGER NOT NULL DEFAULT 1;
ALTER TABLE credit_packages ADD COLUMN IF NOT EXISTS max_human_users INTEGER NOT NULL DEFAULT 1;

UPDATE credit_pricing_settings
SET credit_unit_idr = 50
WHERE is_active = TRUE;

UPDATE credit_packages
SET is_active = FALSE,
    updated_at = NOW()
WHERE name = 'Starter Top Up'
  AND credit_amount = 5000
  AND price = 2500000;

WITH plans(plan_key, name, description, credit_amount, price, max_whatsapp_sessions, max_ai_agents, max_human_users) AS (
  VALUES
    ('starter', 'Starter', 'Untuk bisnis kecil yang mulai pakai CS AI.', 1000, 99000, 1, 2, 3),
    ('growth', 'Growth', 'Untuk tim operasional aktif dengan beberapa agent.', 3000, 299000, 2, 5, 10),
    ('business', 'Business', 'Untuk multi-brand atau volume chat tinggi.', 7000, 699000, 5, 15, 25)
)
INSERT INTO credit_packages (
  organization_id, plan_key, name, description, credit_amount, price,
  max_whatsapp_sessions, max_ai_agents, max_human_users, billing_period, is_active
)
SELECT
  o.id, p.plan_key, p.name, p.description, p.credit_amount, p.price,
  p.max_whatsapp_sessions, p.max_ai_agents, p.max_human_users, 'monthly', TRUE
FROM organizations o
CROSS JOIN plans p
WHERE NOT EXISTS (
  SELECT 1
  FROM credit_packages existing
  WHERE existing.organization_id = o.id
    AND existing.plan_key = p.plan_key
);

CREATE INDEX IF NOT EXISTS idx_credit_packages_org_plan ON credit_packages (organization_id, plan_key);
