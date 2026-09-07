BEGIN;
DO $$
DECLARE
  target_org UUID;
  fixture_package UUID;
BEGIN
  SELECT o.id INTO STRICT target_org
  FROM organizations o JOIN agents a ON a.id = o.created_by
  WHERE o.name = 'Oneflow QA Coexistence'
    AND a.username LIKE 'qa.coexistence.%'
    AND o.created_at > NOW() - INTERVAL '4 hours';

  IF EXISTS (SELECT 1 FROM credit_purchases WHERE organization_id=target_org AND notes='QA_COEXISTENCE_FIXTURE_NO_PAYMENT') THEN
    RAISE NOTICE 'QA fixture already exists';
    RETURN;
  END IF;
  INSERT INTO credit_packages (organization_id,name,credit_amount,price,is_active,plan_key,billing_period,max_whatsapp_sessions,max_ai_agents,max_human_users,description)
  VALUES (target_org,'QA Coexistence - No Payment',0,0,FALSE,'starter','monthly',1,1,1,'Temporary testing fixture; no charge, no credits added')
  RETURNING id INTO fixture_package;
  INSERT INTO credit_purchases (organization_id,credit_package_id,credit_amount,price,payment_method,payment_status,notes,confirmed_at,active_from,active_until,billing_period)
  VALUES (target_org,fixture_package,0,0,'qa_fixture','confirmed','QA_COEXISTENCE_FIXTURE_NO_PAYMENT',NOW(),NOW(),NOW()+INTERVAL '1 day','monthly');
END $$;
COMMIT;
