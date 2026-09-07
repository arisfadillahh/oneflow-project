WITH addons(plan_key, name, description, credit_amount, price) AS (
  VALUES
    ('addon_500', 'Top Up 500', 'Add-on 500 kredit ekstra. Tidak mengubah limit WhatsApp, AI agent, atau user.', 500, 25000),
    ('addon_1500', 'Top Up 1.500', 'Add-on 1.500 kredit ekstra untuk lonjakan chat sementara.', 1500, 75000),
    ('addon_5000', 'Top Up 5.000', 'Add-on 5.000 kredit ekstra untuk campaign atau periode ramai.', 5000, 250000)
)
INSERT INTO credit_packages (
  organization_id, plan_key, name, description, credit_amount, price,
  max_whatsapp_sessions, max_ai_agents, max_human_users, billing_period, is_active
)
SELECT
  o.id, a.plan_key, a.name, a.description, a.credit_amount, a.price,
  0, 0, 0, 'one_time', TRUE
FROM organizations o
CROSS JOIN addons a
WHERE NOT EXISTS (
  SELECT 1
  FROM credit_packages existing
  WHERE existing.organization_id = o.id
    AND existing.plan_key = a.plan_key
);
