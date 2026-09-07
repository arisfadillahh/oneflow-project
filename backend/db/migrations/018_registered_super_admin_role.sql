UPDATE agents
SET platform_role = 'platform_owner',
    updated_at = NOW()
WHERE role::text = 'owner'
  AND username = 'owner'
  AND platform_role = 'none';

UPDATE agents a
SET role = 'super_admin',
    updated_at = NOW()
WHERE a.role::text = 'owner'
  AND a.platform_role = 'none'
  AND EXISTS (
    SELECT 1
    FROM organizations o
    WHERE o.created_by = a.id
      AND o.slug <> 'default-company'
  );
